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

func minApiForArch(ctx android.BaseModuleContext) android.ApiLevel {
	arch := ctx.Arch().ArchType
	minVersion := ctx.Config().MinSupportedSdkVersion()
	firstArchVersions := map[android.ArchType]android.ApiLevel{
		android.Arm:    minVersion,
		android.Arm64:  android.FirstLp64Version,
		android.X86:    minVersion,
		android.X86_64: android.FirstLp64Version,
	}

	firstArchVersion, ok := firstArchVersions[arch]
	if !ok {
		panic(fmt.Errorf("Arch %q not found in firstArchVersions", arch))
	}
	return firstArchVersion
}

func nativeApiLevelFromUser(ctx android.BaseModuleContext,
	raw string) (*android.ApiLevel, error) {

	min := minApiForArch(ctx)
	if raw == "minimum" {
		return android.ApiLevelFromUser(ctx, min.String())
	}

	value, err := android.ApiLevelFromUser(ctx, raw)
	if err != nil {
		return nil, err
	}

	if value.LessThan(min) {
		return &min, nil
	}

	return value, nil
}

func nativeApiLevelFromUserWithDefault(ctx android.BaseModuleContext,
	raw string, defaultValue string) (*android.ApiLevel, error) {
	if raw == "" {
		raw = defaultValue
	}
	return nativeApiLevelFromUser(ctx, raw)
}

func nativeApiLevelOrPanic(ctx android.BaseModuleContext,
	raw string) android.ApiLevel {
	value, err := nativeApiLevelFromUser(ctx, raw)
	if err != nil {
		panic(err.Error())
	}
	return *value
}
