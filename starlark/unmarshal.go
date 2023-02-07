// Copyright 2023 Google Inc. All rights reserved.
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

package starlark

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func Unmarshal[T any](value starlark.Value) (T, error) {
	var zero T
	x, err := UnmarshalReflect(value, reflect.TypeOf(zero))
	return x.Interface().(T), err
}

func UnmarshalReflect(value starlark.Value, ty reflect.Type) (reflect.Value, error) {
	zero := reflect.Zero(ty)
	var result reflect.Value
	if ty.Kind() == reflect.Map {
		result = reflect.MakeMap(ty)
	} else {
		result = reflect.Indirect(reflect.New(ty))
	}

	switch v := value.(type) {
	case starlark.String:
		if result.Type().Kind() != reflect.String {
			return zero, fmt.Errorf("expected string, got %s", result.Type().Kind().String())
		}
		result.SetString(v.GoString())
	case starlark.Int:
		signedValue, signedOk := v.Int64()
		unsignedValue, unsignedOk := v.Uint64()
		switch result.Type().Kind() {
		case reflect.Int64:
			if !signedOk {
				return zero, fmt.Errorf("starlark int didn't fit in go int64")
			}
			result.SetInt(signedValue)
		case reflect.Int32:
			if !signedOk || signedValue > math.MaxInt32 || signedValue < math.MinInt32 {
				return zero, fmt.Errorf("starlark int didn't fit in go int32")
			}
			result.SetInt(signedValue)
		case reflect.Int16:
			if !signedOk || signedValue > math.MaxInt16 || signedValue < math.MinInt16 {
				return zero, fmt.Errorf("starlark int didn't fit in go int16")
			}
			result.SetInt(signedValue)
		case reflect.Int8:
			if !signedOk || signedValue > math.MaxInt8 || signedValue < math.MinInt8 {
				return zero, fmt.Errorf("starlark int didn't fit in go int8")
			}
			result.SetInt(signedValue)
		case reflect.Int:
			if !signedOk || signedValue > math.MaxInt || signedValue < math.MinInt {
				return zero, fmt.Errorf("starlark int didn't fit in go int32")
			}
			result.SetInt(signedValue)
		case reflect.Uint64:
			if !unsignedOk {
				return zero, fmt.Errorf("starlark int didn't fit in go int64")
			}
			result.SetUint(unsignedValue)
		case reflect.Uint32:
			if !unsignedOk || unsignedValue > math.MaxUint32 {
				return zero, fmt.Errorf("starlark int didn't fit in go int32")
			}
			result.SetUint(unsignedValue)
		case reflect.Uint16:
			if !unsignedOk || unsignedValue > math.MaxUint16 {
				return zero, fmt.Errorf("starlark int didn't fit in go int16")
			}
			result.SetUint(unsignedValue)
		case reflect.Uint8:
			if !unsignedOk || unsignedValue > math.MaxUint8 {
				return zero, fmt.Errorf("starlark int didn't fit in go int8")
			}
			result.SetUint(unsignedValue)
		case reflect.Uint:
			if !unsignedOk || unsignedValue > math.MaxUint {
				return zero, fmt.Errorf("starlark int didn't fit in go int32")
			}
			result.SetUint(unsignedValue)
		default:
			return zero, fmt.Errorf("expected integer type, got %s", result.Type().Kind().String())
		}
	case starlark.Float:
		f := float64(v)
		switch result.Type().Kind() {
		case reflect.Float64:
			result.SetFloat(f)
		case reflect.Float32:
			if f > math.MaxFloat32 || f < -math.MaxFloat32 {
				return zero, fmt.Errorf("starlark float didn't fit in go float32")
			}
			result.SetFloat(f)
		default:
			return zero, fmt.Errorf("expected float type, got %s", result.Type().Kind().String())
		}
	case starlark.Bool:
		if result.Type().Kind() != reflect.Bool {
			return zero, fmt.Errorf("expected bool, got %s", result.Type().Kind().String())
		}
		result.SetBool(bool(v))
	case starlark.Tuple:
		if result.Type().Kind() != reflect.Slice {
			return zero, fmt.Errorf("expected slice, got %s", result.Type().Kind().String())
		}
		elemType := result.Type().Elem()
		// TODO: Add this grow call when we're on go 1.20
		//result.Grow(v.Len())
		for i := 0; i < v.Len(); i++ {
			elem, err := UnmarshalReflect(v.Index(i), elemType)
			if err != nil {
				return zero, err
			}
			result = reflect.Append(result, elem)
		}
	case *starlark.List:
		if result.Type().Kind() != reflect.Slice {
			return zero, fmt.Errorf("expected slice, got %s", result.Type().Kind().String())
		}
		elemType := result.Type().Elem()
		// TODO: Add this grow call when we're on go 1.20
		//result.Grow(v.Len())
		for i := 0; i < v.Len(); i++ {
			elem, err := UnmarshalReflect(v.Index(i), elemType)
			if err != nil {
				return zero, err
			}
			result = reflect.Append(result, elem)
		}
	case *starlark.Dict:
		if result.Type().Kind() != reflect.Map {
			return zero, fmt.Errorf("expected map, got %s", result.Type().Kind().String())
		}
		keyType := result.Type().Key()
		valueType := result.Type().Elem()
		for _, pair := range v.Items() {
			key := pair.Index(0)
			value := pair.Index(1)

			unmarshalledKey, err := UnmarshalReflect(key, keyType)
			if err != nil {
				return zero, err
			}
			unmarshalledValue, err := UnmarshalReflect(value, valueType)
			if err != nil {
				return zero, err
			}

			result.SetMapIndex(unmarshalledKey, unmarshalledValue)
		}
	case *starlarkstruct.Struct:
		if result.Type().Kind() != reflect.Struct {
			return zero, fmt.Errorf("expected struct, got %s", result.Type().Kind().String())
		}
		if result.NumField() != len(v.AttrNames()) {
			return zero, fmt.Errorf("starlark struct and go struct have different number of fields (%d and %d)", len(v.AttrNames()), result.NumField())
		}
		for _, attrName := range v.AttrNames() {
			attr, err := v.Attr(attrName)
			if err != nil {
				return zero, err
			}

			// TODO: this should probably support tags to rename the field
			resultField := result.FieldByName(attrName)
			if resultField == (reflect.Value{}) {
				return zero, fmt.Errorf("struct should've had field %s, but was not found", attrName)
			}
			// This hack allows us to change unexported fields
			resultField = reflect.NewAt(resultField.Type(), unsafe.Pointer(resultField.UnsafeAddr())).Elem()
			x, err := UnmarshalReflect(attr, resultField.Type())
			if err != nil {
				return zero, err
			}
			resultField.Set(x)
		}
	default:
		return zero, fmt.Errorf("unimplemented starlark type: %s", value.Type())
	}

	return result, nil
}
