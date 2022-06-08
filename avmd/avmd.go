// Copyright (C) 2022 The Android Open Source Project
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

package avmd

import (
	"fmt"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/apex"
	"android/soong/filesystem"
	"android/soong/java"
)

func init() {
	android.RegisterModuleType("avmd", avmdFactory)
}

type avmd struct {
	android.ModuleBase

	properties avmdProperties

	output     android.OutputPath
	installDir android.InstallPath
}

type avmdProperties struct {
	// Set the name of the output. Defaults to <module_name>.avmd.
	Stem *string

	// List of vbmeta, android_app or apex modules that this AVMD has descriptors for.
	Resources []struct {
		// The namespace the resource is part of.
		Namespace *string

		// The name of the resource withing the namespace.
		Name *string

		// The module that provides the resource.
		Module *string
	}
}

// AVMD is a description of the resources that compose a virtual machine.
func avmdFactory() android.Module {
	module := &avmd{}
	module.AddProperties(&module.properties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibFirst)
	return module
}

type avmdDep struct {
	blueprint.BaseDependencyTag
	kind string
}

var avmdResourceDep = avmdDep{kind: "resource"}

func (a *avmd) DepsMutator(ctx android.BottomUpMutatorContext) {
	modules := make([]string, len(a.properties.Resources))

	for i, r := range a.properties.Resources {
		modules[i] = *r.Module
	}

	ctx.AddDependency(ctx.Module(), avmdResourceDep, modules...)
}

func (a *avmd) installFileName() string {
	return proptools.StringDefault(a.properties.Stem, a.BaseModuleName()+".avmd")
}

var pctx = android.NewPackageContext("android/soong/avmd")

func (a *avmd) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	a.output = android.PathForModuleOut(ctx, a.installFileName()).OutputPath

	builder := android.NewRuleBuilder(pctx, ctx)
	cmd := builder.Command().BuiltTool("avmdtool").Text("create").Output(a.output)

	for i, d := range ctx.GetDirectDepsWithTag(avmdResourceDep) {
		switch r := d.(type) {
		case filesystem.Filesystem:
			signedImage := r.SignedOutputPath()
			if signedImage == nil {
				ctx.PropertyErrorf("resources", "%q(type: %s) is not signed. Use `use_avb: true`",
					r.Name(), ctx.OtherModuleType(r))
				continue
			}
			cmd.Flag("--vbmeta")
			cmd.Text(*a.properties.Resources[i].Namespace)
			cmd.Text(*a.properties.Resources[i].Name)
			cmd.Input(r.SignedOutputPath())
		case *java.AndroidApp:
			cmd.Flag("--apk")
			cmd.Text(*a.properties.Resources[i].Namespace)
			cmd.Text(*a.properties.Resources[i].Name)
			cmd.Input(r.OutputFile())
		case apex.ApexBundle:
			cmd.Flag("--apex-payload")
			cmd.Text(*a.properties.Resources[i].Namespace)
			cmd.Text(*a.properties.Resources[i].Name)
			cmd.Input(r.OutputApexPath())
		default:
			ctx.PropertyErrorf("resources", "%q(type: %s) is not supported",
				r.Name(), ctx.OtherModuleType(r))
		}
	}

	builder.Build("avmd", fmt.Sprintf("avmd %s", ctx.ModuleName()))

	a.installDir = android.PathForModuleInstall(ctx, "etc")
	ctx.InstallFile(a.installDir, a.installFileName(), a.output)
}

var _ android.AndroidMkEntriesProvider = (*avmd)(nil)

// Implements android.AndroidMkEntriesProvider
func (a *avmd) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(a.output),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", a.installDir.String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", a.installFileName())
			},
		},
	}}
}
