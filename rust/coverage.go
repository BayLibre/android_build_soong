// Copyright 2020 The Android Open Source Project
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

package rust

import (
	"github.com/google/blueprint"

	"android/soong/cc"
)

var CovLibraryName = "libprofile-extras"

type CoverageProperties struct {
	Native_coverage *bool

	NeedCoverageVariant bool `blueprint:"mutated"`
	NeedCoverageBuild   bool `blueprint:"mutated"`

	CoverageEnabled   bool `blueprint:"mutated"`
	IsCoverageVariant bool `blueprint:"mutated"`
}

type coverage struct {
	Properties CoverageProperties

	// Whether binaries containing this module need --coverage added to their ldflags
	linkCoverage bool
}

func (cov *coverage) props() []interface{} {
	return []interface{}{&cov.Properties}
}

func (cov *coverage) deps(ctx DepsContext, deps Deps) Deps {
	if cov.Properties.NeedCoverageVariant {
		ctx.AddVariationDependencies([]blueprint.Variation{
			{Mutator: "link", Variation: "static"},
		}, cc.CoverageDepTag, CovLibraryName)
	}

	return deps
}

func (cov *coverage) flags(ctx ModuleContext, flags Flags, deps PathDeps) (Flags, PathDeps) {

	if !ctx.DeviceConfig().NativeCoverageEnabled() && !ctx.DeviceConfig().ClangCoverageEnabled() {
		return flags, deps
	}

	if cov.Properties.CoverageEnabled {
		flags.Coverage = true
		coverage := ctx.GetDirectDepWithTag(CovLibraryName, cc.CoverageDepTag).(cc.LinkableInterface)
		flags.RustFlags = append(flags.RustFlags,
			"-Z profile", "-g", "-C opt-level=0", "-C link-dead-code", "-Z no-landing-pads")
		flags.LinkFlags = append(flags.LinkFlags,
			"--coverage", "-g", coverage.OutputFile().Path().String(), "-Wl,--wrap,getenv")
		deps.StaticLibs = append(deps.StaticLibs, coverage.OutputFile().Path())
	}

	return flags, deps
}

func (cov *coverage) begin(ctx BaseModuleContext) {
	// Coverage is disabled globally
	if !ctx.DeviceConfig().NativeCoverageEnabled() && !ctx.DeviceConfig().ClangCoverageEnabled() {
		return
	}

	var needCoverageVariant bool
	var needCoverageBuild bool

	if ctx.Host() {
		// Host coverage not yet supported.
	} else if !ctx.nativeCoverage() {
		// Native coverage is not supported for this module type.
	} else {
		// Check if Native_coverage is set to false.  This property defaults to true.
		needCoverageVariant = BoolDefault(cov.Properties.Native_coverage, true)

		if needCoverageVariant {
			// Coverage variant is actually built with coverage if enabled for its module path
			needCoverageBuild = ctx.DeviceConfig().CoverageEnabledForPath(ctx.ModuleDir())
		}
	}

	cov.Properties.NeedCoverageBuild = needCoverageBuild
	cov.Properties.NeedCoverageVariant = needCoverageVariant
}
