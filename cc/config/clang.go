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

package config

import (
	"sort"
	"strings"
)

// A version number increased whenever a Clang update uncovers new diagnostic
// issues. This is used to determine which diagnostics enabled by modules should
// be disabled for new toolchains. Toolchain updates are often slowed down by
// needeing to clean up warnings in modules. Disabling those warnings in a
// central location allows the update to happen more quickly and for module
// owners to be able to fix their code at their own pace.
//
// Note that the exception to this behavior is that any diagnostics in the
// NoOverride lists below must still be fixed.
const CurrentClangDiagnosticVersion = 0

// Each Clang diagnostic version has an entry in this list that defines
// additional arguments to be passed to Clang for any modules targeting an older
// diagnostic version than CurrentClangDiagnosticVersion.
var ClangDisableWarningsLists = map[int][]string{
	0: {},
}

// Each Clang diagnostic version has an entry in this list that defines
// clang-tidy checks to be disabled arguments for any modules targeting an older
// diagnostic version than CurrentClangDiagnosticVersion.
var ClangTidyDisablesChecksLists = map[int][]string{
	0: {},
}

// Cflags that should be filtered out when compiling with clang
var ClangUnknownCflags = sorted([]string{
	"-finline-functions",
	"-finline-limit=64",
	"-fno-canonical-system-headers",
	"-Wno-clobbered",
	"-fno-devirtualize",
	"-fno-tree-sra",
	"-fprefetch-loop-arrays",
	"-funswitch-loops",
	"-Werror=unused-but-set-parameter",
	"-Werror=unused-but-set-variable",
	"-Wmaybe-uninitialized",
	"-Wno-error=clobbered",
	"-Wno-error=maybe-uninitialized",
	"-Wno-error=unused-but-set-parameter",
	"-Wno-error=unused-but-set-variable",
	"-Wno-extended-offsetof",
	"-Wno-free-nonheap-object",
	"-Wno-literal-suffix",
	"-Wno-maybe-uninitialized",
	"-Wno-old-style-declaration",
	"-Wno-psabi",
	"-Wno-unused-but-set-parameter",
	"-Wno-unused-but-set-variable",
	"-Wno-unused-local-typedefs",
	"-Wunused-but-set-parameter",
	"-Wunused-but-set-variable",
	"-fdiagnostics-color",
	// http://b/153759688
	"-fuse-init-array",

	// arm + arm64
	"-fgcse-after-reload",
	"-frerun-cse-after-loop",
	"-frename-registers",
	"-fno-strict-volatile-bitfields",

	// arm + arm64
	"-fno-align-jumps",

	// arm
	"-mthumb-interwork",
	"-fno-builtin-sin",
	"-fno-caller-saves",
	"-fno-early-inlining",
	"-fno-move-loop-invariants",
	"-fno-partial-inlining",
	"-fno-tree-copy-prop",
	"-fno-tree-loop-optimize",

	// x86 + x86_64
	"-finline-limit=300",
	"-fno-inline-functions-called-once",
	"-mfpmath=sse",
	"-mbionic",

	// windows
	"--enable-stdcall-fixup",
})

// Ldflags that should be filtered out when linking with clang lld
var ClangUnknownLldflags = sorted([]string{
	"-Wl,--fix-cortex-a8",
	"-Wl,--no-fix-cortex-a8",
})

var ClangLibToolingUnknownCflags = sorted([]string{})

