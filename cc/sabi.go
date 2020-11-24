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
	// Set by `sabiDepsMutator` if this module is a direct or indirect dependency of libraries that
	// need ABI check.
	ShouldCreateSourceAbiDump bool `blueprint:"mutated"`

	// Include directories that may contain ABI information exported by a library.
	// These directories are passed to the header-abi-dumper.
	ReexportedIncludes []string `blueprint:"mutated"`
}

type sabi struct {
	Properties SAbiProperties
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

// Could be called as a nil receiver.
func (sabi *sabi) shouldCreateSourceAbiDump() bool {
	return sabi != nil && sabi.Properties.ShouldCreateSourceAbiDump
}

// Returns a string that represents the class of the ABI dump.
// Returns an empty string if ABI check is disabled for this library.
func classifySourceAbiDump(ctx android.BaseModuleContext) string {
	m := ctx.Module().(*Module)
	if m.library.headerAbiCheckerExplicitlyDisabled() {
		return ""
	}
	// Return NDK if the library is both NDK and LLNDK.
	if m.IsNdk(ctx.Config()) {
		return "NDK"
	}
	if m.isLlndkPublic(ctx.Config()) {
		return "LLNDK"
	}
	if m.UseVndk() && m.IsVndk() && !m.isVndkPrivate(ctx.Config()) {
		if m.isVndkSp() {
			if m.isVndkExt() {
				return "VNDK-SP-ext"
			} else {
				return "VNDK-SP"
			}
		} else {
			if m.isVndkExt() {
				return "VNDK-ext"
			} else {
				return "VNDK-core"
			}
		}
	}
	if m.library.headerAbiCheckerEnabled() || m.library.hasStubsVariants() {
		return "PLATFORM"
	}
	return ""
}

// Check whether ABI dumps should be created for this module.
// ctx should be wrapping a native library type module.
func shouldCreateSourceAbiDumpForLibrary(ctx android.BaseModuleContext) bool {
	if ctx.Fuchsia() {
		return false
	}

	// Only generate ABI dump for device modules.
	if !ctx.Device() {
		return false
	}

	m := ctx.Module().(*Module)

	// Don't create ABI dump for prebuilts.
	if m.Prebuilt() != nil || m.isSnapshotPrebuilt() {
		return false
	}

	// Coverage builds have extra symbols.
	if m.isCoverageVariant() {
		return false
	}

	// Some sanitizer variants may have different ABI.
	if m.sanitize != nil && !m.sanitize.isVariantOnProductionDevice() {
		return false
	}

	// Don't create ABI dump for stubs.
	if m.isNDKStubLibrary() || m.IsStubs() {
		return false
	}

	// Special case for APEX variants.
	if !ctx.Provider(android.ApexInfoProvider).(android.ApexInfo).IsForPlatform() {
		// Don't create ABI dump if this library is for APEX but isn't exported.
		if !m.HasStubsVariants() {
			return false
		}
		if !m.library.headerAbiCheckerEnabled() {
			// Skip ABI checks if this library is for APEX and did not explicitly enable
			// ABI checks.
			// TODO(b/145608479): ABI checks should be enabled by default. Remove this
			// after evaluating the extra build time.
			return false
		}
	}
	return classifySourceAbiDump(ctx) != ""
}

// Mark the direct and transitive dependencies of libraries that need ABI check, so that ABI dumps
// of their dependencies would be generated.
func sabiDepsMutator(mctx android.TopDownMutatorContext) {
	// Escape hatch to not check any ABI dump.
	if mctx.Config().IsEnvTrue("SKIP_ABI_CHECKS") {
		return
	}
	// Only create ABI dump for native shared libraries and their static library dependencies.
	if m, ok := mctx.Module().(*Module); ok && m.sabi != nil {
		if m.library != nil && m.library.shared() && shouldCreateSourceAbiDumpForLibrary(mctx) {
			// Mark this shared library so that .lsdump for this library can be generated.
			m.sabi.Properties.ShouldCreateSourceAbiDump = true
			// Mark all of its direct and transitive static library dependencies.
			mctx.WalkDeps(func(child, parent android.Module) bool {
				depTag := mctx.OtherModuleDependencyTag(child)
				if libDepTag, ok := depTag.(libraryDependencyTag); ok && libDepTag.static() {
					if c, ok := child.(*Module); ok && c.sabi != nil {
						// Mark this static library so that .sdump for this library can be generated.
						c.sabi.Properties.ShouldCreateSourceAbiDump = true
						// Continue the walk only if child is marked.
						return true
					}
				}
				return false
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
