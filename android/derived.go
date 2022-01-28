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
	}
}

//The string associated with derivedProperties is a Starlark function body.
//It should return a map (possibly nested) that provides values for Module.GetProperties()
func evaluateDerivations(ctx BaseMutatorContext, starlarkScript *string) *starlark.Dict {
	api := apiForStarlark(ctx)
	thread := &starlark.Thread{Name: "starlark"}
	globals, err := starlark.ExecFile(thread, ctx.BlueprintsFile(), *starlarkScript, api)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate \"%s.derived\" in %s\n%s\n%w", ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript, err))
	}
	value, hasValue := globals["derive"]
	if !hasValue {
		panic(fmt.Errorf("failed to evaluate \"%s.derived\" in %s\n%s\ndefine a `def derive() -> dict[string, ?]`", ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript))
	}
	if _, isFunc := value.(*starlark.Function); !isFunc {
		panic(fmt.Errorf("failed to evaluate \"%s.derived\" in %s\n%s\ndefine a `def value() -> dict[string, ?]`", ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript))
	}
	value, err = starlark.Call(thread, value, nil, nil)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate \"%s.derived\" in %s\n%s\n%w", ctx.ModuleName(), ctx.BlueprintsFile(), *starlarkScript, err))
	}
	derivations, isDict := value.(*starlark.Dict)
	if !isDict {
		panic(fmt.Errorf("expected \"%s.derived\" in %s to evalueate to a starlark dict but got %T\n%s\n%w", ctx.ModuleName(), ctx.BlueprintsFile(), value, *starlarkScript, err))
	}
	return derivations
}

type set struct {
	m map[string]struct{}
}

func (s *set) String() string {
	keys := make([]string, len(s.m))
	i := 0
	for k := range s.m {
		keys[i] = k
		i++
	}
	sort.Strings(keys) //for determinism
	return strings.Join(keys, ", ")
}

func (s *set) add(item string) {
	s.m[item] = struct{}{}
}

func (s *set) remove(item string) {
	delete(s.m, item)
}

func (s *set) addAll(other *set) {
	if other != nil {
		for k := range other.m {
			s.add(k)
		}
	}
}

func (s *set) removeAll(other *set) {
	if other != nil {
		for k := range other.m {
			s.remove(k)
		}
	}
}

func (s *set) has(item string) bool {
	_, contains := s.m[item]
	return contains
}

func (s *set) isEmpty() bool {
	return len(s.m) == 0
}

func apply(aStruct reflect.Value, parentPath string, key string, value starlark.Value) *set {
	structType := aStruct.Type()
	var used = set{make(map[string]struct{})}
	path := fmt.Sprintf("%s.%s", parentPath, key)
	for i := 0; i < aStruct.NumField(); i++ {
		fieldInfo := structType.Field(i)
		if proptools.FieldNameForProperty(key) != fieldInfo.Name {
			continue
		}
		fieldValue := aStruct.Field(i)
		if !fieldValue.CanSet() {
			panic(fmt.Errorf(`"%s.%s" is not settable`, parentPath, fieldInfo.Name))
		}
		if proptools.HasTag(fieldInfo, "blueprint", "mutated") {
			panic(fmt.Errorf(`"%s.%s" is tagged blueprint:"mutated"`, parentPath, fieldInfo.Name))
		}
		if fieldInfo.Type.Kind() == reflect.Struct {
			valueDict := value.(*starlark.Dict)
			for _, pair := range valueDict.Items() {
				subkey := pair[0].(starlark.String).GoString()
				subvalue := pair[1]
				used.addAll(apply(fieldValue, path, subkey, subvalue))
			}
		} else {
			if !fieldValue.IsZero() {
				maybeStar := ""
				if fieldInfo.Type.Kind() == reflect.Ptr {
					//this is only for human-readability
					fieldValue = fieldValue.Elem()
					maybeStar = "*"
				}
				panic(fmt.Errorf(`already set "%s%s.%s" = %#v`, maybeStar, parentPath, fieldInfo.Name, fieldValue))
			}
			proptools.ExtendBasicType(fieldValue, asValue(value, fieldInfo.Type), proptools.Append)
			used.add(path)
		}
	}
	return &used
}

// TODO @usta parallelize
func applyRoot(pRootStruct interface{}, moduleName string, derivations *starlark.Dict) *set {
	rootStruct := reflect.ValueOf(pRootStruct).Elem()
	var used *set
	for _, derivation := range derivations.Items() {
		key := derivation[0].(starlark.String).GoString()
		value := derivation[1]
		subTreeUsed := apply(rootStruct, moduleName, key, value)
		if used == nil {
			used = subTreeUsed
		} else {
			used.addAll(subTreeUsed)
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
		rootused := applyRoot(pRootStruct, ctx.ModuleName(), derivations)
		unused.removeAll(rootused)
	}
	if !unused.isEmpty() {
		panic(fmt.Errorf("%s unused %q:\n%s", ctx.BlueprintsFile(), unused, derivations.String()))
	}
}

func getPaths(parentPath string, dict *starlark.Dict) *set {
	paths := set{make(map[string]struct{})}
	for _, derivation := range dict.Items() {
		key := derivation[0].(starlark.String).GoString()
		currentPath := fmt.Sprintf("%s.%s", parentPath, key)
		value := derivation[1] //.(starlark.Value)
		if subDict, ok := value.(*starlark.Dict); ok {
			subpaths := getPaths(currentPath, subDict)
			paths.addAll(subpaths)
		} else {
			paths.add(currentPath)
		}
	}
	return &paths
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
