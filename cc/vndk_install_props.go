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
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	android.RegisterSingletonType("vndk_install_props", VndkInstallPropsSingleton)
}

func VndkInstallPropsSingleton() android.Singleton {
	return &vndkInstallPropsSingleton{}
}

type vndkInstallPropsSingleton struct{}

func isVndkModule(m android.Module) bool {
	if m, ok := m.(*Module); ok {
		lib, is_lib := m.linker.(*libraryDecorator)
		prebuilt_lib, is_prebuilt_lib := m.linker.(*prebuiltLibraryLinker)
		if (is_lib && lib.shared()) || (is_prebuilt_lib && prebuilt_lib.shared()) {
			if m.vndkdep.isVndk() && !m.vndkdep.isVndkExt() {
				return true
			}
		}
	}
	return false
}

func createVndkInstallPropsJson(ctx android.SingletonContext, file android.WritablePath) {
	var vndkInstallPropsMap = make(map[string]interface{})

	ctx.VisitAllModulesIf(isVndkModule,
		func(module android.Module) {
			m := module.(*Module)
			name := strings.TrimPrefix(m.Name(), "prebuilt_")
			if _, ok := vndkInstallPropsMap[name]; !ok {
				vndkInstallPropsMap[name] = m.outputFile.String()
			}
		})

	vndkInstallPropsJson, err := json.MarshalIndent(vndkInstallPropsMap, "", "    ")
	if err != nil {
		ctx.Errorf(err.Error())
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        android.WriteFile,
		Description: "generate " + file.Base(),
		Output:      file,
		Args: map[string]string{
			"content": string(vndkInstallPropsJson[:]),
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
