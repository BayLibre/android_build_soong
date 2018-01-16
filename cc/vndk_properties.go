// Copyright 2017 Google Inc. All rights reserved.
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

package cc

import (
	"encoding/json"

	"github.com/google/blueprint"

	"android/soong/android"
)

var (
	vndkPropertiesMap = make(map[string]interface{})
	vndkJsonFiles     []android.Path
)

func parseVndkProperties(m *Module, name string) {
	var property_map map[string]interface{}
	properties := m.GetProperties()
	for _, propertyStruct := range properties {
		propertyStruct_json, _ := json.Marshal(propertyStruct)
		json.Unmarshal(propertyStruct_json, &property_map)
	}
	remove_null_entries(property_map)
	vndkPropertiesMap[name] = property_map
}

func remove_null_entries(m map[string]interface{}) {
	for key := range m {
		if m[key] == nil {
			delete(m, key)
		} else if value, ok := m[key].(map[string]interface{}); ok {
			remove_null_entries(value)
			if len(value) == 0 {
				delete(m, key)
			}
		}
	}
}

func init() {
	android.RegisterSingletonType("vndk_properties", VndkPropertiesSingleton)
}

func VndkPropertiesSingleton() android.Singleton {
	return &vndkPropertiesSingleton{}
}

type vndkPropertiesSingleton struct{}

func createVndkPropertiesJson(ctx android.SingletonContext, name string) {
	jsonFile := GetJsonFilePath(ctx, name)
	vndkJsonFiles = append(vndkJsonFiles, jsonFile)

	jsonStr, err := json.MarshalIndent(vndkPropertiesMap[name], "", "    ")
	if err != nil {
		ctx.Errorf(err.Error())
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        android.WriteFile,
		Description: "generate " + jsonFile.Base(),
		Output:      jsonFile,
		Args: map[string]string{
			"content": string(jsonStr[:]),
		},
	})
}

func GetJsonFilePath(ctx android.PathContext, name string) android.WritablePath {
	return android.PathForOutput(ctx, "vndk", name+".properties.json")
}

func (v *vndkPropertiesSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	for _, name := range vndkCoreLibraries {
		createVndkPropertiesJson(ctx, name)
	}

	for _, name := range vndkSpLibraries {
		createVndkPropertiesJson(ctx, name)
	}

	// Create phony target "vndk_properties"
	ctx.Build(pctx, android.BuildParams{
		Rule:   blueprint.Phony,
		Output: android.PathForPhony(ctx, "vndk_properties"),
		Inputs: vndkJsonFiles,
	})
}
