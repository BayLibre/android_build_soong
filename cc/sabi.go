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
	"strings"

	"android/soong/android"
	"github.com/google/blueprint"

	"android/soong/cc/config"
)

type SAbiProperties struct {
	CreateSAbiDumps        bool `blueprint:"mutated"`
	ReexportedIncludeFlags []string
}

type sabi struct {
	Properties SAbiProperties
}

func (sabimod *sabi) props() []interface{} {
	return []interface{}{&sabimod.Properties}
}

func (sabimod *sabi) begin(ctx BaseModuleContext) {}

func (sabimod *sabi) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func concatenateSlices(slices [][]string) []string {
	var combinedSlice []string
	for _, slice := range slices {
		combinedSlice = append(combinedSlice, slice...)
	}
	return combinedSlice
}

func splitAndFilterList(list []string, filter []string) (remainder []string, filtered []string) {
	// Some elements of the slice might have multiple flags concatentated by spaces.
	jointString := strings.Join(list, " ")
	splitList := strings.Split(jointString, " ")
	return filterList(splitList, filter)
}

func (sabimod *sabi) flags(ctx ModuleContext, flags Flags) Flags {
	// Assuming that the cflags which clang LibTooling tools cannot
	// understand have not been converted to ninja variables yet.

	cFlagsSlices := [][]string{flags.GlobalFlags,
		flags.SystemIncludeFlags,
		flags.CFlags,
		flags.ConlyFlags,
	}
	flags.ToolingCFlags, _ = splitAndFilterList(concatenateSlices(cFlagsSlices), config.ClangLibToolingUnknownCflags)

	cppFlagsSlices := [][]string{
		flags.GlobalFlags,
		flags.SystemIncludeFlags,
		flags.CFlags,
		flags.CppFlags,
	}
	flags.ToolingCppFlags, _ = splitAndFilterList(concatenateSlices(cppFlagsSlices), config.ClangLibToolingUnknownCflags)
	return flags
}

func sabiDepsMutator(mctx android.TopDownMutatorContext) {
	if c, ok := mctx.Module().(*Module); ok &&
		(Bool(c.Properties.Vendor_available) || (inList(c.Name(), config.LLndkLibraries())) ||
			(c.sabi != nil && c.sabi.Properties.CreateSAbiDumps)) {
		mctx.VisitDirectDeps(func(m blueprint.Module) {
			tag := mctx.OtherModuleDependencyTag(m)
			switch tag {
			case staticDepTag, staticExportDepTag, lateStaticDepTag, wholeStaticDepTag:

				cc, _ := m.(*Module)
				if cc == nil {
					return
				}
				cc.sabi.Properties.CreateSAbiDumps = true
			}
		})
	}
}
