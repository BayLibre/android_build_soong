// Copyright 2022 Google Inc. All rights reserved.
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
	"sort"
	"strings"

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
		"IsEnvTrue": starlark.NewBuiltin("IsEnvTrue", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var key string
			if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key); err != nil {
				return nil, err
			}
			return starlark.Bool(ctx.Config().IsEnvTrue(key)), nil
		}),
		"SanitizeHost": starlark.NewBuiltin("SanitizeHost", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var values []starlark.Value
			for _, host := range ctx.Config().SanitizeHost() {
				values = append(values, starlark.String(host))
			}
			return starlark.NewList(values), nil
		}),
	}
}

//The string associated with derivedProperties is a Starlark function body.
//It should return a map (possibly nested) that provides values for Module.GetProperties()
func evaluateDerivations(ctx BaseMutatorContext, starlarkScript *string) *starlark.Dict {
	api := apiForStarlark(ctx)
	thread := &starlark.Thread{Name: "starlark"}
	globals, err := starlark.ExecFile(thread, ctx.BlueprintsFile(), *starlarkScript, api)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate \"%s.derived\" in %s\n%s\n%w",
			ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript, err))
	}
	value, hasValue := globals["derive"]
	if !hasValue {
		panic(fmt.Errorf("failed to evaluate \"%s.derived\" in %s\n%s\ndefine a `def derive() -> dict[string, ?]`",
			ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript))
	}
	if _, isFunc := value.(*starlark.Function); !isFunc {
		panic(fmt.Errorf("failed to evaluate \"%s.derived\" in %s\n%s\ndefine a `def value() -> dict[string, ?]`",
			ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript))
	}
	if len(globals) > 1 {
		panic(fmt.Errorf("expected only `def derive()` in \"%s.derived\" in %s but found [%s]\n%s\n%w",
			ctx.ModuleName(), ctx.BlueprintsFile(), strings.Join(globals.Keys(), ", "), *starlarkScript, err))
	}
	value, err = starlark.Call(thread, value, nil, nil)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate \"%s.derived\" in %s\n%s\n%w",
			ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript, err))
	}
	derivations, isDict := value.(*starlark.Dict)
	if !isDict {
		panic(fmt.Errorf("expected \"%s.derived\" in %s to evaluate to a starlark dict but got %T\n%s\n%w",
			ctx.ModuleName(), ctx.BlueprintsFile(), value, *starlarkScript, err))
	}
	return derivations
}

func setToString(s map[string]struct{}) string {
	keys := make([]string, len(s))
	i := 0
	for k := range s {
		keys[i] = k
		i++
	}
	sort.Strings(keys) //for determinism
	return strings.Join(keys, ", ")
}

func addAll(dest map[string]struct{}, source map[string]struct{}) {
	if source != nil {
		for k, v := range source {
			dest[k] = v
		}
	}
}

func removeAll(s map[string]struct{}, other map[string]struct{}) {
	if other != nil {
		for k := range other {
			delete(s, k)
		}
	}
}

func apply(aStruct reflect.Value, rootStructName string, parentPath string, key string, value starlark.Value) map[string]struct{} {
	structType := aStruct.Type()
	var used = make(map[string]struct{})
	path := fmt.Sprintf("%s.%s", parentPath, key)
	pathWithRootStruct := strings.Replace(path, ".", fmt.Sprintf("[%s].", rootStructName), 1)
	isExported := func(f reflect.StructField) bool {
		return f.PkgPath == ""
	}
	for i := 0; i < aStruct.NumField(); i++ {
		field := structType.Field(i)
		if !isExported(field) || proptools.FieldNameForProperty(key) != field.Name {
			continue
		}

		fieldValue := aStruct.Field(i)
		if !fieldValue.CanSet() {
			panic(fmt.Errorf("%s is not settable", pathWithRootStruct))
		}
		if proptools.HasTag(structType.Field(i), "blueprint", "mutated") {
			panic(fmt.Errorf("%s is tagged blueprint:\"mutated\"", pathWithRootStruct))
		}

		anchor := fieldValue
		switch fieldValue.Kind() {
		case reflect.Interface:
			//run-time defined structs
			fieldValue = fieldValue.Elem()
			if fieldValue.Kind() != reflect.Ptr {
				panic(fmt.Errorf("%s is %s instead of a pointer", pathWithRootStruct, fieldValue.Kind()))
			}
			fallthrough
		case reflect.Ptr:
			//struct pointer to be initialized to point to a "zero" struct
			if fieldValue.Type().Elem().Kind() == reflect.Struct {
				if fieldValue.IsNil() {
					fieldValue = reflect.New(fieldValue.Type().Elem())
					anchor.Set(fieldValue)
				}
				fieldValue = fieldValue.Elem()
			}
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			for _, pair := range value.(*starlark.Dict).Items() {
				subKey := pair[0].(starlark.String).GoString()
				addAll(used, apply(fieldValue, rootStructName, path, subKey, pair[1]))
			}
		default:
			if !fieldValue.IsZero() {
				panic(fmt.Errorf("%s is already set", pathWithRootStruct))
			}
			proptools.ExtendBasicType(fieldValue, asValue(value, fieldValue.Type()), proptools.Append)
			used[path] = struct{}{}
		}
	}
	return used
}

// TODO @usta parallelize
func applyRoot(pRootStruct interface{}, moduleName string, derivations *starlark.Dict) map[string]struct{} {
	//the root struct is guaranteed to be a pointer to a struct
	rootStruct := reflect.ValueOf(pRootStruct).Elem()
	rootName := rootStruct.Type().Name()
	var used map[string]struct{}
	for _, derivation := range derivations.Items() {
		key := derivation[0].(starlark.String).GoString()
		value := derivation[1]
		subTreeUsed := apply(rootStruct, rootName, moduleName, key, value)
		if used == nil {
			used = subTreeUsed
		} else {
			addAll(used, subTreeUsed)
		}
	}
	return used
}

func derivedPropertiesMutator(ctx BottomUpMutatorContext) {
	starlarkScript := ctx.Module().base().derivedProperties.Derived
	if starlarkScript == nil {
		return
	}

	derivations := evaluateDerivations(ctx, starlarkScript)
	unused := getPaths(ctx.ModuleName(), derivations)
	for _, pRootStruct := range ctx.Module().GetProperties() {
		if _, isDerived := pRootStruct.(*derivedProperties); isDerived {
			continue
		}
		rootUsed := applyRoot(pRootStruct, ctx.ModuleName(), derivations)
		removeAll(unused, rootUsed)
	}
	if len(unused) > 0 {
		panic(fmt.Errorf("%s unused %q:\n%s", ctx.BlueprintsFile(), setToString(unused), derivations.String()))
	}
}

func getPaths(parentPath string, dict *starlark.Dict) map[string]struct{} {
	paths := make(map[string]struct{})
	for _, derivation := range dict.Items() {
		key := derivation[0].(starlark.String).GoString()
		currentPath := fmt.Sprintf("%s.%s", parentPath, key)
		value := derivation[1] //.(starlark.Value)
		if subDict, ok := value.(*starlark.Dict); ok {
			subpaths := getPaths(currentPath, subDict)
			addAll(paths, subpaths)
		} else {
			paths[currentPath] = struct{}{}
		}
	}
	return paths
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
		panic(fmt.Errorf("not a primitive %s", starlarkValue))
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
		panic(fmt.Errorf("unsupported %s for %v at %s", starlarkValue, t, "todo pass attribute path for info"))
	}
}
