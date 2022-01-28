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
	"io/ioutil"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/blueprint/pathtools"
	"github.com/google/blueprint/proptools"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var constantsAvailableInStarlark = make(map[string]*string)

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

// RegisterComputedPropertiesPreArchMutator register the mutator that detects
// computedProperties, evaluates them and sets the relevant properties on
// the module. The computation can be specified under the "computed" property in
// two ways:
//	1. with a starlark function inlined in the blueprint file itself
//  2. with a reference to a starlark file and the function in it
// In option 1, a function named `compute` is expected with the following
// signature, using Python type hints for illustrative purpose only:
//    Result = dict[str, int | str | bool | 'Result']
//		def compute(struct ctx) -> Result
// If there are additional parameters (e.g. for use from other functions), then
// default parameter values should be provided for all but the first.
func RegisterComputedPropertiesPreArchMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("computed_properties", computedPropertiesMutator).Parallel()
}

type starlarkErrorReporterFn = func(fmt string, args ...interface{})

const starlarkFailFast = false

func computedPropertiesMutator(ctx BottomUpMutatorContext) {
	starlarkScript, starlarkFilename, starlarkFunctionName := getStarlarkReference(ctx)
	if starlarkScript == nil {
		return
	}
	allComputedProperties := runStarlarkPropertyComputation(ctx, starlarkScript, starlarkFilename, starlarkFunctionName)
	if allComputedProperties == nil {
		return
	}

	reportError := generateStarlarkErrorReporter(ctx, starlarkScript)
	allAppliedLeaves := applyComputedProperties(ctx, reportError, allComputedProperties)

	visitAllComputedProperties(ctx.ModuleName(), allComputedProperties, func(path string) {
		if _, applied := allAppliedLeaves[path]; !applied {
			reportError("property %s does not exist", path)
		}
	})
}

func getStarlarkReference(ctx BottomUpMutatorContext) (starlarkScript *string, starlarkFilename string, starlarkFunctionName string) {
	starlarkScript = ctx.Module().base().computedProperties.Computed
	if starlarkScript == nil {
		return
	}

	starlarkScriptLoader := regexp.MustCompile(`^file:(?P<file>[^#]+)(?:#(?P<fnName>.+))?$`)
	matches := starlarkScriptLoader.FindStringSubmatch(*starlarkScript)
	starlarkFilename = ctx.BlueprintsFile()
	starlarkFunctionName = "compute"
	if matches == nil {
		// inline script
		starlarkFilename += "-inline"
		// Note that when the script is inlined, it becomes tricky to correctly report
		// line numbers in case of syntax errors because from the interpreter, we
		// will get the effective line number in the script we passed to it but the
		// location of the script in Android.bp is unknown (also think about string
		// concatenation support in top level variables in Android.bp)
	} else {
		// not inline script
		starlarkFilename = path.Join(filepath.Dir(ctx.BlueprintsFile()), matches[1])
		file, err := ctx.Config().fs.Open(starlarkFilename)
		if err != nil {
			ctx.PropertyErrorf("compute", err.Error())
			return
		}
		defer func(file pathtools.ReaderAtSeekerCloser) {
			err := file.Close()
			if err != nil {
				//log warning
			}
		}(file)
		data, err := ioutil.ReadAll(file)
		if err != nil {
			ctx.PropertyErrorf("compute", err.Error())
			return
		}
		starlarkScript = proptools.StringPtr(string(data))
		starlarkFunctionName = matches[2]
	}
	return
}

func runStarlarkPropertyComputation(ctx BottomUpMutatorContext, starlarkScript *string, starlarkFilename string, starlarkFnName string) *starlark.Dict {
	thread := &starlark.Thread{Name: "starlark-" + ctx.ModuleName()}
	globals, err := starlark.ExecFile(thread, starlarkFilename, *starlarkScript, nil)
	reportError := generateStarlarkErrorReporter(ctx, starlarkScript)
	if err != nil {
		reportError(err.Error())
		return nil
	}
	value, hasValue := globals[starlarkFnName]
	if !hasValue {
		reportError("no such starlark function: %s", starlarkFnName)
		return nil
	}
	compute, isFunc := value.(*starlark.Function)
	if !isFunc {
		reportError("expected *starlark.Function but got %T", value)
		return nil
	}

	var kwargs []starlark.Tuple
	name, _ := compute.Param(0)
	kwargs = append(kwargs, starlark.Tuple{starlark.String(name), convertGoContextToStarlark(ctx)})

	value, err = starlark.Call(thread, compute, nil, kwargs)
	if err != nil {
		reportError(err.Error())
		return nil
	}
	computedProperties, isDict := value.(*starlark.Dict)
	if !isDict {
		reportError("expected a dict but got %T", value)
		return nil
	}
	computedProperties.Freeze()
	return computedProperties
}

