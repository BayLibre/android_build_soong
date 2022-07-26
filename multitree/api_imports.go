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

package multitree

import (
	"android/soong/android"

	"github.com/google/blueprint"
)

var (
	apiImportNameSuffix = ".apiimport"
)

func init() {
	RegisterApiImportsModule(android.InitRegistrationContext)
}

func RegisterApiImportsModule(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("api_imports", apiImportsFactory)
}

type ApiImports struct {
	android.ModuleBase
	properties apiImportsProperties
}

type apiImportsProperties struct {
	Shared_libs []string
	Header_libs []string
}

func apiImportsFactory() android.Module {
	module := &ApiImports{}
	module.AddProperties(&module.properties)
	android.InitAndroidModule(module)
	return module
}

func (imports *ApiImports) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// ApiImport module does not generate any build actions
}

type ApiImportInfo struct {
	SharedLibs, HeaderLibs map[string]string
}

var ApiImportsProvider = blueprint.NewMutatorProvider(ApiImportInfo{}, "deps")

func (imports *ApiImports) DepsMutator(ctx android.BottomUpMutatorContext) {
	generateNameMapWithSuffix := func(names []string) map[string]string {
		moduleNameMap := make(map[string]string)
		for _, name := range names {
			moduleNameMap[name] = name + ".apiimport"
		}

		return moduleNameMap
	}

	sharedLibs := generateNameMapWithSuffix(imports.properties.Shared_libs)
	headerLibs := generateNameMapWithSuffix(imports.properties.Header_libs)

	ctx.SetProvider(ApiImportsProvider, ApiImportInfo{
		SharedLibs: sharedLibs,
		HeaderLibs: headerLibs,
	})
}

func GetApiImportSuffix() string {
	return apiImportNameSuffix
}
