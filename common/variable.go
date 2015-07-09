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

package common

import (
	"reflect"

	"android/soong"

	"github.com/google/blueprint"
)

func init() {
	soong.RegisterEarlyMutator("variable", VariableMutator)
}

type variableProperties struct {
	Product_variables struct {
		Device_uses_logd struct {
			Cflags []string
			Srcs   []string
		}
		Device_uses_dlmalloc struct {
			Cflags []string
			Srcs   []string
		}
		Device_uses_jemalloc struct {
			Cflags            []string
			Srcs              []string
			Whole_static_libs []string
			Include_dirs      []string
		}
	}
}

var zeroVariableProperties variableProperties

var productVariables = []struct {
	property, variant string
	value             func(*AndroidModuleBase) reflect.Value
}{
	{"product_variables.device_uses_logd", "device_uses_logd",
		func(a *AndroidModuleBase) reflect.Value {
			return reflect.ValueOf(a.variableProperties.Product_variables.Device_uses_logd)
		},
	},
	{"product_variables.device_uses_jemalloc", "device_uses_jemalloc",
		func(a *AndroidModuleBase) reflect.Value {
			return reflect.ValueOf(a.variableProperties.Product_variables.Device_uses_jemalloc)
		},
	},
}

func VariableMutator(mctx blueprint.EarlyMutatorContext) {
	var module AndroidModule
	var ok bool
	if module, ok = mctx.Module().(AndroidModule); !ok {
		return
	}

	// TODO: depend on config variable, create variants, propagate variants up tree
	a := module.base()
	for _, v := range productVariables {
		if mctx.ContainsProperty(v.property) {
			a.setVariableProperties(mctx, v.property, v.value(a))
		}
	}
}

func (a *AndroidModuleBase) setVariableProperties(ctx blueprint.EarlyMutatorContext,
	prefix string, value reflect.Value) {

	generalPropertyValues := make([]reflect.Value, len(a.generalProperties))
	for i := range a.generalProperties {
		generalPropertyValues[i] = reflect.ValueOf(a.generalProperties[i]).Elem()
	}

	extendProperties(ctx, "", prefix, generalPropertyValues, value, nil)
}