// applyComputedProperties finds all the properties in
// allComputedProperties that were actually assigned to the module
func applyComputedProperties(
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
				allAppliedLeaves[applied] = struct{}{}
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

// in case of syntax error in `compute()`, the starlark interpreter will report
// line numbers relative to the starlark code, which won't correspond to actual
// line numbers in the Android.bp file. Thus, we enumerate the starlark script.
func generateStarlarkErrorReporter(ctx BottomUpMutatorContext, starlarkScript *string) starlarkErrorReporterFn {
	withLineNums := ""
	lines := strings.Split(*starlarkScript, "\n")
	maxLine := len(fmt.Sprintf("%d", len(lines)))
	formatString := fmt.Sprintf(`%%%dd: %%s\n`, maxLine)
	for i, s := range lines {
		withLineNums += fmt.Sprintf(formatString, i+1, s)
	}
	return func(format string, args ...interface{}) {
		ctx.PropertyErrorf("computed", format+"\n%s", append(args, withLineNums)...)
		if starlarkFailFast {
			panic("FAIL FAST MODE ON")
		}
	}
}

// applyComputeProperty effectively attempts `targetStruct[key] = value`
// returns property paths that were found and set
func applyComputedProperty(
	reportError starlarkErrorReporterFn,
	pathPrefix string,
	targetStruct reflect.Value,
	computedProperty starlark.Tuple,
) stringSet {

	appliedLeaves := make(stringSet)
	key := computedProperty[0].(starlark.String).GoString()
	structType := targetStruct.Type()
	if structType.Kind() != reflect.Struct {
		reportError("Expected a struct %s %T", pathPrefix, structType)
		return appliedLeaves
	}
	fieldName := proptools.FieldNameForProperty(key)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := targetStruct.Field(i)
		if field.Anonymous {
			//assume struct pointer
			if field.Type.Kind() != reflect.Ptr && field.Type.Elem().Kind() != reflect.Struct {
				reportError("expected an embedded pointer to struct under %s not %s", pathPrefix, field.Type.Kind())
				continue
			}
			if fieldValue.IsNil() {
				fieldValue = reflect.New(field.Type.Elem())
				targetStruct.Field(i).Set(fieldValue)
			}
			appliedLeaves2 := applyComputedProperty(reportError, pathPrefix, fieldValue.Elem(), computedProperty)
			for k, v := range appliedLeaves2 {
				appliedLeaves[k] = v
			}
		} else if field.Name == fieldName {
			//no "break" - must follow all anonymous/embedded struct pointers
			thisPath := fmt.Sprintf("%s.%s", pathPrefix, key)
			if properKey := proptools.PropertyNameForField(fieldName); key != properKey {
				reportError("property %s does not exist - did you mean %s?", thisPath,
					fmt.Sprintf("%s.%s", pathPrefix, properKey))
				continue
			}
			if !field.IsExported() {
				reportError("property %s is not visible", thisPath)
				continue
			}
			if proptools.HasTag(field, "blueprint", "mutated") {
				reportError("property %s is tagged blueprint:\"mutated\"", thisPath)
				continue
			}
			if !fieldValue.CanSet() {
				reportError("property %s is not settable", thisPath)
				continue
			}

			anchor := fieldValue
			switch fieldValue.Kind() {
			case reflect.Interface:
				//run-time defined structs
				fieldValue = fieldValue.Elem()
				if fieldValue.Kind() != reflect.Ptr {
					reportError("property %s is %s instead of a pointer", thisPath, fieldValue.Kind())
					continue
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
					appliedLeaves2 := applyComputedProperty(reportError, thisPath, fieldValue, pair)
					for k, v := range appliedLeaves2 {
						appliedLeaves[k] = v
					}
				}
			} else {
				if !fieldValue.IsZero() {
					reportError("property %s is already set", thisPath)
				}
				v, err := convertStarlarkToGo(value, fieldValue.Type())
				if err != nil {
					reportError("property %s: %s", thisPath, err.Error())
				}
				proptools.ExtendBasicType(fieldValue, v, proptools.Append)
				appliedLeaves[thisPath] = struct{}{}
			}
		}
	}
	return appliedLeaves
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

func callMethodOnStruct(pSourceStruct interface{}, methodName string) (starlark.Value, error) {
	r := reflect.ValueOf(pSourceStruct).Elem()
	method, has := r.Type().MethodByName(methodName)
	if has && method.IsExported() {
		return convertGoValueToStarlark(r.MethodByName(methodName))
	}
	return starlark.None, fmt.Errorf("no such method: %s.%s", r.Type(), methodName)
}

func convertGoUnaryFunctionToStarlark(fnName string, goFn func(singleArg string) (starlark.Value, error)) *starlark.Builtin {
	return starlark.NewBuiltin(fnName, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var singleArg string
		if err := starlark.UnpackArgs(b.Name(), args, kwargs, 1, &singleArg); err != nil {
			return nil, err
		}
		return goFn(singleArg)
	})
}