func init() {
	pctx.StaticVariable("ClangExtraCflags", strings.Join([]string{
		"-D__compiler_offsetof=__builtin_offsetof",

		// Emit address-significance table which allows linker to perform safe ICF. Clang does
		// not emit the table by default on Android since NDK still uses GNU binutils.
		"-faddrsig",

		// Help catch common 32/64-bit errors.
		"-Werror=int-conversion",

		// Enable the new pass manager.
		"-fexperimental-new-pass-manager",

		// Disable overly aggressive warning for macros defined with a leading underscore
		// This happens in AndroidConfig.h, which is included nearly everywhere.
		// TODO: can we remove this now?
		"-Wno-reserved-id-macro",

		// Workaround for ccache with clang.
		// See http://petereisentraut.blogspot.com/2011/05/ccache-and-clang.html.
		"-Wno-unused-command-line-argument",

		// Force clang to always output color diagnostics. Ninja will strip the ANSI
		// color codes if it is not running in a terminal.
		"-fcolor-diagnostics",

		// Warnings from clang-7.0
		"-Wno-sign-compare",

		// Warnings from clang-8.0
		"-Wno-defaulted-function-deleted",

		// Disable -Winconsistent-missing-override until we can clean up the existing
		// codebase for it.
		"-Wno-inconsistent-missing-override",

		// Warnings from clang-10
		// Nested and array designated initialization is nice to have.
		"-Wno-c99-designator",
	}, " "))

	pctx.StaticVariable("ClangExtraCppflags", strings.Join([]string{
		// -Wimplicit-fallthrough is not enabled by -Wall.
		"-Wimplicit-fallthrough",

		// Enable clang's thread-safety annotations in libcxx.
		"-D_LIBCPP_ENABLE_THREAD_SAFETY_ANNOTATIONS",

		// libc++'s math.h has an #include_next outside of system_headers.
		"-Wno-gnu-include-next",
	}, " "))

	pctx.StaticVariable("ClangExtraTargetCflags", strings.Join([]string{
		"-nostdlibinc",
	}, " "))

	pctx.StaticVariable("ClangExtraNoOverrideCflags", strings.Join([]string{
		"-Werror=address-of-temporary",
		// Bug: http://b/29823425 Disable -Wnull-dereference until the
		// new cases detected by this warning in Clang r271374 are
		// fixed.
		//"-Werror=null-dereference",
		"-Werror=return-type",

		// http://b/72331526 Disable -Wtautological-* until the instances detected by these
		// new warnings are fixed.
		"-Wno-tautological-constant-compare",
		"-Wno-tautological-type-limit-compare",
		// http://b/145210666
		"-Wno-reorder-init-list",
		// http://b/145211066
		"-Wno-implicit-int-float-conversion",
		// New warnings to be fixed after clang-r377782.
		"-Wno-int-in-bool-context",          // http://b/148287349
		"-Wno-sizeof-array-div",             // http://b/148815709
		"-Wno-tautological-overlap-compare", // http://b/148815696
		// New warnings to be fixed after clang-r383902.
		"-Wno-deprecated-copy",                      // http://b/153746672
		"-Wno-range-loop-construct",                 // http://b/153747076
		"-Wno-misleading-indentation",               // http://b/153746954
		"-Wno-zero-as-null-pointer-constant",        // http://b/68236239
		"-Wno-deprecated-anon-enum-enum-conversion", // http://b/153746485
		"-Wno-deprecated-enum-enum-conversion",      // http://b/153746563
		"-Wno-string-compare",                       // http://b/153764102
		"-Wno-enum-enum-conversion",                 // http://b/154138986
		"-Wno-enum-float-conversion",                // http://b/154255917
		"-Wno-pessimizing-move",                     // http://b/154270751
	}, " "))

	// Extra cflags for external third-party projects to disable warnings that
	// are infeasible to fix in all the external projects and their upstream repos.
	pctx.StaticVariable("ClangExtraExternalCflags", strings.Join([]string{
		"-Wno-enum-compare",
		"-Wno-enum-compare-switch",

		// http://b/72331524 Allow null pointer arithmetic until the instances detected by
		// this new warning are fixed.
		"-Wno-null-pointer-arithmetic",

		// Bug: http://b/29823425 Disable -Wnull-dereference until the
		// new instances detected by this warning are fixed.
		"-Wno-null-dereference",

		// http://b/145211477
		"-Wno-pointer-compare",
		// http://b/145211022
		"-Wno-xor-used-as-pow",
		// http://b/145211022
		"-Wno-final-dtor-non-final-class",
	}, " "))
}

func ClangRewriteCflags(cflags []string, diagnosticsVersion int) []string {
	return clangDisableDiagnostics(ClangFilterUnknownCflags(cflags), diagnosticsVersion)
}

func filterStringList(list []string, filterOut []string) []string {
	ret := make([]string, 0, len(list))
	for _, e := range list {
		if !inListSorted(e, filterOut) {
			ret = append(ret, e)
		}
	}

	return ret
}

func ClangFilterUnknownCflags(cflags []string) []string {
	return filterStringList(cflags, ClangUnknownCflags)
}

// Disables any diagnostics that would break the build with a newer version of
// the compiler than this module was last made warning-free for. Module owners
// should increase the clang_diagnostics_version of their module to match
// CurrentClangDiagnosticVersion and clean up new diagnostics.
func clangDisableDiagnostics(cflags []string, diagnosticsVersion int) []string {
	for version := diagnosticsVersion + 1; version <= CurrentClangDiagnosticVersion; version++ {
		cflags = append(cflags, ClangDisableWarningsLists[version]...)
	}
	return cflags
}

func clangTidyNegateChecks(checks []string) []string {
	ret := make([]string, 0, len(checks))
	for _, c := range checks {
		if strings.HasPrefix(c, "-") {
			ret = append(ret, c)
		} else {
			ret = append(ret, "-"+c)
		}
	}
	return ret
}

func ClangRewriteTidyChecks(checks []string, diagnosticsVersion int) []string {
	disableChecks := []string{}
	for version := diagnosticsVersion + 1; version <= CurrentClangDiagnosticVersion; version++ {
		disableChecks = append(disableChecks, ClangTidyDisablesChecksLists[version]...)
	}

	checks = append(checks, clangTidyNegateChecks(disableChecks)...)
	return filterStringList(checks, disableChecks)
}

func ClangFilterUnknownLldflags(lldflags []string) []string {
	return filterStringList(lldflags, ClangUnknownLldflags)
}

func inListSorted(s string, list []string) bool {
	for _, l := range list {
		if s == l {
			return true
		} else if s < l {
			return false
		}
	}
	return false
}

func sorted(list []string) []string {
	sort.Strings(list)
	return list
}
