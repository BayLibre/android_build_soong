// Copyright 2025 Google Inc. All rights reserved.
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
	"github.com/google/blueprint"
)

func init() {
	InitRegistrationContext.RegisterSingletonType("otatools_package_cert_zip_singleton", otatoolsPackageCertZipSingletonFactory)
	RegisterOtatoolsPackageBuildComponents(InitRegistrationContext)
	pctx.HostBinToolVariable("SoongZipCmd", "soong_zip")
}

func RegisterOtatoolsPackageBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("otatools_package", OtatoolsPackageFactory)
}

type OtatoolsPackage struct {
	ModuleBase
}

func OtatoolsPackageFactory() Module {
	module := &OtatoolsPackage{}
	InitAndroidModule(module)
	return module
}

var (
	otatoolsPackageCertRule = pctx.AndroidStaticRule("otatools_package_cert", blueprint.RuleParams{
		Command:     "${SoongZipCmd} -o $out -l $in",
		Description: "Zip otatools-package cert files",
		CommandDeps: []string{"${SoongZipCmd}"},
	})
)

func (fg *OtatoolsPackage) GenerateAndroidBuildActions(ctx ModuleContext) {
	fileListFile := PathForArbitraryOutput(ctx, ".module_paths", "OtaToolsCertFiles.list")
	otatoolsPackageCertZip := PathForModuleOut(ctx, "otatools_package_cert.zip")
	ctx.Build(pctx, BuildParams{
		Rule:   otatoolsPackageCertRule,
		Input:  fileListFile,
		Output: otatoolsPackageCertZip,
	})
	ctx.SetOutputFiles([]Path{otatoolsPackageCertZip}, "")
}

func otatoolsPackageCertZipSingletonFactory() Singleton {
	return &otatoolsPackageCertZipSingleton{}
}

type otatoolsPackageCertZipSingleton struct{}

func (s *otatoolsPackageCertZipSingleton) GenerateBuildActions(ctx SingletonContext) {
	fileListFile := PathForArbitraryOutput(ctx, ".module_paths", "OtaToolsCertFiles.list")
	out := PathForOutput(ctx, "otatools_package_cert.zip")
	dep := PathForOutput(ctx, "otatools_package_cert.zip.d")

	builder := NewRuleBuilder(pctx, ctx)
	builder.Command().BuiltTool("soong_zip").
		FlagWithOutput("-o ", out).
		FlagWithInput("-l ", fileListFile)
	builder.Command().Textf("echo '%s : ' $(cat %s) > ", out, fileListFile).DepFile(dep)
	builder.Build("otatools_package_cert_zip", "build otatools_package_cert zip")

	ctx.Phony("otatools_package_cert", out)
}
