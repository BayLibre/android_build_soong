// Copyright 2016 Google Inc. All rights reserved.
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
	"github.com/google/blueprint"

	"android/soong/android"
	"strings"
)

func init() {
	android.RegisterModuleType("ndk_compatible_cc_test", ndkCompatibleTestFactory)
}

type ndkCompatibleTestLinker struct {
	*testBinary
}

func (test *ndkCompatibleTestLinker) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps = test.testBinary.linkerDeps(ctx, deps)
	rewriteNdkLibs := func(list []string) ([]string, []string, string) {
		variantLibs := []string{}
		nonvariantLibs := []string{}
		ndk_api := "current"
		for _, entry := range list {
			if strings.Index(entry, "libc.ndk.") == 0 {
				variantLibs = append(variantLibs, "libc.ndk")
				ndk_api = entry[len("libc.ndk."):]
			} else if strings.Index(entry, "libm.ndk.") == 0 {
				variantLibs = append(variantLibs, "libm.ndk")
				ndk_api = entry[len("libm.ndk."):]
			} else {
				nonvariantLibs = append(nonvariantLibs, entry)
			}
		}
		return nonvariantLibs, variantLibs, ndk_api
	}
	variantLateNdkLibs := []string{}
	var ndk_api string
	deps.LateSharedLibs, variantLateNdkLibs, ndk_api = rewriteNdkLibs(deps.LateSharedLibs)
	ctx.AddVariationDependencies([]blueprint.Variation{
		{"ndk_api", ndk_api}, {"link", "shared"}}, ndkLateStubDepTag, variantLateNdkLibs...)
	return deps
}

func ndkCompatibleTestFactory() (blueprint.Module, []interface{}) {
	module := NewTest(android.DeviceSupported)
	test := &ndkCompatibleTestLinker{
		testBinary: module.linker.(*testBinary),
	}
	module.linker = test
	return module.Init()
}
