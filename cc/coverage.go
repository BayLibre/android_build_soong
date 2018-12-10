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
	"sync"

	"android/soong/android"
	"android/soong/genrule"
)

var (
	coverageModuleLock sync.Mutex
)

type CoverageProperties struct {
	Native_coverage *bool

	CoverageEnabled bool `blueprint:"mutated"`

	LinkCoverage bool   `blueprint:"mutated"`
	VariantName  string `blueprint:"mutated"`
}

type coverage struct {
	Properties CoverageProperties
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
		cov.Properties.LinkCoverage = true
	}

	// Even if we don't have coverage enabled, if any of our object files were compiled
	// with coverage, then we need to add --coverage to our ldflags.
	if !cov.Properties.LinkCoverage {
		if ctx.static() && !ctx.staticBinary() {
			// For static libraries, the only thing that changes our object files
			// are included whole static libraries, so check to see if any of
			// those have coverage enabled.
			ctx.VisitDirectDepsWithTag(wholeStaticDepTag, func(m android.Module) {
				if cc, ok := m.(*Module); ok && cc.coverage != nil {
					if cc.coverage.Properties.LinkCoverage {
						cov.Properties.LinkCoverage = true
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

				if cc.coverage.Properties.LinkCoverage {
					cov.Properties.LinkCoverage = true
				}
			})
		}
	}

	if cov.Properties.LinkCoverage {
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
		var enabled bool
		var coveragePresent bool
		var optOut bool

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
			coveragePresent = true
			if c.coverage.Properties.Native_coverage != nil {
				enabled = *c.coverage.Properties.Native_coverage
				optOut = !enabled
			} else {
				enabled = mctx.DeviceConfig().CoverageEnabledForPath(mctx.ModuleDir())
			}

			if String(c.Properties.Sdk_version) != "current" {
				if fromApi, err := strconv.Atoi(String(c.Properties.Sdk_version)); err == nil && fromApi < 23 {
					enabled = false
				}
			}
		}

		variations := []string{""}

		if coveragePresent && !optOut {
			if enabled {
				variations = append(variations, "cov")
			} else {
				variations = append(variations, "deps")
			}
		}

		m := mctx.CreateVariations(variations...)

		// Setup the non-coverage version.  Set HideFromMake and
		// PreventInstall if set in the original module, or if there's
		// a coverage-enabled variant.
		if coveragePresent {
			m[0].(*Module).coverage.Properties.CoverageEnabled = false
			m[0].(*Module).coverage.Properties.VariantName = variations[0]
		}
		m[0].(*Module).Properties.HideFromMake = hideFromMake || len(m) > 1
		m[0].(*Module).Properties.PreventInstall = preventInstall || len(m) > 1

		if coveragePresent && !optOut {
			m[1].(*Module).coverage.Properties.CoverageEnabled = enabled
			m[1].(*Module).coverage.Properties.VariantName = variations[1]

			m[1].(*Module).Properties.HideFromMake = hideFromMake
			m[1].(*Module).Properties.PreventInstall = preventInstall

			coverageEnabledModules := coverageEnabledModules(mctx.Config())
			coverageModuleLock.Lock()
			(*coverageEnabledModules)[c.Name()] = variations[len(variations)-1]
			coverageModuleLock.Unlock()
		}
	} else if c, ok := mctx.Module().(*genrule.Module); ok && c.Extra != nil {
		mctx.CreateVariations("")
	}
}

func coverageEnabledModules(config android.Config) *map[string]string {
	return config.Once("coverageEnabledModules", func() interface{} {
		moduleMap := make(map[string]string)
		return &moduleMap
	}).(*map[string]string)
}

func GetCoverageVariation(config android.Config, moduleName string) string {
	coverageModuleLock.Lock()
	val := (*coverageEnabledModules(config))[moduleName]
	coverageModuleLock.Unlock()

	return val
}

func (c *Module) HasCoverage() bool {
	return c.coverage != nil && c.coverage.Properties.VariantName != ""
}
