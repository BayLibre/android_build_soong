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

	"github.com/google/blueprint"
)

func init() {
	RegisterLibraryCmakeComponents(android.InitRegistrationContext)
}

func RegisterLibraryCmakeComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("cc_library_cmake_snapshot", LibraryCmakeSnapshotFactory)
}

type LibraryCmakeSnapshot struct {
	android.ModuleBase

	cmakeFilePath android.WritablePath
}

var (
	ccLibCmakeRule = pctx.StaticRule("ccLibCmakeRule", blueprint.RuleParams{
		Command: `rm -f ${out} && { set -e; ` +
			`echo 'cmake_minimum_required(VERSION 3.18)';` +
			`echo "project(\"${name}\" CXX)";` +
			`} >> ${out}`,
		Description: ": ${out}",
	}, "name")
)

func (m *LibraryCmakeSnapshot) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	m.cmakeFilePath = android.PathForModuleOut(ctx, "CMakeLists.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:   ccLibCmakeRule,
		Output: m.cmakeFilePath,
		Args: map[string]string{
			"name": m.Name(),
		},
	})
}

func LibraryCmakeSnapshotFactory() android.Module {
	module := &LibraryCmakeSnapshot{}
	android.InitAndroidModule(module)
	return module
}
