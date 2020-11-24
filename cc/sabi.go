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
	"sync"

	"android/soong/android"
	"android/soong/cc/config"
)

var (
	lsdumpPaths     []string
	lsdumpPathsLock sync.Mutex
)

type SAbiProperties struct {
	// Whether ABI dump should be created for this module.
	// Set by `sabiDepsMutator` if this modules is a direct or indirect dependency of libraries that
	// need ABI check.
	CreateSourceAbiDump bool `blueprint:"mutated"`

	// Include directories that may contain ABI information exported by a library.
	// These directories are passed to the header-abi-dumper.
	ReexportedIncludes []string `blueprint:"mutated"`
}

type sabi struct {
	Properties SAbiProperties

	// Protect `Properties.CreateSourceAbiDump` during parallel `sabiDepsMutator`.
	CreateSourceAbiDumpLock sync.Mutex
}

func (sabi *sabi) props() []interface{} {
	return []interface{}{&sabi.Properties}
}

func (sabi *sabi) begin(ctx BaseModuleContext) {}

func (sabi *sabi) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (sabi *sabi) flags(ctx ModuleContext, flags Flags) Flags {
	// Filter out flags which libTooling don't understand.
	// This is here for legacy reasons and future-proof, in case the version of libTooling and clang
	// diverge.
	flags.Local.ToolingCFlags = config.ClangLibToolingFilterUnknownCflags(flags.Local.CFlags)
	flags.Global.ToolingCFlags = config.ClangLibToolingFilterUnknownCflags(flags.Global.CFlags)
	flags.Local.ToolingCppFlags = config.ClangLibToolingFilterUnknownCflags(flags.Local.CppFlags)
	flags.Global.ToolingCppFlags = config.ClangLibToolingFilterUnknownCflags(flags.Global.CppFlags)
	return flags
}

func (sabi *sabi) markCreateSourceAbiDump() {
	sabi.CreateSourceAbiDumpLock.Lock()
	defer sabi.CreateSourceAbiDumpLock.Unlock()
	sabi.Properties.CreateSourceAbiDump = true
}

// Mark the direct and transitive dependencies of libraries that need ABI check, so that ABI dumps
// of their dependencies would be generated.
func sabiDepsMutator(mctx android.TopDownMutatorContext) {
	if c, ok := mctx.Module().(*Module); ok {
		ctx := &baseModuleContext{
			BaseModuleContext: mctx,
			moduleContextImpl: moduleContextImpl{
				mod: c,
			},
		}
		ctx.ctx = ctx
		if shouldCreateSourceAbiDump(ctx) {
			mctx.VisitDirectDeps(func(m android.Module) {
				if tag, ok := mctx.OtherModuleDependencyTag(m).(libraryDependencyTag); ok && tag.static() {
					if cc, ok := m.(*Module); ok {
						cc.sabi.markCreateSourceAbiDump()
					}
				}
			})
		}
	}
}

// Add an entry to the global list of lsdump. The list is exported to a Make variable by
// `cc.makeVarsProvider`.
func addLsdumpPath(lsdumpPath string) {
	lsdumpPathsLock.Lock()
	defer lsdumpPathsLock.Unlock()
	lsdumpPaths = append(lsdumpPaths, lsdumpPath)
}
