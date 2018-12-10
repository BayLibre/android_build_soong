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
	"strconv"

	"android/soong/android"
)

type CoverageProperties struct {
	Native_coverage *bool

	CoverageEnabled bool `blueprint:"mutated"`
}

type coverage struct {
	Properties CoverageProperties

	// Whether binaries containing this module need --coverage added to their ldflags
	linkCoverage bool
}

func (cov *coverage) props() []interface{} {
	return []interface{}{&cov.Properties}
}

func (cov *coverage) begin(ctx BaseModuleContext) {}

func (cov *coverage) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (cov *coverage) flags(ctx ModuleContext, flags Flags) Flags {
	if !ctx.DeviceConfig().NativeCoverageEnabled() {
		return flags
	}

	if cov.Properties.CoverageEnabled {
		flags.Coverage = true
		flags.GlobalFlags = append(flags.GlobalFlags, "--coverage", "-O0")
		cov.linkCoverage = true
	}

	// Even if we don't have coverage enabled, if any of our object files were compiled
	// with coverage, then we need to add --coverage to our ldflags.
	if !cov.linkCoverage {
		if ctx.static() && !ctx.staticBinary() {
			// For static libraries, the only thing that changes our object files
			// are included whole static libraries, so check to see if any of
			// those have coverage enabled.
			ctx.VisitDirectDepsWithTag(wholeStaticDepTag, func(m android.Module) {
				if cc, ok := m.(*Module); ok && cc.coverage != nil {
					if cc.coverage.linkCoverage {
						cov.linkCoverage = true
					}
				}
			})
		} else {
			// For executables and shared libraries, we need to check all of
			// our static dependencies.
			ctx.VisitDirectDeps(func(m android.Module) {
				cc, ok := m.(*Module)
				if !ok || cc.coverage == nil {
					return
				}

				if static, ok := cc.linker.(libraryInterface); !ok || !static.static() {
					return
				}

				if cc.coverage.linkCoverage {
					cov.linkCoverage = true
				}
			})
		}
	}

	if cov.linkCoverage {
		flags.LdFlags = append(flags.LdFlags, "--coverage")
	}

	return flags
}

func coverageMutator(mctx android.BottomUpMutatorContext) {
	// Coverage is disabled globally
	if !mctx.DeviceConfig().NativeCoverageEnabled() {
		return
	}

	if c, ok := mctx.Module().(*Module); ok {
		var needCoverageVariant bool
		var needCoverageBuild bool

		hideFromMake := c.Properties.HideFromMake
		preventInstall := c.Properties.PreventInstall

		if mctx.Host() {
			// TODO(dwillemsen): because of -nodefaultlibs, we must depend on libclang_rt.profile-*.a
			// Just turn off for now.
		} else if c.useVndk() || c.hasVendorVariant() {
			// Do not enable coverage for VNDK libraries
		} else if c.isNDKStubLibrary() {
			// Do not enable coverage for NDK stub libraries
		} else if c.coverage != nil {
			if c.coverage.Properties.Native_coverage == nil {
				// Native_coverage property defaults to true.  Enable coverage if this property is absent
				needCoverageVariant = true
			} else {
				// Native_coverage property present.  Use its value.
				needCoverageVariant = *c.coverage.Properties.Native_coverage
			}

			if sdk_version := String(c.Properties.Sdk_version); sdk_version != "current" {
				// Native coverage is not supported for SDK versions < 23
				if fromApi, err := strconv.Atoi(sdk_version); err == nil && fromApi < 23 {
					needCoverageVariant = false
				}
			}

			if needCoverageVariant {
				// Coverage variant is actually built with coverage if enabled for its module path
				needCoverageBuild = mctx.DeviceConfig().CoverageEnabledForPath(mctx.ModuleDir())
			}
		}

		if needCoverageVariant {
			m := mctx.CreateVariations("", "cov")

			// Setup the non-coverage version.  Set HideFromMake and
			// PreventInstall if set in the original module, or if there's
			// a coverage-enabled variant.
			if c.coverage != nil {
				m[0].(*Module).coverage.Properties.CoverageEnabled = false
			}
			m[0].(*Module).Properties.HideFromMake = true
			m[0].(*Module).Properties.PreventInstall = true

			m[1].(*Module).coverage.Properties.CoverageEnabled = needCoverageBuild
			m[1].(*Module).Properties.HideFromMake = hideFromMake
			m[1].(*Module).Properties.PreventInstall = preventInstall
		}
	}
}
