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
	registerBuildNumberFilegroupBuildComponents(InitRegistrationContext)
}

func registerBuildNumberFilegroupBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("build_number_filegroup", buildNumberFilegroupFactory)
}

// filegroup contains a list of files that are referenced by other modules
// properties (such as "srcs") using the syntax ":<name>". filegroup are
// also be used to export files across package boundaries.
func buildNumberFilegroupFactory() Module {
	module := &buildNumberFilegroup{}
	InitAndroidModule(module)
	return module
}

type buildNumberFilegroup struct {
	ModuleBase
	buildNumberFile Path
}

func (b *buildNumberFilegroup) GenerateAndroidBuildActions(ctx ModuleContext) {
	if ctx.ModuleDir() != "build/soong" {
		ctx.ModuleErrorf("a build_number_filegroup can only be defined in build/soong. This is a tightly controlled module, please use libbuildversion instead")
	}
	b.buildNumberFile = ctx.Config().BuildNumberFile(ctx)
}

func (b *buildNumberFilegroup) Srcs() Paths {
	return []Path{b.buildNumberFile}
}
