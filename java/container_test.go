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

package java

import (
	"android/soong/android"
	"fmt"
	"testing"
)

var prepareForJavaTestWithContainer = android.GroupFixturePreparers(
	prepareForJavaTest,
	android.PrepareForTestWithContainer,
)

// func init() {
// 	android.RegisterContainerMutator(android.InitRegistrationContext)
// }

func TestJavaContainersModuleProperties(t *testing.T) {
	result := android.GroupFixturePreparers(
		prepareForJavaTestWithContainer,
	).RunTestWithBp(t, `
		java_library {
			name: "foo",
			srcs: ["A.java"],
		}
	
		java_library {
			name: "foo_vendor",
			srcs: ["A.java"],
			vendor: true,
			sdk_version: "current",
		}
		java_library {
			name: "foo_soc_specific",
			srcs: ["A.java"],
			soc_specific: true,
			sdk_version: "current",
		}
		java_library {
			name: "foo_product_specific",
			srcs: ["A.java"],
			product_specific: true,
			sdk_version: "current",
		}
	`)

	testcases := []struct {
		moduleName         string
		isSystemContainer  bool
		isVendorContainer  bool
		isProductContainer bool
	}{
		{
			moduleName:         "foo",
			isSystemContainer:  true,
			isVendorContainer:  false,
			isProductContainer: false,
		},
		{
			moduleName:         "foo_vendor",
			isSystemContainer:  false,
			isVendorContainer:  true,
			isProductContainer: false,
		},
		{
			moduleName:         "foo_soc_specific",
			isSystemContainer:  false,
			isVendorContainer:  true,
			isProductContainer: false,
		},
		{
			moduleName:         "foo_product_specific",
			isSystemContainer:  false,
			isVendorContainer:  false,
			isProductContainer: true,
		},
	}

	checkContainerMatch := func(name string, container string, expected bool, actual bool) {
		errorMessage := fmt.Sprintf("module %s container %s expected: %t, actual %t", name, container, expected, actual)
		android.AssertBoolEquals(t, errorMessage, expected, actual)
	}

	for _, c := range testcases {
		m := result.ModuleForTests(c.moduleName, "android_common")
		if lib, ok := m.Module().(android.InstallableModule); ok {
			checkContainerMatch(c.moduleName, "system", c.isSystemContainer, lib.InSystemContainer())
			checkContainerMatch(c.moduleName, "vendor", c.isVendorContainer, lib.InVendorContainer())
			checkContainerMatch(c.moduleName, "product", c.isProductContainer, lib.InProductContainer())
		}
	}
}
