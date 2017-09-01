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
	"fmt"
	"strings"

	"android/soong/android"
)

var (
	// Add flags to ignore warnings that profiles are old or missing for
	// some functions
	profileUseOtherFlags = []string{}
)

const pgoProfileProject = "toolchain/pgo-profiles"

const profileInstrumentFlag = "-fprofile-generate"
const profileSamplingFlag = "-gline-tables-only"
const profileUseInstrumentFormat = "-fprofile-use=%s"
const profileUseSamplingFormat = "-fprofile-sample-use=%s"

type PgoProperties struct {
	Pgo struct {
		Profile_kind *string
		Profile_file *string `android:"arch_variant"`
		Benchmarks   []string
	} `android:"arch_variant"`

	PgoPresent          bool `blueprint:"mutated"`
	IsInstrumenting     bool `blueprint:"mutated"`
	ShouldProfileModule bool `blueprint:"mutated"`
}

type pgo struct {
	Properties PgoProperties
}

func (pgo *pgo) props() []interface{} {
	return []interface{}{&pgo.Properties}
}

func (pgo *pgo) profileGatherFlags(ctx ModuleContext) string {
	switch *pgo.Properties.Pgo.Profile_kind {
	case "instrumentation":
		return profileInstrumentFlag
	case "sampling":
		return profileSamplingFlag
	default:
		ctx.PropertyErrorf("pgo.profile_kind", "unknown value")
	}
	return ""
}

func (pgo *pgo) profileUseFlag(ctx ModuleContext, file string) string {
	switch *pgo.Properties.Pgo.Profile_kind {
	case "instrumentation":
		return fmt.Sprintf(profileUseInstrumentFormat, file)
	case "sampling":
		return fmt.Sprintf(profileUseSamplingFormat, file)
	default:
		ctx.PropertyErrorf("pgo.profile_kind", "unknown value")
	}
	return ""
}

func (pgo *pgo) profileUseFlags(ctx ModuleContext, file string) []string {
	flags := []string{pgo.profileUseFlag(ctx, file)}
	flags = append(flags, profileUseOtherFlags...)
	return flags
}

func (props *PgoProperties) isPGO(ctx BaseModuleContext) bool {
	kindPresent := props.Pgo.Profile_kind != nil
	filePresent := props.Pgo.Profile_file != nil
	benchmarksPresent := len(props.Pgo.Benchmarks) > 0

	// If all three properties are absent, PGO is OFF for this module
	if !kindPresent && !filePresent && !benchmarksPresent {
		return false
	}

	// If at least one property exists, validate that all properties exist
	if !kindPresent || !filePresent || !benchmarksPresent {
		var missing []string
		if !kindPresent {
			missing = append(missing, "profile_kind property")
		}
		if !filePresent {
			missing = append(missing, "profile_file property")
		}
		if !benchmarksPresent {
			missing = append(missing, "non-empty benchmarks property")
		}
		missingProps := strings.Join(missing, ", ")
		ctx.ModuleErrorf("PGO specification is missing " + missingProps)
	}

	// Validate profile_kind
	if *(props.Pgo.Profile_kind) != "instrumentation" {
		ctx.ModuleErrorf("profile_kind must be \"instrumentation\" (\"sampling\" is not supported yet)")
	}

	return true
}

func getPgoProfilesDir(ctx ModuleContext) android.OptionalPath {
	return android.ExistentPathForSource(ctx, "", pgoProfileProject)
}

func (pgo *pgo) begin(ctx BaseModuleContext) {
	// TODO Evaluate if we need to support PGO for host modules
	if ctx.Host() {
		return
	}

	// Check if PGO is needed for this module
	pgo.Properties.PgoPresent = pgo.Properties.isPGO(ctx)

	// profileInstrumentFlag (-fprofile-generate) should be passed to link
	// step for *all* modules if ANDROID_PGO_INSTRUMENT contains a
	// benchmark that needs instrumentation.  If this flag is specified,
	// the Clang driver adds the libclang_rt.profile libraries to the
	// linker.  Adding these flags will be a no-op if none of the objects
	// being linked together were themselves compiled with
	// -fprofile-generate
	//
	// TODO When sampling is supported, this flag should be set only when
	// any of the benchmarks specified uses instrumentation.
	pgoBenchmarks := ctx.AConfig().Getenv("ANDROID_PGO_INSTRUMENT")
	pgo.Properties.IsInstrumenting = pgoBenchmarks != ""

	if !pgo.Properties.IsInstrumenting || !pgo.Properties.PgoPresent {
		return
	}

	// This module should be instrumented if ANDROID_PGO_INSTRUMENT is set
	// and includes a benchmark listed for this module
	//
	// TODO Validate that each benchmark instruments at least one module
	pgo.Properties.ShouldProfileModule = false
	pgoBenchmarksMap := make(map[string]bool)
	for _, b := range strings.Split(pgoBenchmarks, ",") {
		pgoBenchmarksMap[b] = true
	}

	for _, b := range pgo.Properties.Pgo.Benchmarks {
		if pgoBenchmarksMap[b] == true {
			pgo.Properties.ShouldProfileModule = true
			break
		}
	}
}

func (pgo *pgo) flags(ctx ModuleContext, flags Flags) Flags {
	if ctx.Host() {
		return flags
	}

	props := pgo.Properties

	// Add flags to profile this module based on its profile_kind
	if props.ShouldProfileModule {
		profileGatherFlags := pgo.profileGatherFlags(ctx)
		flags.LdFlags = append(flags.LdFlags, profileGatherFlags)
		flags.CFlags = append(flags.CFlags, profileGatherFlags)
		return flags
	}

	// If *some* module needs to be instrumented, append flags for
	// instrumentation profiles to link steps of *all* modules.
	if props.IsInstrumenting {
		flags.LdFlags = append(flags.LdFlags, profileInstrumentFlag)
		return flags
	}

	// If the PGO profiles project is found, and this module has PGO
	// enabled, add flags to use the profile
	if profilesDir := getPgoProfilesDir(ctx); props.PgoPresent && profilesDir.Valid() {
		profileFile := android.PathForSource(ctx, profilesDir.String(), *(props.Pgo.Profile_file))
		profileUseFlags := pgo.profileUseFlags(ctx, profileFile.String())

		flags.CFlags = append(flags.CFlags, profileUseFlags...)
		flags.LdFlags = append(flags.LdFlags, profileUseFlags...)

		// Update CFlagsDeps and LdFlagsDeps so the module is rebuilt
		// if profileFile gets updated
		flags.CFlagsDeps = append(flags.CFlagsDeps, profileFile)
		flags.LdFlagsDeps = append(flags.LdFlagsDeps, profileFile)
	}

	return flags
}
