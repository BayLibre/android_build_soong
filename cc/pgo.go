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
	profileInstrumentFlag = "-fprofile-generate"
	profileSampingFlag    = "-gline-tables-only"

	profileUseInstrumentFormat = "-fprofile-use=%s"
	profileUseSamplingFormat   = "-fprofile-sample-use=%s"

	// Add flags to ignore warnings that profiles are old or missing for
	// some functions
	profileUseOtherFlags = []string{}
)

const pgoProfileProject = "toolchain/pgo-profiles"

func init() {
}

type PgoProperties struct {
	Pgo struct {
		Profile_kind *string
		Profile_file *string `android:"arch_variant"`
		Benchmarks   []string
	} `android:"arch_variant"`

	PgoPresent    bool
	ShouldProfile bool
}

type pgo struct {
	Properties PgoProperties
}

func (pgo *pgo) props() []interface{} {
	return []interface{}{&pgo.Properties}
}

func (pgo *pgo) profileGatherFlags() string {
	switch *pgo.Properties.Pgo.Profile_kind {
	case "instrumentation":
		return profileInstrumentFlag
	case "sampling":
		return profileSampingFlag
	}
	panic("Unreachable: unknown Profile_kind")
}

func (pgo *pgo) profileUseFlag(file string) string {
	switch *pgo.Properties.Pgo.Profile_kind {
	case "instrumentation":
		return fmt.Sprintf(profileUseInstrumentFormat, file)
	case "sampling":
		return fmt.Sprintf(profileUseSamplingFormat, file)
	}
	panic("Unreachable: unknown Profile_kind")
}

func (pgo *pgo) profileUseFlags(file string) []string {
	flags := []string{pgo.profileUseFlag(file)}
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

func getPgoProfilesDir(ctx ModuleContext) string {
	p := android.ExistentPathForSource(ctx, "", pgoProfileProject)
	if p.Valid() {
		return p.Path().String()
	}
	return ""
}

func (pgo *pgo) begin(ctx BaseModuleContext) {
	// TODO Evaluate if we need to support PGO for host modules
	if ctx.Host() {
		return
	}
	// Check if PGO is needed for this module
	if pgo.Properties.isPGO(ctx) == false {
		pgo.Properties.PgoPresent = false
		return
	}
	pgo.Properties.PgoPresent = true

	// TODO Validate that each benchmark instruments at least one module

	// This module should be instrumented if ANDROID_PGO_INSTRUMENT is set
	// and includes a benchmark listed for this module
	pgoBenchmarks := ctx.AConfig().Getenv("ANDROID_PGO_INSTRUMENT")

	pgo.Properties.ShouldProfile = false
	pgoBenchmarksMap := make(map[string]bool)
	for _, b := range strings.Split(pgoBenchmarks, ",") {
		pgoBenchmarksMap[b] = true
	}

	for _, b := range pgo.Properties.Pgo.Benchmarks {
		if pgoBenchmarksMap[b] == true {
			pgo.Properties.ShouldProfile = true
			break
		}
	}
}

func (pgo *pgo) flags(ctx ModuleContext, flags Flags) Flags {
	if ctx.Host() {
		return flags
	}
	if pgo.Properties.PgoPresent == false {
		return flags
	}

	// If this module needs to be profiled, append flags for
	// gathering profiles
	if pgo.Properties.ShouldProfile {
		profileGatherFlags := pgo.profileGatherFlags()
		flags.CFlags = append(flags.CFlags, profileGatherFlags)
		flags.LdFlags = append(flags.LdFlags, profileGatherFlags)
		return flags
	}

	// If the PGO profiles project is found, add flags to use the profile
	if profilesDir := getPgoProfilesDir(ctx); profilesDir != "" {
		profileFile := android.PathForSource(ctx, profilesDir, *(pgo.Properties.Pgo.Profile_file))
		profileUseFlags := pgo.profileUseFlags(profileFile.String())

		flags.CFlags = append(flags.CFlags, profileUseFlags...)
		flags.LdFlags = append(flags.LdFlags, profileUseFlags...)

		// Update CFlagsDeps and LdFlagsDeps so the module is rebuilt
		// if profileFile gets udpated
		flags.CFlagsDeps = append(flags.CFlagsDeps, profileFile)
		flags.LdFlagsDeps = append(flags.LdFlagsDeps, profileFile)
	}

	return flags
}
