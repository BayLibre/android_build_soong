// Copyright 2024 Google Inc. All rights reserved.
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

package android

import (
	"fmt"
	"strconv"
)

func getSdkVersionMapVendorApiLevels() map[string]int {
	return map[string]int{
		"202404": 35,
		"202504": 36,
	}
}

func GetSdkVersionForVendorApiLevel(vendorApiLevel string) (ApiLevel, error) {
	vendorApiLevelInt, err := strconv.Atoi(vendorApiLevel)
	if err != nil {
		return NoneApiLevel, fmt.Errorf("The vendor API level %q must be able to be parsed as an integer", vendorApiLevel)
	}
	if vendorApiLevelInt < 35 {
		return uncheckedFinalApiLevel(vendorApiLevelInt), nil
	}

	sdkVersionMap := getSdkVersionMapVendorApiLevels()
	if sdkInt, ok := sdkVersionMap[vendorApiLevel]; ok {
		return uncheckedFinalApiLevel(sdkInt), nil
	}
	return NoneApiLevel, fmt.Errorf("Unknown vendor API level %q. Requires updating the map in vendor_api_level.go?", vendorApiLevel)
}
