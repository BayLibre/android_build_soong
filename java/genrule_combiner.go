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
	"fmt"
	"io"
	"reflect"

	"android/soong/android"
	"android/soong/dexpreopt"

	"github.com/google/blueprint/depset"
	"github.com/google/blueprint/proptools"
)

type GenruleCombiner struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties GenruleCombinerProperties

	headerJars                    android.Paths
	implementationJars            android.Paths
	implementationAndResourceJars android.Paths
	resourceJars                  android.Paths
	aconfigProtoFiles             android.Paths

	srcJarArgs []string
	srcJarDeps android.Paths

	combinedHeaderJar         android.Path
	combinedImplementationJar android.Path
}

type GenruleCombinerProperties struct {
	// List of modules whose implementation (and resources) jars will be visible to modules
	// that depend on this module.
	Libs proptools.Configurable[[]string] `android:"arch_variant"`

	// List of modules whose header jars will be visible to modules that depend on this module.
	Headers proptools.Configurable[[]string] `android:"arch_variant"`
}

// java_genrule_combiner provides the implementation from its sources, with the header jars from
// `headers`.
//
// TODO: Add example.
//
// It is rarely necessary.
func GenruleCombinerFactory() android.Module {
	module := &GenruleCombiner{}

	module.AddProperties(&module.properties)
	InitJavaModule(module, android.HostAndDeviceSupported)
	return module
}

var genruleCombinerHeaderDepTag = dependencyTag{name: "genrule_combiner_header"}
var genruleCombinerLibDepTag = dependencyTag{name: "genrule_combiner_lib"}

func (d *GenruleCombiner) DepsMutator(ctx android.BottomUpMutatorContext) {
	ctx.AddFarVariationDependencies(ctx.Config().AndroidCommonTarget.Variations(),
		genruleCombinerLibDepTag, d.properties.Libs.GetOrDefault(ctx, nil)...)
	ctx.AddFarVariationDependencies(ctx.Config().AndroidCommonTarget.Variations(),
		genruleCombinerHeaderDepTag, d.properties.Headers.GetOrDefault(ctx, nil)...)
}

func (d *GenruleCombiner) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if len(d.properties.Libs.GetOrDefault(ctx, nil)) < 1 {
		ctx.PropertyErrorf("libs", "at least one dependency is required")
	}

	if len(d.properties.Headers.GetOrDefault(ctx, nil)) < 1 {
		ctx.PropertyErrorf("headers", "at least one dependency is required")
	}

	var transitiveHeaderJars []depset.DepSet[android.Path]
	var transitiveImplementationJars []depset.DepSet[android.Path]
	var transitiveResourceJars []depset.DepSet[android.Path]

	// Collect the headers first, so that aconfig flag values for the libraries will override
	// values from the headers (if they are different).
	ctx.VisitDirectDepsWithTag(genruleCombinerHeaderDepTag, func(m android.Module) {
		if dep, ok := android.OtherModuleProvider(ctx, m, JavaInfoProvider); ok {
			d.headerJars = append(d.headerJars, dep.HeaderJars...)

			d.srcJarArgs = append(d.srcJarArgs, dep.SrcJarArgs...)
			d.srcJarDeps = append(d.srcJarDeps, dep.SrcJarDeps...)
			d.aconfigProtoFiles = append(d.aconfigProtoFiles, dep.AconfigIntermediateCacheOutputPaths...)

			transitiveHeaderJars = append(transitiveHeaderJars, dep.TransitiveStaticLibsHeaderJars)
		} else {
			ctx.PropertyErrorf("headers", "module %q cannot be used as a dependency", ctx.OtherModuleName(m))
		}
	})
	ctx.VisitDirectDepsWithTag(genruleCombinerLibDepTag, func(m android.Module) {
		if dep, ok := android.OtherModuleProvider(ctx, m, JavaInfoProvider); ok {
			d.implementationJars = append(d.implementationJars, dep.ImplementationJars...)
			d.implementationAndResourceJars = append(d.implementationAndResourceJars, dep.ImplementationAndResourcesJars...)
			d.resourceJars = append(d.resourceJars, dep.ResourceJars...)

			transitiveImplementationJars = append(transitiveImplementationJars, dep.TransitiveStaticLibsImplementationJars)
			transitiveResourceJars = append(transitiveResourceJars, dep.TransitiveStaticLibsResourceJars)
			d.aconfigProtoFiles = append(d.aconfigProtoFiles, dep.AconfigIntermediateCacheOutputPaths...)
		} else if reflect.TypeOf(m).String() == "*genrule.Module" {
			// The only supported non-java module types are java_genrule and genrule.
			if dep, ok := android.OtherModuleProvider(ctx, m, android.OutputFilesProvider); ok {
				d.implementationJars = append(d.implementationJars, dep.DefaultOutputFiles...)
				d.implementationAndResourceJars = append(d.implementationAndResourceJars, dep.DefaultOutputFiles...)
			} else {
				ctx.PropertyErrorf("libs", "module %q cannot be used as a dependency (no output)", ctx.OtherModuleName(m))
			}
		} else {
			ctx.PropertyErrorf("libs", "module %q cannot be used as a dependency", ctx.OtherModuleName(m))
		}
	})

	jarName := ctx.ModuleName() + ".jar"

	if len(d.implementationAndResourceJars) > 1 {
		outputFile := android.PathForModuleOut(ctx, "combined", jarName)
		TransformJarsToJar(ctx, outputFile, "combine", d.implementationAndResourceJars,
			android.OptionalPath{}, false, nil, nil)
		d.combinedImplementationJar = outputFile
	} else if len(d.implementationAndResourceJars) == 1 {
		d.combinedImplementationJar = d.implementationAndResourceJars[0]
	}

	if len(d.headerJars) > 1 {
		outputFile := android.PathForModuleOut(ctx, "turbine-combined", jarName)
		TransformJarsToJar(ctx, outputFile, "turbine combine", d.headerJars,
			android.OptionalPath{}, false, nil, []string{"META-INF/TRANSITIVE"})
		d.combinedHeaderJar = outputFile
	} else if len(d.headerJars) == 1 {
		d.combinedHeaderJar = d.headerJars[0]
	}

	javaInfo := &JavaInfo{
		HeaderJars:                             d.headerJars,
		LocalHeaderJars:                        d.headerJars,
		TransitiveStaticLibsHeaderJars:         depset.New(depset.PREORDER, nil, transitiveHeaderJars),
		TransitiveStaticLibsImplementationJars: depset.New(depset.PREORDER, nil, transitiveImplementationJars),
		TransitiveStaticLibsResourceJars:       depset.New(depset.PREORDER, nil, transitiveResourceJars),
		ImplementationAndResourcesJars:         d.implementationAndResourceJars,
		ImplementationJars:                     d.implementationJars,
		ResourceJars:                           d.resourceJars,
		SrcJarArgs:                             d.srcJarArgs,
		SrcJarDeps:                             d.srcJarDeps,
		StubsLinkType:                          Implementation,
		AconfigIntermediateCacheOutputPaths:    d.aconfigProtoFiles,
	}
	setExtraJavaInfo(ctx, d, javaInfo)
	android.SetProvider(ctx, JavaInfoProvider, javaInfo)

}

