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
	"strings"

	"github.com/google/blueprint/proptools"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var constantsAvailableInStarlark = make(map[string]*string)

//used as an information-less marker value in stringSet
var starlarkComputedPropertyMarker = struct{}{}

//the value is immaterial and will always be set to starlarkComputedPropertyMarker
type stringSet map[string]struct{}

// RegisterConstantForStarlark registers constants to be made available to Starlark
// interpreter that evaluates the `compute()` function
// Cf bp2build.bazelConstant
func RegisterConstantForStarlark(key string, value *string) {
	existingValue := constantsAvailableInStarlark[key]
	if existingValue != nil && *existingValue != *value {
		panic(fmt.Errorf("already registered %s = %s", key, *existingValue))
	}
	constantsAvailableInStarlark[key] = value
}

func RegisterComputedPropertiesPreArchMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("computed_properties", computedPropertiesMutator).Parallel()
}

type starlarkErrorReporterFn = func(fmt string, args ...interface{})

func computedPropertiesMutator(ctx BottomUpMutatorContext) {
	starlarkScript := ctx.Module().base().computedProperties.Computed
	if starlarkScript == nil {
		return
	}

	allComputedProperties := runStarlarkPropertyComputation(ctx, starlarkScript)
	if allComputedProperties == nil {
		return
	}

	reportError := generateStarlarkErrorReporter(ctx, starlarkScript)
	allAppliedLeaves := filterAppliedStarlarkComputedProperties(ctx, reportError, allComputedProperties)

	visitAllComputedProperties(ctx.ModuleName(), allComputedProperties, func(path string) {
		if _, applied := allAppliedLeaves[path]; !applied {
			reportError("%s does not exist", path)
		}
	})
}

// filterAppliedStarlarkComputedProperties finds all the properties in
// allComputedProperties that were actually assigned to the module
func filterAppliedStarlarkComputedProperties(
	ctx EarlyModuleContext,
	reportError starlarkErrorReporterFn,
	allComputedProperties *starlark.Dict,
) stringSet {
	allAppliedLeaves := make(stringSet)
	// TODO @usta parallelize ?
	for _, computedProperty := range allComputedProperties.Items() {
		for _, pRootStruct := range ctx.Module().GetProperties() {
			if _, hasComputedProperties := pRootStruct.(*computedProperties); hasComputedProperties {
				continue
			}
			// We are guaranteed to reach here for some `pRootStruct` because
			// `computed` must have come from one.
			rootStruct := reflect.ValueOf(pRootStruct).Elem() // <== ptr dereference
			appliedLeaves := applyComputedProperty(reportError, ctx.ModuleName(), rootStruct, computedProperty)
			for applied := range appliedLeaves {
				allAppliedLeaves[applied] = starlarkComputedPropertyMarker
			}
		}
	}
	return allAppliedLeaves
}

// visitAllComputedProperties calls `visit(path)` if path is a leaf,
// i.e. `value` is NOT a `dict` otherwise it recursively descends intothe children paths
func visitAllComputedProperties(path string, value starlark.Value, visit func(path string)) {
	if dict, isDict := value.(*starlark.Dict); isDict {
		for _, computedProperty := range dict.Items() {
			key := computedProperty[0].(starlark.String).GoString()
			visitAllComputedProperties(fmt.Sprintf("%s.%s", path, key), computedProperty[1], visit)
		}
	} else {
		visit(path)
	}
}

// starlarkScript MUST have a `compute()` function, which should evaluate to a
// dict(most likely nested) or a string in case of error
func runStarlarkPropertyComputation(ctx BottomUpMutatorContext, starlarkScript *string) *starlark.Dict {
	thread := &starlark.Thread{Name: "starlark-" + ctx.ModuleName()}
	globals, err := starlark.ExecFile(thread, "script", *starlarkScript, nil)
	reportError := generateStarlarkErrorReporter(ctx, starlarkScript)
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
		kwargs = append(kwargs, starlark.Tuple{starlark.String(name), convertGoContextToStarlark(ctx)})
	}
	value, err = starlark.Call(thread, compute, nil, kwargs)
	if err != nil {
		reportError(err.Error())
		return nil
	}
	computedProperties, isDict := value.(*starlark.Dict)
	if !isDict {
		reportError(value.String())
		return nil
	}
	computedProperties.Freeze()
	return computedProperties
}

