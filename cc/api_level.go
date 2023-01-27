// Copyright 2020 Google Inc. All rights reserved.
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

	"android/soong/android"
)

var (
	apiLevelExportedVars = android.NewExportedVariables(pctx)
	minApiForArchMap     = map[android.ArchType]android.ApiLevel{
		android.Arm64:   android.FirstLp64Version,
		android.X86_64:  android.FirstLp64Version,
		android.Riscv64: android.FutureApiLevel,
	}
)

func init() {
	apiLevelExportedVars.ExportStringDict(
		"MinApiForArch",
		map[string]string{
			android.Arm.String():     "${MinApiForArch_Arm}",
			android.X86.String():     "${MinApiForArch_X86}",
			android.Arm64.String():   minApiForArchMap[android.Arm64].String(),
			android.X86_64.String():  minApiForArchMap[android.X86_64].String(),
			android.Riscv64.String(): minApiForArchMap[android.Riscv64].String(),
		},
	)
	apiLevelExportedVars.ExportVariableConfigMethod("MinApiForArch_Arm", func(config android.Config) string {
		return config.MinSupportedSdkVersion().String()
	})
	apiLevelExportedVars.ExportVariableConfigMethod("MinApiForArch_X86", func(config android.Config) string {
		return config.MinSupportedSdkVersion().String()
	})
}

func minApiForArch(ctx android.EarlyModuleContext,
	arch android.ArchType) android.ApiLevel {

	switch arch {
	case android.Arm, android.X86:
		return ctx.Config().MinSupportedSdkVersion()
	default:
		if level, ok := minApiForArchMap[arch]; ok {
			return level
		}
		panic(fmt.Errorf("Unknown arch %q", arch))
	}
}

func nativeApiLevelFromUser(ctx android.BaseModuleContext,
	raw string) (android.ApiLevel, error) {

	min := minApiForArch(ctx, ctx.Arch().ArchType)
	if raw == "minimum" {
		return min, nil
	}

	value, err := android.ApiLevelFromUser(ctx, raw)
	if err != nil {
		return android.NoneApiLevel, err
	}

	if value.LessThan(min) {
		return min, nil
	}

	return value, nil
}

func nativeApiLevelOrPanic(ctx android.BaseModuleContext,
	raw string) android.ApiLevel {
	value, err := nativeApiLevelFromUser(ctx, raw)
	if err != nil {
		panic(err.Error())
	}
	return value
}

func BazelCcApiLevelToolchainVars(config android.Config) string {
	return android.BazelToolchainVars(config, apiLevelExportedVars)
}
