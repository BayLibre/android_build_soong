// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"fmt"
	"reflect"

	"github.com/google/blueprint/proptools"
	"go.starlark.net/starlark"
)

func RegisterDerivedPropertiesPreArchMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("derived_properties", derivedPropertiesMutator).Parallel()
}

//TODO @usta: consult with @jingwen @cparsons to leverage cquery syntax
func apiForStarlark(ctx BaseMutatorContext) starlark.StringDict {
	return starlark.StringDict{
		"Getenv": starlark.NewBuiltin("Getenv", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var key string
			if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key); err != nil {
				return nil, err
			}
			return starlark.String(ctx.Config().Getenv(key)), nil
		}),
	}
}

//The string associated with derivedProperties is a Starlark function body.
//It should return a map (possibly nested) that provides values for Module.GetProperties()
func evaluateDerivations(ctx BaseMutatorContext, starlarkScript *string) *starlark.Dict {
	thread := &starlark.Thread{Name: "starklark"}
	api := apiForStarlark(ctx)
	globals, err := starlark.ExecFile(thread, ctx.BlueprintsFile(), *starlarkScript, api)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate \"derived\" properties for %s in %s %q", ctx.ModuleName(), ctx.BlueprintsFile(), err))
	}
	value, ok := globals["value"]
	if !ok {
		value, err = starlark.ExprFunc(ctx.BlueprintsFile(), *starlarkScript, nil)
	}
	if err != nil {
		panic(fmt.Errorf("failed to evaluate \"derived\" properties for %s in %s %q", ctx.ModuleName(), ctx.BlueprintsFile(), err))
	}
	_, isFunction := value.(*starlark.Function)
	if isFunction {
		value, err = starlark.Call(thread, value, nil, nil)
	}
	if err != nil {
		panic(fmt.Errorf("failed to evaluate \"derived\" properties for %s in %s %q", ctx.ModuleName(), ctx.BlueprintsFile(), err))
	}
	derivations, isDict := value.(*starlark.Dict)
	if !isDict {
		panic(fmt.Errorf("expected a starlark.Dict but found %q for %s in %s\n%s", value.Type(), ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript))
	}
	return derivations
}

// TODO @usta parallelize
func use(ctx BaseMutatorContext, key string, value starlark.Value) (used bool) {
	for _, p := range ctx.Module().GetProperties() {
		if _, isDerived := p.(*derivedProperties); isDerived {
			continue
		}
		used = used || apply(p, fmt.Sprintf("%s/", ctx.ModuleName()), key, value)
	}
	return
}

func apply(p interface{}, pathForLogging string, key string, value starlark.Value) (used bool) {
	propStruct := reflect.ValueOf(p).Elem() //pointer to struct
	propType := propStruct.Type()
	for i := 0; i < propStruct.NumField(); i++ {
		field := propType.Field(i)
		if proptools.FieldNameForProperty(key) != field.Name {
			continue
		}
		fieldValue := propStruct.Field(i)
		if !fieldValue.CanSet() {
			panic(fmt.Errorf("field %s.%s is not settable", pathForLogging, field.Name))
		}
		if proptools.HasTag(field, "blueprint", "mutated") {
			panic(fmt.Errorf(`%q is marked blueprint:"mutated"`, field.Name))
		}
		if field.Type.Kind() == reflect.Struct {
			panic("TODO recursive decent")
		}
		if !fieldValue.IsZero() {
			if fieldValue.Type().Kind() == reflect.Ptr {
				fieldValue = fieldValue.Elem()
			}
			panic(fmt.Errorf("already set %q = %#v", field.Name, fieldValue))
		}
		proptools.ExtendBasicType(fieldValue, asValue(value, field.Type), proptools.Append)
		used = true
	}
	return
}

func derivedPropertiesMutator(ctx BottomUpMutatorContext) {
	starlarkScript := ctx.Module().base().derivedProperties.Derived
	if starlarkScript == nil {
		return
	}

	derivations := evaluateDerivations(ctx, starlarkScript)

	for _, derivation := range derivations.Items() {
		key := derivation[0].(starlark.String).GoString()
		value := derivation[1].(starlark.Value)
		if !use(ctx, key, value) {
			panic(fmt.Errorf("no such property %s to set the value: %v", key, value))
		}
	}
}

func asValue(starlarkValue starlark.Value, t reflect.Type) reflect.Value {
	primitive := func(starlarkValue starlark.Value) interface{} {
		boolVal, ok := starlarkValue.(starlark.Bool)
		if ok {
			return bool(boolVal)
		}
		stringVal, ok := starlarkValue.(starlark.String)
		if ok {
			return stringVal.GoString()
		}
		intVal, ok := starlarkValue.(starlark.Int).Int64()
		if ok {
			return intVal
		}
		panic(fmt.Errorf("not a primitive %v", starlarkValue))
	}
	switch t.Kind() {
	case reflect.Bool, reflect.String, reflect.Int64:
		return reflect.ValueOf(primitive(starlarkValue))
	case reflect.Slice:
		l := starlarkValue.(*starlark.List)
		slice := reflect.MakeSlice(t, 0, l.Len())
		for i := 0; i < l.Len(); i++ {
			slice = reflect.Append(slice, asValue(l.Index(i), t.Elem()))
		}
		return slice
	case reflect.Ptr:
		var value = primitive(starlarkValue)
		switch t.Elem().Kind() {
		case reflect.Bool:
			return reflect.ValueOf(proptools.BoolPtr(value.(bool)))
		case reflect.String:
			return reflect.ValueOf(proptools.StringPtr(value.(string)))
		case reflect.Int64:
			return reflect.ValueOf(proptools.Int64Ptr(value.(int64)))
		}
		fallthrough
	default:
		panic(fmt.Errorf("unsupported %T for %v at %s", starlarkValue, t, "todo pass attribute path for info"))
	}
}
