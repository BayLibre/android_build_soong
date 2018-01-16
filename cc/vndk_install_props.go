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
	vndkInstallPropsMap = make(map[string]interface{})
)

func parseInstallProps(m *Module, name string) {
	var propertyMap map[string]interface{}
	properties := m.GetProperties()
	for _, propertyStruct := range properties {
		if _, ok := propertyStruct.(*InstallerProperties); ok {
			propertyStructJsonStr, _ := json.Marshal(propertyStruct)
			json.Unmarshal(propertyStructJsonStr, &propertyMap)
			continue
		}
	}
	removeNull(propertyMap)
	vndkInstallPropsMap[name] = propertyMap
}

func removeNull(m map[string]interface{}) {
	for key := range m {
		if m[key] == nil {
			delete(m, key)
		} else if value, ok := m[key].(map[string]interface{}); ok {
			removeNull(value)
			if len(value) == 0 {
				delete(m, key)
			}
		}
	}
}

func init() {
	android.RegisterSingletonType("vndk_install_props", VndkInstallPropsSingleton)
}

func VndkInstallPropsSingleton() android.Singleton {
	return &vndkInstallPropsSingleton{}
}

type vndkInstallPropsSingleton struct{}

func createVndkInstallPropsJson(ctx android.SingletonContext, file android.WritablePath) {
	jsonStr, err := json.MarshalIndent(vndkInstallPropsMap, "", "    ")
	if err != nil {
		ctx.Errorf(err.Error())
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        android.WriteFile,
		Description: "generate " + file.Base(),
		Output:      file,
		Args: map[string]string{
			"content": string(jsonStr[:]),
		},
	})
}

func GetJsonFilePath(ctx android.PathContext) android.WritablePath {
	return android.PathForOutput(ctx, "vndk", "vndk_install_props.json")
}

func (v *vndkInstallPropsSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	vndkInstallPropsJson := GetJsonFilePath(ctx)
	createVndkInstallPropsJson(ctx, vndkInstallPropsJson)

	// Create phony target "vndk_install_props"
	ctx.Build(pctx, android.BuildParams{
		Rule:   blueprint.Phony,
		Output: android.PathForPhony(ctx, "vndk_install_props"),
		Input:  vndkInstallPropsJson,
	})
}