// in case of syntax error in `compute()`, the starlark interpreter will report
// line numbers relative to the starlark code, which won't correspond to actual
// line numbers in the Android.bp file. Thus, we enumerate the starlark script.
func generateStarlarkErrorReporter(ctx BottomUpMutatorContext, starlarkScript *string) starlarkErrorReporterFn {
	withLineNums := ""
	for i, s := range strings.Split(*starlarkScript, "\n") {
		withLineNums += fmt.Sprintf("%3d: %s\n", i+1, s)
	}
	return func(format string, args ...interface{}) {
		ctx.PropertyErrorf("computed", format+"\n%s", append(args, withLineNums)...)
	}
}

// applyComputeProperty effectively attempts `targetStruct[key] = value`
// `appliedLeaves` are property paths that were found and set
func applyComputedProperty(
	reportError starlarkErrorReporterFn,
	pathPrefix string,
	targetStruct reflect.Value,
	computedProperty starlark.Tuple,
) (appliedLeaves stringSet) {
	appliedLeaves = make(stringSet)
	key := computedProperty[0].(starlark.String).GoString()
	path := fmt.Sprintf("%s.%s", pathPrefix, key)
	structType := targetStruct.Type()
	fieldName := proptools.FieldNameForProperty(key)
	field, hasField := structType.FieldByName(fieldName)
	if !hasField {
		return
	}
	if !field.IsExported() {
		reportError("%s is not visible", path)
		return
	}
	if proptools.HasTag(field, "blueprint", "mutated") {
		reportError("%s is tagged blueprint:\"mutated\"", path)
		return
	}
	fieldValue := targetStruct.FieldByName(fieldName)
	if !fieldValue.CanSet() {
		reportError("%s is not settable", path)
		return
	}

	anchor := fieldValue
	switch fieldValue.Kind() {
	case reflect.Interface:
		//run-time defined structs
		fieldValue = fieldValue.Elem()
		if fieldValue.Kind() != reflect.Ptr {
			reportError("%s is %s instead of a pointer", path, fieldValue.Kind())
			return
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

	value := computedProperty[1]
	if fieldValue.Kind() == reflect.Struct {
		for _, pair := range value.(*starlark.Dict).Items() {
			appliedLeaves2 := applyComputedProperty(reportError, path, fieldValue, pair)
			for k, v := range appliedLeaves2 {
				appliedLeaves[k] = v
			}
		}
		return
	} else {
		if !fieldValue.IsZero() {
			reportError("%s is already set", path)
			return
		}
		var v reflect.Value
		v, err := convertStarlarkToGo(value, fieldValue.Type())
		if err != nil {
			reportError(err.Error())
			return
		}
		proptools.ExtendBasicType(fieldValue, v, proptools.Append)
		appliedLeaves[path] = starlarkComputedPropertyMarker
		return
	}
}

func convertGoValueToStarlark(goValue reflect.Value) (starlark.Value, error) {
	switch goValue.Kind() {
	case reflect.Ptr:
		return convertGoValueToStarlark(goValue.Elem())
	case reflect.String:
		return starlark.String(goValue.Interface().(string)), nil
	case reflect.Slice:
		var slice []starlark.Value
		for i := 0; i < goValue.Len(); i++ {
			elem, err := convertGoValueToStarlark(goValue.Index(i))
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

func getFieldValueFromStruct(pSourceStruct interface{}, fieldName string) (starlark.Value, error) {
	r := reflect.ValueOf(pSourceStruct).Elem()
	field, has := r.Type().FieldByName(fieldName)
	if has && field.IsExported() {
		return convertGoValueToStarlark(r.FieldByName(fieldName))
	}
	return starlark.None, fmt.Errorf("no such field: %s", fieldName)
}

func callMethodOnStruct(pSourceStruct interface{}, methodName string) (starlark.Value, error) {
	r := reflect.ValueOf(pSourceStruct).Elem()
	method, has := r.Type().MethodByName(methodName)
	if has && method.IsExported() {
		return convertGoValueToStarlark(r.MethodByName(methodName))
	}
	return starlark.None, fmt.Errorf("no such method: %s", methodName)
}

func convertGoFunctionToStarlark(fnName string, goFn func(singleArg string) (starlark.Value, error)) *starlark.Builtin {
	return starlark.NewBuiltin(fnName, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var singleArg string
		if err := starlark.UnpackArgs(b.Name(), args, kwargs, 1, &singleArg); err != nil {
			return nil, err
		}
		return goFn(singleArg)
	})
}

func convertGoContextToStarlark(ctx BaseMutatorContext) *starlarkstruct.Struct {

	consts := starlark.StringDict{}
	for k, v := range constantsAvailableInStarlark {
		consts[k] = starlark.String(*v)
	}

	//TODO(usta) allow-list what's accessible in Starlark?
	device := starlark.StringDict{
		"get": convertGoFunctionToStarlark("get", func(key string) (starlark.Value, error) {
			return callMethodOnStruct(ctx.DeviceConfig().deviceConfig, key)
		}),
	}

	//TODO(usta) allow-list what's accessible in Starlark?
	env := starlark.StringDict{
		"get": convertGoFunctionToStarlark("get", func(key string) (starlark.Value, error) {
			value := ctx.Config().Getenv(key)
			if value == "" {
				return starlark.None, nil
			}
			return starlark.String(value), nil
		}),
		"is_true": convertGoFunctionToStarlark("is_true", func(key string) (starlark.Value, error) {
			return starlark.Bool(ctx.Config().IsEnvTrue(key)), nil
		}),
	}

	//TODO(usta) allow-list what's accessible in Starlark?
	product := starlark.StringDict{
		"get": convertGoFunctionToStarlark("get", func(key string) (starlark.Value, error) {
			p := ctx.Config().productVariables
			return getFieldValueFromStruct(&p, key)
		}),
	}

	starlarkStruct := starlarkstruct.FromStringDict(starlark.None, starlark.StringDict{
		"module_name":       starlark.String(ctx.ModuleName()), // moduleName can be changes by other mutators - caution using this
		"constants":         starlarkstruct.FromStringDict(starlark.None, consts),
		"device":            starlarkstruct.FromStringDict(starlark.None, device),
		"env":               starlarkstruct.FromStringDict(starlark.None, env),
		"product_variables": starlarkstruct.FromStringDict(starlark.None, product),
	})
	starlarkStruct.Freeze()
	return starlarkStruct
}

func convertStarlarkToGo(starlarkValue starlark.Value, t reflect.Type) (_ reflect.Value, err error) {
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
			err = e
			return
		}
		return reflect.ValueOf(value), nil
	case reflect.Slice:
		l := starlarkValue.(*starlark.List)
		slice := reflect.MakeSlice(t, 0, l.Len())
		for i := 0; i < l.Len(); i++ {
			value, e := convertStarlarkToGo(l.Index(i), t.Elem())
			if e != nil {
				err = e
				return
			}
			slice = reflect.Append(slice, value)
		}
		return slice, nil
	case reflect.Ptr:
		value, e := primitive(starlarkValue)
		if e != nil {
			err = e
			return
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
		err = fmt.Errorf("unsupported %s for %v at %s", starlarkValue, t, "todo pass attribute path for info")
		return
	}
}
