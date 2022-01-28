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

//the value is immaterial and will always be set to garbage
type stringSet map[string]struct{}

//used simply as an unuseful value in maps to use them as a set
var garbage = struct{}{}

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
	//adds all entries from the source to the target
	union := func(target stringSet, source stringSet) {
		for k, v := range source {
			target[k] = v
		}
	}
	// TODO @usta parallelize ?
	for _, computedProperty := range allComputedProperties.Items() {
		rootProperty := computedProperty[0].(starlark.String).GoString()
		rootValue := computedProperty[1]
		var allErrors []string
		allMissing := stringSet{fmt.Sprintf("%s.%s", ctx.ModuleName(), rootProperty): garbage}
		allAppliedLeaves := make(stringSet)
		for _, pRootStruct := range ctx.Module().GetProperties() {
			if _, hasComputedProperties := pRootStruct.(*computedProperties); hasComputedProperties {
				continue
			}
			// We are guaranteed to reach here for some `pRootStruct` because
			// `computed` must have come from one.
			rootStruct := reflect.ValueOf(pRootStruct).Elem() // <== ptr dereference
			missing, appliedLeaves, errors := applyComputedProperty(ctx.ModuleName(), rootStruct, rootProperty, rootValue)
			for _, e := range errors {
				allErrors = append(allErrors, fmt.Sprintf("[%s] %s", rootStruct.Type().Name(), e.Error()))
			}
			//TODO extract this housekeeping into a separate function for readability
			if len(missing) == 0 {
				// this pRootStruct consumed everything from rootValue
				allMissing = missing
			}
			if len(allMissing) == 0 {
				// a previous pRootStruct already consumed everything from rootValue
				continue
			}
			union(allAppliedLeaves, appliedLeaves)
			for k1 := range allMissing {
				for k2 := range missing {
					if strings.HasPrefix(k2, k1+".") {
						//one (or more) specific path is in missing so the parent can be
						//replaced by this (and its siblings from subsequent loop)
						delete(allMissing, k1)
						allMissing[k2] = garbage
					}
				}
			}
			for k2 := range allAppliedLeaves {
				delete(allMissing, k2)
			}
		}
		for _, e := range allErrors {
			reportError(e)
		}
		for m := range allMissing {
			reportError("%s does not exist", m)
		}
	}
}

// starlarkScript MUST have define a `compute()` function, which should
// evaluate to a dict(most likely nested) or a string in case of error
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

// in case of syntax error in the starlark function, the interpreter will report
// line numbers relevant to the starlark code block which won't correspond to
// any physical file. Thus we attach the resultant script with line numbers
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
// `pathPrefix` is the "path" in the module where the `targetStruct` is located
// `missing` are property paths that aren't applicable to the `targetStruct`
// `appliedLeaves` are property paths that were found and set
// `errors` correspond to property paths that were found but couldn't be set
func applyComputedProperty(pathPrefix string, targetStruct reflect.Value, key string, value starlark.Value) (missing stringSet, appliedLeaves stringSet, errors []error) {
	missing = make(stringSet)
	appliedLeaves = make(stringSet)
	path := fmt.Sprintf("%s.%s", pathPrefix, key)
	structType := targetStruct.Type()
	fieldName := proptools.FieldNameForProperty(key)
	field, hasField := structType.FieldByName(fieldName)
	if !hasField {
		missing[path] = garbage
		return
	}
	if !field.IsExported() {
		errors = append(errors, fmt.Errorf("%s is not visible", path))
		return
	}
	if proptools.HasTag(field, "blueprint", "mutated") {
		errors = append(errors, fmt.Errorf("%s is tagged blueprint:\"mutated\"", path))
		return
	}

	fieldValue := targetStruct.FieldByName(fieldName)
	if !fieldValue.CanSet() {
		errors = append(errors, fmt.Errorf("%s is not settable", path))
		return
	}

	anchor := fieldValue
	switch fieldValue.Kind() {
	case reflect.Interface:
		//run-time defined structs
		fieldValue = fieldValue.Elem()
		if fieldValue.Kind() != reflect.Ptr {
			errors = append(errors, fmt.Errorf("%s is %s instead of a pointer", path, fieldValue.Kind()))
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

	switch fieldValue.Kind() {
	case reflect.Struct:
		for _, pair := range value.(*starlark.Dict).Items() {
			subKey := pair[0].(starlark.String).GoString()
			missing2, applied2, errors2 := applyComputedProperty(path, fieldValue, subKey, pair[1])
			for k, v := range missing2 {
				missing[k] = v
			}
			for k, v := range applied2 {
				appliedLeaves[k] = v
			}
			errors = append(errors, errors2...)
		}
		return
	default:
		if !fieldValue.IsZero() {
			errors = append(errors, fmt.Errorf("%s is already set", path))
			return
		}
		var v reflect.Value
		v, err := convertStarlarkToGo(value, fieldValue.Type())
		if err != nil {
			errors = append(errors, err)
			return
		}
		proptools.ExtendBasicType(fieldValue, v, proptools.Append)
		appliedLeaves[path] = garbage
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
