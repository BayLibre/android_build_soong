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

type pathSet map[string]bool

var constants = make(map[string]*string)
var garbage = reflect.Value{}

// RegisterConstant registers constants to be made available to Starlark
// interpreter that evaluates the `compute()` function
// Cf bp2build.bazelConstant
func RegisterConstant(key string, value *string) {
	if constants[key] != nil {
		panic(fmt.Errorf("already registered %s = %s", key, *constants[key]))
	}
	constants[key] = value
}

func RegisterDerivedPropertiesPreArchMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("computed_properties", computedPropertiesMutator).Parallel()
}

func computedPropertiesMutator(ctx BottomUpMutatorContext) {
	starlarkScript := ctx.Module().base().computedProperties.Computed
	if starlarkScript == nil {
		return
	}

	reportError := func() func(fmt string, args ...interface{}) {
		withLineNums := ""
		for i, s := range strings.Split(*starlarkScript, "\n") {
			withLineNums += fmt.Sprintf("%3d: %s\n", i+1, s)
		}
		return func(fmt string, args ...interface{}) {
			ctx.PropertyErrorf("computed", fmt+"\n%s", append(args, withLineNums)...)
		}
	}()

	contextInStarlark := contextInStarlark(ctx)
	contextInStarlark.Freeze()

	// starlarkScript MUST have define a `compute()` function, which should
	// evaluate to a dict(possibly nested) or a string in case of error
	allComputations := func() *starlark.Dict {
		thread := &starlark.Thread{Name: "starlark-" + ctx.ModuleName()}
		globals, err := starlark.ExecFile(thread, ctx.BlueprintsFile(), *starlarkScript, nil)
		if err != nil {
			reportError(err.Error())
			return nil
		}
		value, hasValue := globals["compute"]
		if !hasValue {
			reportError("missing `def compute(ctx)`")
			return nil
		}
		compute, isFunc := value.(*starlark.Function)
		if !isFunc {
			reportError("`%T` but expected a `function`", value)
			return nil
		}

		if compute.NumParams() > 1 {
			reportError("expected a single parameter")
			return nil
		}
		var kwargs []starlark.Tuple
		if compute.NumParams() == 1 {
			name, _ := compute.Param(0)
			kwargs = append(kwargs, starlark.Tuple{starlark.String(name), contextInStarlark})
		}
		value, err = starlark.Call(thread, compute, nil, kwargs)
		if err != nil {
			reportError(err.Error())
			return nil
		}
		computations, isDict := value.(*starlark.Dict)
		if !isDict {
			reportError(value.String())
			return nil
		}
		return computations
	}()
	if allComputations == nil {
		return
	}
	allComputations.Freeze()

	// flattened list of properties that have computed values
	// the format of each entry: {moduleName}.{property}.{property}...
	computedPaths := func() pathSet {
		accumulator := make(pathSet)
		var helper func(parentPath string, dict *starlark.Dict)
		helper = func(parentPath string, dict *starlark.Dict) {
			for _, computation := range dict.Items() {
				key := computation[0].(starlark.String).GoString()
				currentPath := fmt.Sprintf("%s.%s", parentPath, key)
				if subDict, ok := computation[1].(*starlark.Dict); ok {
					helper(currentPath, subDict)
				} else {
					accumulator[currentPath] = false
				}
			}
		}
		helper(ctx.ModuleName(), allComputations)
		return accumulator
	}()

	var apply func(targetStruct reflect.Value, pathPrefix string, key string, value starlark.Value) error
	apply = func(targetStruct reflect.Value, pathPrefix string, key string, value starlark.Value) error {
		path := fmt.Sprintf("%s.%s", pathPrefix, key)
		structType := targetStruct.Type()
		for i := 0; i < targetStruct.NumField(); i++ {
			field := structType.Field(i)
			if !field.IsExported() || proptools.FieldNameForProperty(key) != field.Name {
				continue
			}

			fieldValue := targetStruct.Field(i)
			if !fieldValue.CanSet() {
				return fmt.Errorf("%s is not settable", path)
			}
			if proptools.HasTag(structType.Field(i), "blueprint", "mutated") {
				return fmt.Errorf("%s is tagged blueprint:\"mutated\"", path)
			}

			anchor := fieldValue
			switch fieldValue.Kind() {
			case reflect.Interface:
				//run-time defined structs
				fieldValue = fieldValue.Elem()
				if fieldValue.Kind() != reflect.Ptr {
					return fmt.Errorf("%s is %s instead of a pointer", path, fieldValue.Kind())
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
					e := apply(fieldValue, path, subKey, pair[1])
					if e != nil {
						return e
					}
				}
			default:
				if !fieldValue.IsZero() {
					return fmt.Errorf("%s is already set", path)
				}
				v, e := convertStarlarkToGo(value, fieldValue.Type())
				if e != nil {
					return e
				}
				proptools.ExtendBasicType(fieldValue, v, proptools.Append)
				computedPaths[path] = true
			}
		}
		return nil
	}
	// TODO @usta parallelize
	for _, pRootStruct := range ctx.Module().GetProperties() {
		if _, hasComputedProperties := pRootStruct.(*computedProperties); hasComputedProperties {
			continue
		}
		//the root struct is guaranteed to be a pointer to a struct
		rootStruct := reflect.ValueOf(pRootStruct).Elem()
		for _, computation := range allComputations.Items() {
			key := computation[0].(starlark.String).GoString()
			value := computation[1]
			err := apply(rootStruct, ctx.ModuleName(), key, value)
			if err != nil {
				reportError(err.Error())
			}
		}
	}

	unusedPaths := getUnusdPaths(computedPaths)
	if len(unusedPaths) > 0 {
		reportError("unused:\n\t%s", strings.Join(unusedPaths, "\n\t"))
	}
}

