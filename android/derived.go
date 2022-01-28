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
	"go.starlark.net/starlarkstruct"
)

var constants = make(map[string]*string)
var GARBAGE = reflect.Value{}

// RegisterConstant registers constants to be made available to Starlark
// interpreter that evaluates the derived() function
// Cf bp2build.bazelConstant
func RegisterConstant(key string, value *string) {
	if constants[key] != nil {
		panic(fmt.Errorf("already registered %s = %s", key, *constants[key]))
	}
	constants[key] = value
}

func RegisterDerivedPropertiesPreArchMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("derived_properties", derivedPropertiesMutator).Parallel()
}

func makeCtx(ctx BaseMutatorContext) *starlarkstruct.Struct {
	env := starlark.StringDict{
		"get": starlark.NewBuiltin("get",
			func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				var key string
				if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key); err != nil {
					return nil, err
				}
				value := ctx.Config().Getenv(key)
				if value == "" {
					return starlark.None, nil
				}
				return starlark.String(value), nil
			}),
		"is_true": starlark.NewBuiltin("is_true",
			func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				var key string
				if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key); err != nil {
					return nil, err
				}
				return starlark.Bool(ctx.Config().IsEnvTrue(key)), nil
			}),
		"sanitize_host": starlark.NewBuiltin("sanitize_host",
			func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				var values []starlark.Value
				for _, host := range ctx.Config().SanitizeHost() {
					values = append(values, starlark.String(host))
				}
				return starlark.NewList(values), nil
			}),
	}
	device := starlark.StringDict{
		"bt_config_include_dir": starlark.NewBuiltin("bt_config_include_dir",
			func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				return starlark.String(ctx.Config().deviceConfig.BtConfigIncludeDir()), nil
			}),
	}
	consts := starlark.StringDict{}
	for k, v := range constants {
		consts[k] = starlark.String(*v)
	}
	ctxStarlark := starlarkstruct.FromStringDict(starlark.None, starlark.StringDict{
		"module_name": starlark.String(ctx.ModuleName()), // moduleName can be changes by other mutators - caution using this
		"env":         starlarkstruct.FromStringDict(starlark.None, env),
		"device":      starlarkstruct.FromStringDict(starlark.None, device),
		"constants":   starlarkstruct.FromStringDict(starlark.None, consts),
	})
	ctxStarlark.Freeze()
	return ctxStarlark
}

func withLineNums(in *string) string {
	out := ""
	for i, s := range strings.Split(*in, "\n") {
		out += fmt.Sprintf("%3d: %s\n", i+1, s)
	}
	return out
}

//The string associated with derivedProperties is a Starlark function body.
//It should return a map (possibly nested) that provides values for Module.GetProperties()
func evaluateDerivations(ctx BaseMutatorContext, starlarkScript *string) *starlark.Dict {
	thread := &starlark.Thread{Name: "starlark"}
	globals, err := starlark.ExecFile(thread, ctx.BlueprintsFile(), *starlarkScript, nil)
	if err != nil {
		ctx.PropertyErrorf("derived", "%e\n%s", err, withLineNums(starlarkScript))
		return nil
	}
	value, hasValue := globals["derive"]
	if !hasValue {
		ctx.PropertyErrorf("derived", "missing `def derive()`\n%s", withLineNums(starlarkScript))
		return nil
	}
	fn, isFunc := value.(*starlark.Function)
	if !isFunc {
		ctx.PropertyErrorf("derived", "`%T` but expected a `function`\n%s", value, withLineNums(starlarkScript))
		return nil
	}

	if fn.NumParams() > 1 {
		ctx.PropertyErrorf("derived", "expected a single parameter\n%s", withLineNums(starlarkScript))
		return nil
	}
	var kwargs []starlark.Tuple
	if fn.NumParams() == 1 {
		name, _ := fn.Param(0)
		kwargs = append(kwargs, starlark.Tuple{starlark.String(name), makeCtx(ctx)})
	}
	value, err = starlark.Call(thread, fn, nil, kwargs)
	if err != nil {
		ctx.PropertyErrorf("derived", "%e\n%s", err, withLineNums(starlarkScript))
		return nil
	}
	derivations, isDict := value.(*starlark.Dict)
	if !isDict {
		ctx.PropertyErrorf("derived", "%s\n%s", value, withLineNums(starlarkScript))
		return nil
	}
	return derivations
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

func apply(aStruct reflect.Value, rootStructName string, parentPath string, key string, value starlark.Value) (map[string]struct{}, error) {
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
			return nil, fmt.Errorf("%s is not settable", pathWithRootStruct)
		}
		if proptools.HasTag(structType.Field(i), "blueprint", "mutated") {
			return nil, fmt.Errorf("%s is tagged blueprint:\"mutated\"", pathWithRootStruct)
		}

		anchor := fieldValue
		switch fieldValue.Kind() {
		case reflect.Interface:
			//run-time defined structs
			fieldValue = fieldValue.Elem()
			if fieldValue.Kind() != reflect.Ptr {
				return nil, fmt.Errorf("%s is %s instead of a pointer", pathWithRootStruct, fieldValue.Kind())
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
				v, e := apply(fieldValue, rootStructName, path, subKey, pair[1])
				if e != nil {
					return nil, e
				}
				addAll(used, v)
			}
		default:
			if !fieldValue.IsZero() {
				return nil, fmt.Errorf("%s is already set", pathWithRootStruct)
			}
			v, e := asValue(value, fieldValue.Type())
			if e != nil {
				return nil, e
			}
			proptools.ExtendBasicType(fieldValue, v, proptools.Append)
			used[path] = struct{}{}
		}
	}
	return used, nil
}

