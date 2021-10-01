// Copyright 2019 Google Inc. All rights reserved.
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
)

// This file implements support for automatically adding dependencies on any module referenced
// with the ":module" module reference syntax in a property that is annotated with `android:"path"`.
// The dependency is used by android.PathForModuleSrc to convert the module reference into the path
// to the output file of the referenced module.

func registerPathDepsMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("pathdeps", pathDepsMutator).Parallel()
}

// The pathDepsMutator automatically adds dependencies on any module that is listed with the
// ":module" module reference syntax in a property that is tagged with `android:"path"`.
func pathDepsMutator(ctx BottomUpMutatorContext) {
	props := ctx.Module().base().generalProperties
	addPathDepsForProps(ctx, props)
}

type pathPropertyValue struct {
	module string
	tag    string

	includeLicense bool
}

func firstUniquePathProperties(list []pathPropertyValue) []pathPropertyValue {
	k := 0
outer:
	for i := 0; i < len(list); i++ {
		for j := 0; j < k; j++ {
			if list[i].module == list[j].module && list[i].tag == list[j].tag {
				list[j].includeLicense = list[j].includeLicense || list[i].includeLicense
				continue outer
			}
		}
		list[k] = list[i]
		k++
	}
	return list[:k]
}

func addPathDepsForProps(ctx BottomUpMutatorContext, props []interface{}) {
	// Iterate through each property struct of the module extracting the contents of all properties
	// tagged with `android:"path"`.
	var pathProperties []pathPropertyValue
	for _, ps := range props {
		pathProperties = append(pathProperties, pathPropertiesForPropertyStruct(ps)...)
	}

	// Remove duplicates to avoid multiple dependencies.
	pathProperties = firstUniquePathProperties(pathProperties)

	// Add dependencies to anything that is a module reference.
	for _, s := range pathProperties {
		ctx.AddDependency(ctx.Module(), sourceOrOutputDepTag(s.module, s.tag, s.includeLicense), s.module)
	}
}

// pathPropertiesForPropertyStruct uses the indexes of properties that are tagged with
// android:"path" to extract all their values from a property struct, returning them as a single
// slice of strings.
func pathPropertiesForPropertyStruct(ps interface{}) []pathPropertyValue {
	v := reflect.ValueOf(ps)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("type %s is not a pointer to a struct", v.Type()))
	}

	// If the property struct is a nil pointer it can't have any paths set in it.
	if v.IsNil() {
		return nil
	}

	// v is now the reflect.Value for the concrete property struct.
	v = v.Elem()

	// Get or create the list of indexes of properties that are tagged with `android:"path"`.
	pathPropertyIndexes := pathPropertyIndexesForPropertyStruct(ps)

	var ret []pathPropertyValue

	for _, prop := range pathPropertyIndexes {
		i := prop.index
		var values []reflect.Value
		fieldsByIndex(v, i, &values)
		for _, sv := range values {
			if !sv.IsValid() {
				// Skip properties inside a nil pointer.
				continue
			}

			// If the field is a non-nil pointer step into it.
			if sv.Kind() == reflect.Ptr {
				if sv.IsNil() {
					continue
				}
				sv = sv.Elem()
			}

			// Collect paths from all strings and slices of strings.
			switch sv.Kind() {
			case reflect.String:
				if m, t := SrcIsModuleWithTag(sv.String()); m != "" {
					ret = append(ret, pathPropertyValue{
						module: m,
						tag:    t,

						includeLicense: prop.includeLicense,
					})
				}
			case reflect.Slice:
				strs := sv.Interface().([]string)
				paths := make([]pathPropertyValue, 0, len(strs))
				for _, str := range strs {
					if m, t := SrcIsModuleWithTag(str); m != "" {
						paths = append(paths, pathPropertyValue{
							module: m,
							tag:    t,

							includeLicense: prop.includeLicense,
						})
					}
				}
				ret = append(ret, paths...)
			default:
				panic(fmt.Errorf(`field %s in type %s has tag android:"path" but is not a string or slice of strings, it is a %s`,
					v.Type().FieldByIndex(i).Name, v.Type(), sv.Type()))
			}
		}
	}

	return ret
}

// fieldsByIndex is similar to reflect.Value.FieldByIndex, but is more robust: it doesn't track
// nil pointers and it returns multiple values when there's slice of struct.
func fieldsByIndex(v reflect.Value, index []int, values *[]reflect.Value) {
	// leaf case
	if len(index) == 1 {
		if isSliceOfStruct(v) {
			for i := 0; i < v.Len(); i++ {
				*values = append(*values, v.Index(i).Field(index[0]))
			}
		} else {
			// Dereference it if it's a pointer.
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return
				}
				v = v.Elem()
			}
			*values = append(*values, v.Field(index[0]))
		}
		return
	}

	// recursion
	if v.Kind() == reflect.Ptr {
		// don't track nil pointer
		if v.IsNil() {
			return
		}
		v = v.Elem()
	} else if isSliceOfStruct(v) {
		// do the recursion for all elements
		for i := 0; i < v.Len(); i++ {
			fieldsByIndex(v.Index(i).Field(index[0]), index[1:], values)
		}
		return
	}
	fieldsByIndex(v.Field(index[0]), index[1:], values)
	return
}

func isSliceOfStruct(v reflect.Value) bool {
	return v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Struct
}

var pathPropertyIndexesCache OncePer

type pathProperty struct {
	index          []int
	includeLicense bool
}

// pathPropertyIndexesForPropertyStruct returns a list of all of the indexes of properties in
// property struct type that are tagged with `android:"path"`.  Each index is a []int suitable for
// passing to reflect.Value.FieldByIndex.  The value is cached in a global cache by type.
func pathPropertyIndexesForPropertyStruct(ps interface{}) []pathProperty {
	key := NewCustomOnceKey(reflect.TypeOf(ps))
	return pathPropertyIndexesCache.Once(key, func() interface{} {
		t := reflect.TypeOf(ps)
		if !isStructPtr(t) {
			panic(fmt.Errorf("type %s is not a pointer to a struct", t))
		}
		t = t.Elem()

		return propertyIndexesWithTag(t)
	}).([]pathProperty)
}

func propertyIndexesWithTag(t reflect.Type) []pathProperty {
	var indexes []pathProperty

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		ft := field.Type
		if isStruct(ft) || isStructPtr(ft) || isSliceOfStructType(ft) {
			if ft.Kind() == reflect.Ptr || ft.Kind() == reflect.Slice || ft.Kind() == reflect.Map {
				ft = ft.Elem()
			}
			subIndexes := propertyIndexesWithTag(ft)
			for _, sub := range subIndexes {
				sub.index = append([]int{i}, sub.index...)
				indexes = append(indexes, sub)
			}
		} else if proptools.HasTag(field, "android", "path") {
			indexes = append(indexes, pathProperty{
				index:          field.Index,
				includeLicense: proptools.HasTag(field, "android", "include_licenses"),
			})
		}
	}

	return indexes
}

func isStruct(t reflect.Type) bool {
	return t.Kind() == reflect.Struct
}

func isStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

func isSlice(t reflect.Type) bool {
	return t.Kind() == reflect.Slice
}

func isSliceOfStructType(t reflect.Type) bool {
	return isSlice(t) && isStruct(t.Elem())
}
