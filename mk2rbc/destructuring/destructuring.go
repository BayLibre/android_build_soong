// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package destructuring

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"unsafe"
)

func Destructure(input reflect.Value, result reflect.Value) bool {
	//fmt.Fprintf(os.Stderr, "Input kind: %s, result kind: %s\n", input.Kind().String(), result.Kind().String())
	if input.Kind() == reflect.Interface && result.Kind() != reflect.Interface {
		input = input.Elem()
	}
	fmt.Fprintf(os.Stderr, "Input kind: %s, result kind: %s\n", input.Kind().String(), result.Kind().String())

	switch result.Kind() {
	case reflect.Bool:
		fallthrough
	case reflect.Int,  reflect.Int8,  reflect.Int16,  reflect.Int32, reflect.Int64:
		fallthrough
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fallthrough
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		fallthrough
	case reflect.String:
		if input.Kind() != result.Kind() {
			fmt.Fprintf(os.Stderr, "Simple kinds did not match\n")
			return false
		}
		result.Set(input)
		return true
	case reflect.Interface:
		if input.Kind() != reflect.Interface {
			return false
		}
		return Destructure(input.Elem(), result.Elem())
	case reflect.Pointer:
		if input.Kind() != reflect.Pointer {
			return false
		}
		// If the input is a null pointer, set the result to a null pointer
		if input.Elem().Kind() == reflect.Invalid {
			result.Set(reflect.New(result.Type()).Elem())
			return true
		}
		// if the result is a null pointer, point it at a new object before recursing
		if result.Elem().Kind() == reflect.Invalid {
			result.Set(reflect.New(result.Type().Elem()))
		}
		return Destructure(input.Elem(), result.Elem())
	case reflect.Slice:
		if input.Kind() != reflect.Slice {
			fmt.Fprintf(os.Stderr, "Input is not a slice\n")
			return false
		}
		resultElementType := result.Type().Elem()
		if !resultElementType.AssignableTo(input.Type().Elem()) {
			fmt.Fprintf(os.Stderr, "Result type is not assignable to input type\n")
			return false
		}
		if result.Len() != input.Len() {
			return false
		}
		for i := 0; i < input.Len(); i++ {
			if !Destructure(input.Index(i), result.Index(i)) {
				fmt.Fprintf(os.Stderr, "Recursive destructure of slice element failed\n")
				return false
			}
		}
		return true
	case reflect.Struct:
		if !structsCompatible(input.Type(), result.Type()) {
			fmt.Fprintf(os.Stderr, "Structs incompatible\n")
			return false
		}
		if !input.CanAddr() || !result.CanAddr() {
			panic(fmt.Sprintf("Destructuring a struct requires the structs to be addressable. Input addressable? %t, Result addressable? %t", input.CanAddr(), result.CanAddr()))
		}
		if f, ok := result.Type().FieldByName("StructType"); ok {
			inputStructName := input.Type().Name()
			if f.Tag.Get("required") != inputStructName {
				return false
			}
			result.FieldByName("StructType").Set(reflect.ValueOf(inputStructName))
		}
		for i := 0; i < input.NumField(); i++ {
			inputFieldType := input.Type().Field(i)
			inputField := input.Field(i)
			fmt.Fprintf(os.Stderr, "field name: %q\n", inputFieldType.Name)
			resultFieldType, _ := result.Type().FieldByName(inputFieldType.Name)
			resultField := result.FieldByName(inputFieldType.Name)
			// These next two lines are a workaround that allow us to access unexported fields
			inputField = reflect.NewAt(inputField.Type(), unsafe.Pointer(inputField.UnsafeAddr())).Elem()
			resultField = reflect.NewAt(resultField.Type(), unsafe.Pointer(resultField.UnsafeAddr())).Elem()
			if required := resultFieldType.Tag.Get("required"); !requiredMatches(required, inputField, resultField) {
				fmt.Fprintf(os.Stderr, "required doesn't match\n")
				return false
			}
			if !Destructure(inputField, resultField) {
				fmt.Fprintf(os.Stderr, "Nested destructure failed\n")
				return false
			}
		}
		return true
	default:
		panic("Unsupported type: "+result.Kind().String())
	}
	return false
}

func structsCompatible(struct1 reflect.Type, struct2 reflect.Type) bool {
	if struct1 == struct2 {
		return true
	}
	check := func(a reflect.Type, b reflect.Type) bool {
		for i:=0; i < a.NumField(); i++ {
			name := a.Field(i).Name
			if name == "StructType" {
				continue
			}
			if _, ok := b.FieldByName(name); !ok {
				fmt.Fprintf(os.Stderr, "Does not have field %s\n", name)
				return false
			}
		}
		return true
	}
	if !check(struct1, struct2) {
		return false
	}
	return check(struct2, struct1)
}

func requiredMatches(required string, inputField reflect.Value, resultField reflect.Value) bool {
	fmt.Fprintf(os.Stderr, "Required was %q\n", required)
	if required == "" {
		return true
	}
	if !isSimpleKind(resultField.Kind()) {
		panic("Can only set required on simple fields")
	}
	if required == "dynamic" {
		return inputField == resultField
	}
	var parsed interface{}
	var err error
	switch resultField.Kind() {
	case reflect.Bool:
		parsed, err = strconv.ParseBool(required)
	case reflect.Int:
		parsed, err = strconv.ParseInt(required, 10, 0)
	case reflect.Int8:
		parsed, err = strconv.ParseInt(required, 10, 8)
	case reflect.Int16:
		parsed, err = strconv.ParseInt(required, 10, 16)
	case reflect.Int32:
		parsed, err = strconv.ParseInt(required, 10, 32)
	case reflect.Int64:
		parsed, err = strconv.ParseInt(required, 10, 64)
	case reflect.Uint:
		parsed, err = strconv.ParseUint(required, 10, 0)
	case reflect.Uint8:
		parsed, err = strconv.ParseUint(required, 10, 8)
	case reflect.Uint16:
		parsed, err = strconv.ParseUint(required, 10, 16)
	case reflect.Uint32:
		parsed, err = strconv.ParseUint(required, 10, 32)
	case reflect.Uint64:
		parsed, err = strconv.ParseUint(required, 10, 64)
	case reflect.Float32:
		parsed, err = strconv.ParseFloat(required, 32)
	case reflect.Float64:
		parsed, err = strconv.ParseFloat(required, 64)
	case reflect.Complex64:
		parsed, err = strconv.ParseComplex(required, 64)
	case reflect.Complex128:
		parsed, err = strconv.ParseComplex(required, 128)
	case reflect.String:
		parsed, err = required, nil
	default:
		panic("Unknown kind")
	}

	if err != nil {
		panic("Not correct type: "+required)
	}
	return inputField.Interface() == parsed
}

func isSimpleKind(kind reflect.Kind) bool {
	return 	kind == reflect.Bool ||
		kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 ||
		kind == reflect.Uint || kind ==reflect.Uint8 || kind ==reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 ||
		kind == reflect.Float32 || kind ==reflect.Float64 || kind ==reflect.Complex64 || kind ==reflect.Complex128 ||
		kind == reflect.String
}