// TODO @usta parallelize
func applyRoot(pRootStruct interface{}, moduleName string, derivations *starlark.Dict) (map[string]struct{}, error) {
	//the root struct is guaranteed to be a pointer to a struct
	rootStruct := reflect.ValueOf(pRootStruct).Elem()
	rootName := rootStruct.Type().Name()
	var used map[string]struct{}
	for _, derivation := range derivations.Items() {
		key := derivation[0].(starlark.String).GoString()
		value := derivation[1]
		subTreeUsed, e := apply(rootStruct, rootName, moduleName, key, value)
		if e != nil {
			return nil, e
		}
		if used == nil {
			used = subTreeUsed
		} else {
			addAll(used, subTreeUsed)
		}
	}
	return used, nil
}

func derivedPropertiesMutator(ctx BottomUpMutatorContext) {
	starlarkScript := ctx.Module().base().derivedProperties.Derived
	if starlarkScript == nil {
		return
	}

	setToString := func(s map[string]struct{}) string {
		keys := make([]string, len(s))
		i := 0
		for k := range s {
			keys[i] = k
			i++
		}
		sort.Strings(keys) //for determinism
		return strings.Join(keys, ", ")
	}

	derivations := evaluateDerivations(ctx, starlarkScript)
	if derivations != nil {
		unused := getPaths(ctx.ModuleName(), derivations)
		for _, pRootStruct := range ctx.Module().GetProperties() {
			if _, isDerived := pRootStruct.(*derivedProperties); isDerived {
				continue
			}
			rootUsed, e := applyRoot(pRootStruct, ctx.ModuleName(), derivations)
			if e != nil {
				ctx.PropertyErrorf("derived", e.Error())
			} else {
				removeAll(unused, rootUsed)
			}
		}
		if len(unused) > 0 {
			ctx.PropertyErrorf("derived", "unused %q:\n%s", setToString(unused), derivations)
		}
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

func asValue(starlarkValue starlark.Value, t reflect.Type) (reflect.Value, error) {
	primitive := func(starlarkValue starlark.Value) (interface{}, error) {
		boolVal, ok := starlarkValue.(starlark.Bool)
		if ok {
			return bool(boolVal), nil
		}
		stringVal, ok := starlarkValue.(starlark.String)
		if ok {
			return stringVal.GoString(), nil
		}
		intVal, ok := starlarkValue.(starlark.Int).Int64()
		if ok {
			return intVal, nil
		}
		return nil, fmt.Errorf("not a primitive %s", starlarkValue)
	}
	switch t.Kind() {
	case reflect.Bool, reflect.String, reflect.Int64:
		value, e := primitive(starlarkValue)
		if e != nil {
			return GARBAGE, e
		}
		return reflect.ValueOf(value), nil
	case reflect.Slice:
		l := starlarkValue.(*starlark.List)
		slice := reflect.MakeSlice(t, 0, l.Len())
		for i := 0; i < l.Len(); i++ {
			value, e := asValue(l.Index(i), t.Elem())
			if e != nil {
				return GARBAGE, e
			}
			fmt.Print(slice.Interface())
			slice = reflect.Append(slice, value)
		}
		return slice, nil
	case reflect.Ptr:
		value, e := primitive(starlarkValue)
		if e != nil {
			return GARBAGE, e
		}
		switch t.Elem().Kind() {
		case reflect.Bool:
			return reflect.ValueOf(proptools.BoolPtr(value.(bool))), nil
		case reflect.String:
			return reflect.ValueOf(proptools.StringPtr(value.(string))), nil
		case reflect.Int64:
			return reflect.ValueOf(proptools.Int64Ptr(value.(int64))), nil
		}
		fallthrough
	default:
		return GARBAGE, fmt.Errorf("unsupported %s for %v at %s", starlarkValue, t, "todo pass attribute path for info")
	}
}
