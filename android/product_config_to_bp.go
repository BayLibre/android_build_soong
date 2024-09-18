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

func init() {
	ctx := InitRegistrationContext
	ctx.RegisterParallelSingletonType("product_config_to_bp_singleton", productConfigToBpSingletonFactory)
}

// productConfigToBpSingleton generates a bp file from make-based product config
func productConfigToBpSingletonFactory() Singleton {
	return &productConfigToBpSingleton{}
}

type productConfigToBpSingleton struct{}

func (s *productConfigToBpSingleton) GenerateBuildActions(ctx SingletonContext) {
	content := s.generateSystemImage(ctx)
	generatedBp := PathForOutput(ctx, "soong_generated_product_config.bp")
	WriteFileRule(ctx, generatedBp, content)
	ctx.Phony("product_config_to_bp", generatedBp)
}

func (s *productConfigToBpSingleton) generateSystemImage(ctx SingletonContext) string {
	partitionVars := ctx.Config().productVariables.PartitionVarsForSoongMigrationOnlyDoNotUse
	productPackages := partitionVars.ProductPackages
	ctx.VisitAllModules(func(m Module) {
		if !installInSystem(m) {
			return
		}

	})

}

func installInSystem(m Module) bool {
	return !m.InstallInData() && !m.InstallInDebugRamdisk() && !m.InstallInOdm() &&
		!m.InstallInProduct() && !m.InstallInRamdisk() && !m.InstallInRecovery() && !m.InstallInRoot() &&
		!m.InstallInSanitizerDir() && !m.InstallInTestcases() && !m.InstallInVendor() &&
		!m.InstallInVendorRamdisk()
}
