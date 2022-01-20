// Copyright 2022 The Android Open Source Project
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
	"fmt"

	"android/soong/android"
)

var (
	globalAfdoProfileProjects = []string{
		"vendor/google_data/pgo_profile/sampling/",
		"toolchain/pgo-profiles/sampling/",
	}
)

var afdoProfileProjectsConfigKey = android.NewOnceKey("AfdoProfileProjects")

const afdoFlagFormat = "-Zprofile-sample-use=%s"

type AfdoProperties struct {
	Afdo bool
}

type afdo struct {
	Properties AfdoProperties
}

func (afdo *afdo) props() []interface{} {
	return []interface{}{&afdo.Properties}
}

func (afdo *afdo) flags(ctx ModuleContext, flags Flags, deps PathDeps) (Flags, PathDeps) {
	if ctx.Host() {
		return flags, deps
	}

	if afdo != nil && afdo.Properties.Afdo {
		if profileFile := afdo.Properties.getAfdoProfileFile(ctx, ctx.ModuleName()); profileFile.Valid() {
			profileUseFlag := fmt.Sprintf(afdoFlagFormat, profileFile)
			flags.RustFlags = append(flags.RustFlags, profileUseFlag)

			profileFilePath := profileFile.Path()
			deps.AfdoProfiles = append(deps.AfdoProfiles, profileFilePath)
		}
	}
	return flags, deps
}

// Get list of profile file names, ordered by level of specialisation. For example:
//   1. libfoo_arm64.afdo
//   2. libfoo.afdo
// Add more specialisation as needed.
func getProfileFiles(ctx BaseModuleContext, moduleName string) []string {
	var files []string
	files = append(files, moduleName+"_"+ctx.Arch().ArchType.String()+".afdo")
	files = append(files, moduleName+".afdo")
	return files
}

func getAfdoProfileProjects(config android.DeviceConfig) []string {
	return config.OnceStringSlice(afdoProfileProjectsConfigKey, func() []string {
		return append(globalAfdoProfileProjects, config.AfdoAdditionalProfileDirs()...)
	})
}

func (props *AfdoProperties) getAfdoProfileFile(ctx BaseModuleContext, module string) android.OptionalPath {
	// Test if the profile_file is present in any of the Afdo profile projects
	for _, profileFile := range getProfileFiles(ctx, module) {
		for _, profileProject := range getAfdoProfileProjects(ctx.DeviceConfig()) {
			path := android.ExistentPathForSource(ctx, profileProject, profileFile)
			if path.Valid() {
				return path
			}
		}
	}

	// TODO: Record that this module's profile file is absent

	return android.OptionalPathForPath(nil)
}