func convertGoToStarlark(goValue reflect.Value) (starlark.Value, error) {
	switch goValue.Kind() {
	case reflect.Ptr:
		return convertGoToStarlark(goValue.Elem())
	case reflect.String:
		return starlark.String(goValue.Interface().(string)), nil
	case reflect.Slice:
		var slice []starlark.Value
		for i := 0; i < goValue.Len(); i++ {
			elem, err := convertGoToStarlark(goValue.Index(i))
			if err != nil {
				return starlark.None, err
			}
			slice = append(slice, elem)
		}
		return starlark.NewList(slice), nil
	default:
		return starlark.None, fmt.Errorf("unsupported type %T: %s", goValue, goValue)
	}
}

func getStructField(pSourceStruct interface{}, fieldName string) (starlark.Value, error) {
	r := reflect.ValueOf(pSourceStruct).Elem()
	field, has := r.Type().FieldByName(fieldName)
	if has && field.IsExported() {
		return convertGoToStarlark(r.FieldByName(fieldName))
	}
	return starlark.None, fmt.Errorf("no such field: %s", fieldName)
}

func callMethod(pSourceStruct interface{}, methodName string) (starlark.Value, error) {
	r := reflect.ValueOf(pSourceStruct).Elem()
	method, has := r.Type().MethodByName(methodName)
	if has && method.IsExported() {
		return convertGoToStarlark(r.MethodByName(methodName))
	}
	return starlark.None, fmt.Errorf("no such method: %s", methodName)
}

func contextInStarlark(ctx BaseMutatorContext) *starlarkstruct.Struct {
	consts := starlark.StringDict{}
	for k, v := range constants {
		consts[k] = starlark.String(*v)
	}
	//TODO(usta) whitelist what's accessible in Starlark?
	device := starlark.StringDict{
		"getvar": starlark.NewBuiltin("getvar",
			func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				var key string
				if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key); err != nil {
					return nil, err
				}
				return callMethod(ctx.DeviceConfig().deviceConfig, key)
			}),
	}
	//TODO(usta) whitelist what's accessible in Starlark?
	env := starlark.StringDict{
		"getvar": starlark.NewBuiltin("getvar",
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
	}
	//TODO(usta) whitelist what's accessible in Starlark?
	product := starlark.StringDict{
		"getvar": starlark.NewBuiltin("getvar",
			func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				var key string
				if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key); err != nil {
					return nil, err
				}
				p := ctx.Config().productVariables
				return getStructField(&p, key)
			}),
	}
	return starlarkstruct.FromStringDict(starlark.None, starlark.StringDict{
		"module_name":       starlark.String(ctx.ModuleName()), // moduleName can be changes by other mutators - caution using this
		"constants":         starlarkstruct.FromStringDict(starlark.None, consts),
		"device":            starlarkstruct.FromStringDict(starlark.None, device),
		"env":               starlarkstruct.FromStringDict(starlark.None, env),
		"product_variables": starlarkstruct.FromStringDict(starlark.None, product),
	})
}

func convertStarlarkToGo(starlarkValue starlark.Value, t reflect.Type) (reflect.Value, error) {
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
			return garbage, e
		}
		return reflect.ValueOf(value), nil
	case reflect.Slice:
		l := starlarkValue.(*starlark.List)
		slice := reflect.MakeSlice(t, 0, l.Len())
		for i := 0; i < l.Len(); i++ {
			value, e := convertStarlarkToGo(l.Index(i), t.Elem())
			if e != nil {
				return garbage, e
			}
			fmt.Print(slice.Interface())
			slice = reflect.Append(slice, value)
		}
		return slice, nil
	case reflect.Ptr:
		value, e := primitive(starlarkValue)
		if e != nil {
			return garbage, e
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
		return garbage, fmt.Errorf("unsupported %s for %v at %s", starlarkValue, t, "todo pass attribute path for info")
	}
}

func getUnusdPaths(paths pathSet) []string {
	var unusedPaths []string
	for path, used := range paths {
		if !used {
			unusedPaths = append(unusedPaths, path)
		}
	}
	sort.Strings(unusedPaths) //for determinism
	return unusedPaths
}
