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

package fsgen

import (
	"slices"

	"android/soong/android"
)

// AdditionalDeps are added according to the cuttlefish system image bp.
var additionalDeps = []string{
	"android_vintf_manifest",
	"com.android.apex.cts.shim.v1_prebuilt",
	"dex_bootjars",
	"framework_compatibility_matrix.device.xml",
	"idc_data",
	"init.environ.rc-soong",
	"keychars_data",
	"keylayout_data",
	"libclang_rt.asan",
	"libcompiler_rt",
	"libdmabufheap",
	"libgsi",
	"llndk.libraries.txt",
	"logpersist.start",
	"preloaded-classes",
	"public.libraries.android.txt",
	"update_engine_sideload",
}

func init() {
	ctx := android.InitRegistrationContext
	ctx.RegisterParallelSingletonType("product_config_to_bp_singleton", productConfigToBpSingletonFactory)
}

// productConfigToBpSingleton generates a bp file from make-based product config
func productConfigToBpSingletonFactory() android.Singleton {
	return &productConfigToBpSingleton{}
}

type productConfigToBpSingleton struct{}

func (s *productConfigToBpSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	content := s.generateSystemImage(ctx)
	generatedBp := android.PathForOutput(ctx, "soong_generated_product_config.bp")
	android.WriteFileRule(ctx, generatedBp, content)
	ctx.Phony("product_config_to_bp", generatedBp)
}

func (s *productConfigToBpSingleton) generateSystemImage(ctx android.SingletonContext) string {
	partitionVars := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse
	depCandidates := slices.Concat(partitionVars.ProductPackages, partitionVars.ProductPackagesDebug)

	ctx.VisitAllModules(func(m android.Module) {
		if slices.Contains(depCandidates, m.Name()) {
			if !installInSystem(ctx, m) {
				return
			}
			// TODO: categorize the dependencies to deps, multilib,...
			// fmt.Println(m.Name())
		}
		return
	})

	return ""
}

func installInSystem(ctx android.SingletonContext, m android.Module) bool {
	return m.PartitionTag(ctx.DeviceConfig()) == "system" && !m.InstallInData() &&
		!m.InstallInTestcases() && !m.InstallInSanitizerDir() && !m.InstallInVendorRamdisk() &&
		!m.InstallInDebugRamdisk() && !m.InstallInRecovery() && !m.InstallInOdm() &&
		!m.InstallInVendor()
}
