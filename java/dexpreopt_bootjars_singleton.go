// Copyright 2019 Google Inc. All rights reserved.
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
	"sort"
	"strings"

	"android/soong/android"
)

func init() {
	android.RegisterSingletonType("dex_bootjars_singleton", DexpreoptBootJarsSingleton)
}

func DexpreoptBootJarsSingleton() android.Singleton {
	return &DexpreoptBootJarsS{}
}

// dexpreoptBoot singleton rules
func (d *DexpreoptBootJarsS) GenerateBuildActions(ctx android.SingletonContext) {
	if skipDexpreoptBootJars(ctx) {
		return
	}

	d.dexpreoptConfigForMake = android.PathForOutput(ctx, ctx.Config().DeviceName(), "dexpreopt.config")
	writeGlobalConfigForMake(ctx, d.dexpreoptConfigForMake)

	// find the unique DexpreoptBootJars module
	var dexpreopt *DexpreoptBootJars
	ctx.VisitAllModules(func(module android.Module) {
		if j, ok := module.(*DexpreoptBootJars); ok {
			if dexpreopt != nil {
				ctx.Errorf("multiple DexpreoptBootJars modules found: %s, %s",
					dexpreopt.Name(), j.Name())
			}
			dexpreopt = j
		}
	})
	if dexpreopt == nil {
		ctx.Errorf("no DexpreoptBootJars module found")
	}

	// use the necessary parts of the module in the singleton
	d.defaultBootImage = dexpreopt.defaultBootImage
	d.otherImages = dexpreopt.otherImages
}

type DexpreoptBootJarsS struct {
	dexpreoptConfigForMake android.WritablePath
	defaultBootImage       *bootImage
	otherImages            []*bootImage
}

// Export paths for default boot image to Make
func (d *DexpreoptBootJarsS) MakeVars(ctx android.MakeVarsContext) {
	if d.dexpreoptConfigForMake != nil {
		ctx.Strict("DEX_PREOPT_CONFIG_FOR_MAKE", d.dexpreoptConfigForMake.String())
	}

	image := d.defaultBootImage
	if image != nil {
		ctx.Strict("DEXPREOPT_IMAGE_PROFILE_BUILT_INSTALLED", image.profileInstalls.String())
		ctx.Strict("DEXPREOPT_BOOTCLASSPATH_DEX_FILES", strings.Join(image.dexPaths.Strings(), " "))
		ctx.Strict("DEXPREOPT_BOOTCLASSPATH_DEX_LOCATIONS", strings.Join(image.dexLocations, " "))
		ctx.Strict("DEXPREOPT_IMAGE_ZIP_"+image.name, image.zip.String())

		var imageNames []string
		for _, current := range append(d.otherImages, image) {
			imageNames = append(imageNames, current.name)
			var arches []android.ArchType
			for arch, _ := range current.images {
				arches = append(arches, arch)
			}

			sort.Slice(arches, func(i, j int) bool { return arches[i].String() < arches[j].String() })

			for _, arch := range arches {
				ctx.Strict("DEXPREOPT_IMAGE_VDEX_BUILT_INSTALLED_"+current.name+"_"+arch.String(), current.vdexInstalls[arch].String())
				ctx.Strict("DEXPREOPT_IMAGE_"+current.name+"_"+arch.String(), current.images[arch].String())
				ctx.Strict("DEXPREOPT_IMAGE_DEPS_"+current.name+"_"+arch.String(), strings.Join(current.imagesDeps[arch].Strings(), " "))
				ctx.Strict("DEXPREOPT_IMAGE_BUILT_INSTALLED_"+current.name+"_"+arch.String(), current.installs[arch].String())
				ctx.Strict("DEXPREOPT_IMAGE_UNSTRIPPED_BUILT_INSTALLED_"+current.name+"_"+arch.String(), current.unstrippedInstalls[arch].String())
				if current.zip != nil {
				}
			}
		}
		ctx.Strict("DEXPREOPT_IMAGE_NAMES", strings.Join(imageNames, " "))
	}
}

func writeGlobalConfigForMake(ctx android.SingletonContext, path android.WritablePath) {
	data := dexpreoptGlobalConfigRaw(ctx).data

	ctx.Build(pctx, android.BuildParams{
		Rule:   android.WriteFile,
		Output: path,
		Args: map[string]string{
			"content": string(data),
		},
	})
}
