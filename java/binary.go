// Copyright 2021 Google Inc. All rights reserved.
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
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

// This file contains the module implementations for java_binary and java_binary_host.

func init() {
	RegisterBinaryBuildComponents(android.InitRegistrationContext)
}

func RegisterBinaryBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("java_binary", BinaryFactory)
	ctx.RegisterModuleType("java_binary_host", BinaryHostFactory)
}

//
// Java Binaries (.jar file plus wrapper script)
//

type binaryProperties struct {
	// installable script to execute the resulting jar
	Wrapper *string `android:"path"`

	// Name of the class containing main to be inserted into the manifest as Main-Class.
	Main_class *string

	// Names of modules containing JNI libraries that should be installed alongside the host
	// variant of the binary.
	Jni_libs []string
}

type Binary struct {
	Library

	binaryProperties binaryProperties

	isWrapperVariant bool

	wrapperFile android.Path
	binaryFile  android.InstallPath
}

func (j *Binary) HostToolPath() android.OptionalPath {
	return android.OptionalPathForPath(j.binaryFile)
}

func (j *Binary) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if ctx.Arch().ArchType == android.Common {
		// Compile the jar
		if j.binaryProperties.Main_class != nil {
			if j.properties.Manifest != nil {
				ctx.PropertyErrorf("main_class", "main_class cannot be used when manifest is set")
			}
			manifestFile := android.PathForModuleOut(ctx, "manifest.txt")
			GenerateMainClassManifest(ctx, manifestFile, String(j.binaryProperties.Main_class))
			j.overrideManifest = android.OptionalPathForPath(manifestFile)
		}

		j.Library.GenerateAndroidBuildActions(ctx)
	} else {
		// Handle the binary wrapper
		j.isWrapperVariant = true

		if j.binaryProperties.Wrapper != nil {
			j.wrapperFile = android.PathForModuleSrc(ctx, *j.binaryProperties.Wrapper)
		} else {
			j.wrapperFile = android.PathForSource(ctx, "build/soong/scripts/jar-wrapper.sh")
		}

		// The host installation rules make the installed wrapper depend on all the dependencies
		// of the wrapper variant, which will include the common variant's jar file and any JNI
		// libraries.  This is verified by TestBinary.
		j.binaryFile = ctx.InstallExecutable(android.PathForModuleInstall(ctx, "bin"),
			ctx.ModuleName(), j.wrapperFile)
	}
}

func (j *Binary) DepsMutator(ctx android.BottomUpMutatorContext) {
	if ctx.Arch().ArchType == android.Common || ctx.BazelConversionMode() {
		j.deps(ctx)
	}
	if ctx.Arch().ArchType != android.Common || ctx.BazelConversionMode() {
		// These dependencies ensure the host installation rules will install the jar file and
		// the jni libraries when the wrapper is installed.
		ctx.AddVariationDependencies(nil, jniInstallTag, j.binaryProperties.Jni_libs...)
		ctx.AddVariationDependencies(
			[]blueprint.Variation{{Mutator: "arch", Variation: android.CommonArch.String()}},
			binaryInstallTag, ctx.ModuleName())
	}
}

// java_binary builds a `.jar` file and a shell script that executes it for the device, and possibly for the host
// as well.
//
// By default, a java_binary has a single variant that produces a `.jar` file containing `classes.dex` files that were
// compiled against the device bootclasspath.
//
// Specifying `host_supported: true` will produce two variants, one compiled against the device bootclasspath and one
// compiled against the host bootclasspath.
func BinaryFactory() android.Module {
	module := &Binary{}

	module.addHostAndDeviceProperties()
	module.AddProperties(&module.binaryProperties)

	module.Module.properties.Installable = proptools.BoolPtr(true)

	android.InitAndroidArchModule(module, android.HostAndDeviceSupported, android.MultilibCommonFirst)
	android.InitDefaultableModule(module)
	return module
}

// java_binary_host builds a `.jar` file and a shell script that executes it for the host.
//
// A java_binary_host has a single variant that produces a `.jar` file containing `.class` files that were
// compiled against the host bootclasspath.
func BinaryHostFactory() android.Module {
	module := &Binary{}

	module.addHostProperties()
	module.AddProperties(&module.binaryProperties)

	module.Module.properties.Installable = proptools.BoolPtr(true)

	android.InitAndroidArchModule(module, android.HostSupported, android.MultilibCommonFirst)
	android.InitDefaultableModule(module)
	return module
}
