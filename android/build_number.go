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

package android

func init() {
	ctx := InitRegistrationContext
	ctx.RegisterModuleType("build_number_file", buildNumberFileFactory)
}

// build_number_file exposes a file that contains the current build number. Dependencies
// on this module are special-cased to always be order-only dependencies, meaning that
// the module will not be rebuilt purely based on build number changes.
func buildNumberFileFactory() Module {
	module := &buildNumberModule{}
	InitAndroidModule(module)
	return module
}

type buildNumberModule struct {
	ModuleBase
	buildNumberFile Paths
}

func (m *buildNumberModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	m.HideFromMake()
	if ctx.ModuleDir() != "build/soong" || ctx.ModuleName() != "build_number_file" {
		// We're going to lock down access to this module via visibility,
		// so we don't want to allow creating other modules that might circumvent
		// the visibility.
		ctx.ModuleErrorf("there can only be one build_number_file module")
	}
	m.buildNumberFile = []Path{ctx.Config().BuildNumberFile(ctx)}
	ctx.blueprintModuleContext().AddAlwaysOrderOnlyDeps(ctx.Config().BuildNumberFile(ctx).String())
	ctx.SetOutputFiles(m.buildNumberFile, "")
}
