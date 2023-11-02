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

package cc

import (
	"android/soong/android"
	"fmt"
	"github.com/google/blueprint"
	// "os"
)

func init() {
	RegisterLibraryCmakeComponents(android.InitRegistrationContext)
	android.PreArchMutators(RegisterPreArchMutators)
}

func RegisterLibraryCmakeComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("cc_library_cmake_snapshot", LibraryCmakeSnapshotFactory)
}

func RegisterPreArchMutators(ctx android.RegisterMutatorsContext) {
	ctx.BottomUp("addCMakeLibs", addCMakeLibs).Parallel()
}

type LibraryCmakeSnapshotProperties struct {
	Cc_libs []string
}

type LibraryCmakeSnapshot struct {
	android.ModuleBase

	cmakeFilePath android.WritablePath
	properties    LibraryCmakeSnapshotProperties
	Srcs          []string
}

type LibraryCmakeSnapshotDepTag struct {
	blueprint.BaseDependencyTag
}

func addCMakeLibs(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*LibraryCmakeSnapshot); ok {
		mctx.AddDependency(mctx.Module(), LibraryCmakeSnapshotDepTag{}, m.properties.Cc_libs...)
	}
}

var (
	ccLibCmakeRule = pctx.StaticRule("ccLibCmakeRule", blueprint.RuleParams{
		Command:     `rm -f '${out}' && echo '${cmakeFileBody}' > '${out}'`,
		Description: ": ${out}",
	}, "cmakeFileBody")
)

func (m *LibraryCmakeSnapshot) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	m.cmakeFilePath = android.PathForModuleOut(ctx, "CMakeLists.txt")

	//f, _ := os.Create(m.cmakeFilePath.String())
	//defer f.Close()

	//f.WriteString("cmake_minimum_required(VERSION 3.18)\n")
	//f.WriteString(fmt.Sprintf("project(%s CXX)\n", m.Name()))
	//f.WriteString("set(CMAKE_CXX_STANDARD, 20)\n")

	//ctx.VisitDirectDeps(func(dep android.Module) {
	//    if mdep, ok := dep.(*Module); ok {
	//        f.WriteString(fmt.Sprintf("set(%s_SRC", mdep.Name()))
	//        f.WriteString(")\n")
	//        f.WriteString("\n")
	//        f.WriteString(fmt.Sprintf("add_library(%s, ${%s_SRC})\n", mdep.Name(), mdep.Name()))
	//    }
	//})

	//ctx.PropertyErrorf("cmake", " ============== %q ========", m.Name())
	rule := android.NewRuleBuilder(pctx, ctx)
	rule.Command().Text("rm").Flag("-f").Output(m.cmakeFilePath)
	rule.Command().Text("echo 'cmake_minimum_required(VERSION 3.18)' >> ").Output(m.cmakeFilePath)
	rule.Command().Text(fmt.Sprintf("echo 'project(%s CXX)' >> ", m.Name())).Output(m.cmakeFilePath)
	rule.Build(m.Name(), "CMake snapshot "+m.Name())
	//ctx.Build(pctx, android.BuildParams{
	//	Rule:   ccLibCmakeRule,
	//	Output: m.cmakeFilePath,
	//	Args: map[string]string{
	//		"cmakeFileBody": cmakeFileBody,
	//	},
	//})
}

func LibraryCmakeSnapshotFactory() android.Module {
	module := &LibraryCmakeSnapshot{}
	module.AddProperties(&module.properties)
	android.InitAndroidModule(module)
	return module
}