func (d *GenruleCombiner) HeaderJars() android.Paths {
	return d.headerJars
}

func (d *GenruleCombiner) ImplementationAndResourcesJars() android.Paths {
	return d.implementationAndResourceJars
}

func (d *GenruleCombiner) DexJarBuildPath(ctx android.ModuleErrorfContext) android.Path {
	return nil
}

func (d *GenruleCombiner) DexJarInstallPath() android.Path {
	return nil
}

func (d *GenruleCombiner) AidlIncludeDirs() android.Paths {
	return nil
}

func (d *GenruleCombiner) ClassLoaderContexts() dexpreopt.ClassLoaderContextMap {
	return nil
}

func (d *GenruleCombiner) JacocoReportClassesFile() android.Path {
	return nil
}

func (d *GenruleCombiner) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "JAVA_LIBRARIES",
		OutputFile: android.OptionalPathForPath(d.combinedImplementationJar),
		// Make does not support Windows Java modules
		Disabled: d.Os() == android.Windows,
		Include:  "$(BUILD_SYSTEM)/soong_java_prebuilt.mk",
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				fmt.Fprintln(w, "LOCAL_UNINSTALLABLE_MODULE := true")
				fmt.Fprintln(w, "LOCAL_SOONG_HEADER_JAR :=", d.combinedHeaderJar.String())
				fmt.Fprintln(w, "LOCAL_SOONG_CLASSES_JAR :=", d.combinedImplementationJar.String())
			},
		},
	}
}

// implement the following interface for IDE completion.
var _ android.IDEInfo = (*GenruleCombiner)(nil)

func (d *GenruleCombiner) IDEInfo(ctx android.BaseModuleContext, ideInfo *android.IdeInfo) {
	ideInfo.Deps = append(ideInfo.Deps, d.properties.Libs.GetOrDefault(ctx, nil)...)
	ideInfo.Libs = append(ideInfo.Libs, d.properties.Libs.GetOrDefault(ctx, nil)...)
	ideInfo.Deps = append(ideInfo.Deps, d.properties.Headers.GetOrDefault(ctx, nil)...)
	ideInfo.Libs = append(ideInfo.Libs, d.properties.Headers.GetOrDefault(ctx, nil)...)
}