func convertGoNullaryFunctionToStarlark(fnName string, goFn func() (starlark.Value, error)) *starlark.Builtin {
	return starlark.NewBuiltin(fnName, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return goFn()
	})
}

func convertGoContextToStarlark(ctx BaseMutatorContext) *starlarkstruct.Struct {

	consts := starlark.StringDict{}
	for k, v := range constantsAvailableInStarlark {
		consts[k] = starlark.String(*v)
	}

	//TODO(usta) allow-list what's accessible in Starlark?
	device := starlark.StringDict{
		"get": convertGoUnaryFunctionToStarlark("get", func(key string) (starlark.Value, error) {
			return callMethodOnStruct(ctx.DeviceConfig().deviceConfig, key)
		}),
	}

	//TODO(usta) allow-list what's accessible in Starlark?
	env := starlark.StringDict{
		"get": convertGoUnaryFunctionToStarlark("get", func(key string) (starlark.Value, error) {
			value := ctx.Config().Getenv(key)
			if value == "" {
				return starlark.None, nil
			}
			return starlark.String(value), nil
		}),
		"is_true": convertGoUnaryFunctionToStarlark("is_true", func(key string) (starlark.Value, error) {
			return starlark.Bool(ctx.Config().IsEnvTrue(key)), nil
		}),
	}

	starlarkContext := starlarkstruct.FromStringDict(starlark.None, starlark.StringDict{
		"module_name": starlark.String(ctx.ModuleName()), // moduleName can be changes by other mutators - caution using this
		"constants":   starlarkstruct.FromStringDict(starlark.None, consts),
		"device":      starlarkstruct.FromStringDict(starlark.None, device),
		"env":         starlarkstruct.FromStringDict(starlark.None, env),
		"sanitize_host": convertGoNullaryFunctionToStarlark("sanitize_host", func() (starlark.Value, error) {
			var values []starlark.Value
			for _, x := range ctx.Config().SanitizeHost() {
				values = append(values, starlark.String(x))
			}
			return starlark.NewList(values), nil
		}),
	})
	starlarkContext.Freeze()
	return starlarkContext
}

func convertStarlarkToGo(starlarkValue starlark.Value, t reflect.Type) (_ reflect.Value, err error) {
	switch t.Kind() {
	case reflect.Bool:
		value, isBool := starlarkValue.(starlark.Bool)
		if !isBool {
			err = fmt.Errorf("%T %s cannot be cast to %s", starlarkValue, starlarkValue, t)
			return
		}
		return reflect.ValueOf(bool(value)), nil
	case reflect.String:
		value, isString := starlarkValue.(starlark.String)
		if !isString {
			err = fmt.Errorf("%T %s cannot be cast to %s", starlarkValue, starlarkValue, t)
			return
		}
		return reflect.ValueOf(value.GoString()), nil
	case reflect.Int64:
		value, isInt := starlarkValue.(starlark.Int)
		if !isInt {
			err = fmt.Errorf("%T %s cannot be cast to %s", starlarkValue, starlarkValue, t)
			return
		}
		intVal, ok := value.Int64()
		if !ok {
			err = fmt.Errorf("%T %s cannot be cast to %s", starlarkValue, starlarkValue, t)
			return
		}
		return reflect.ValueOf(intVal), nil
	case reflect.Slice:
		l := starlarkValue.(*starlark.List)
		slice := reflect.MakeSlice(t, 0, l.Len())
		for i := 0; i < l.Len(); i++ {
			value, e := convertStarlarkToGo(l.Index(i), t.Elem())
			if e != nil {
				err = fmt.Errorf("%s[index:%d]: %s", l, i, e)
				return
			}
			slice = reflect.Append(slice, value)
		}
		return slice, nil
	case reflect.Ptr:
		switch t.Elem().Kind() {
		case reflect.Bool, reflect.String, reflect.Int64:
			value, e := convertStarlarkToGo(starlarkValue, t.Elem())
			if e != nil {
				err = e
				return
			} else {
				switch t.Elem().Kind() {
				case reflect.Bool:
					return reflect.ValueOf(proptools.BoolPtr(value.Interface().(bool))), nil
				case reflect.String:
					return reflect.ValueOf(proptools.StringPtr(value.Interface().(string))), nil
				case reflect.Int64:
					return reflect.ValueOf(proptools.Int64Ptr(value.Interface().(int64))), nil
				}
			}
		}
		fallthrough
	default:
		err = fmt.Errorf("unsupported %s for %v", starlarkValue, t)
		return
	}
}
