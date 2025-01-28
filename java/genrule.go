// Copyright 2017 Google Inc. All rights reserved.
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
	"reflect"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/genrule"
)

func init() {
	RegisterGenRuleBuildComponents(android.InitRegistrationContext)
}

func RegisterGenRuleBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("java_genrule", GenRuleFactory)
	ctx.RegisterModuleType("java_genrule_host", GenRuleFactoryHost)
	ctx.FinalDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.BottomUp("java_genrule_tool_deps", toolDepsMutator)
	})
}

func toolDepsMutator(ctx android.BottomUpMutatorContext) {
	if g, ok := ctx.Module().(*JavaGenRuleModule); ok {
		g.Module.AddToolDeps(ctx)
	}
}

type javaGenRuleProperties struct {
	// When true, the header jar(s) for the Srcs file is provided by this module.
	Provide_java_info proptools.Configurable[bool]
}

type JavaGenRuleModule struct {
	*genrule.Module

	javaGenRuleProperties javaGenRuleProperties
}

// java_genrule is a genrule that can depend on other java_* objects.
//
// By default a java_genrule has a single variant that will run against the device variant of its dependencies and
// produce an output that can be used as an input to a device java rule.
//
// Specifying `host_supported: true` will produce two variants, one that uses device dependencies and one that uses
// host dependencies.  Each variant will run the command.
//
// Use a java_genrule instead of a genrule when it needs to depend on or be depended on by other java modules, unless
// the dependency is for a generated source file.
//
// Examples:
//
// Use a java_genrule to package generated java resources:
//
//	java_genrule {
//	    name: "generated_resources",
//	    tools: [
//	        "generator",
//	        "soong_zip",
//	    ],
//	    srcs: ["generator_inputs/**/*"],
//	    out: ["generated_android_icu4j_resources.jar"],
//	    cmd: "$(location generator) $(in) -o $(genDir) " +
//	        "&& $(location soong_zip) -o $(out) -C $(genDir)/res -D $(genDir)/res",
//	}
//
//	java_library {
//	    name: "lib_with_generated_resources",
//	    srcs: ["src/**/*.java"],
//	    static_libs: ["generated_resources"],
//	}
func GenRuleFactory() android.Module {
	module := NewJavaGenRule()

	android.InitAndroidArchModule(module, android.HostAndDeviceSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)

	return module
}

// java_genrule_host is a genrule that can depend on other java_* objects.
//
// A java_genrule_host has a single variant that will run against the host variant of its dependencies and
// produce an output that can be used as an input to a host java rule.
func GenRuleFactoryHost() android.Module {
	module := NewJavaGenRule()

	android.InitAndroidArchModule(module, android.HostSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)

	return module
}

func NewJavaGenRule() *JavaGenRuleModule {
	module := &JavaGenRuleModule{
		Module: genrule.NewGenRule(),
	}
	module.AddProperties(&module.javaGenRuleProperties)
	return module
}

func (g *JavaGenRuleModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	g.Module.GenerateAndroidBuildActions(ctx)

	if g.javaGenRuleProperties.Provide_java_info.GetOrDefault(ctx, false) {
		g.generateJavaInfoProvider(ctx)
	}
}

// Generate our JavaInfoProvider by copying and updating the source module's provider data.
func (g *JavaGenRuleModule) generateJavaInfoProvider(ctx android.ModuleContext) {
	if len(g.SrcFiles) != 1 {
		// We do not merge the providers, so we can only accept one source.
		ctx.ModuleErrorf("Must have exactly 1 source when using Provide_java_info")
	}

	var javaInfo *JavaInfo
	var foundName string
	outputFiles := g.Module.OutputFiles()
	ctx.VisitDirectDepsProxy(func(dep android.ModuleProxy) {
		if provider, ok := android.OtherModuleProvider(ctx, dep, JavaInfoProvider); ok {
			// Since the input java package is specified with `srcs` (making the dependency
			// tag `android.sourceOrOutputDependencyTag`, just look for a module whose
			// JavaInfoProvider.OutputFile matches our (single) source file.
			if g.SrcFiles[0] != provider.OutputFile {
				return
			}
			if !reflect.DeepEqual(provider.ImplementationJars, provider.ImplementationAndResourcesJars) {
				// This is not a supported case, since the rule has no way to tell us what to use for
				// ImplementationAndResourcesJars.
				ctx.ModuleErrorf("%s has different implementation and resources", dep.Name())
			}
			if javaInfo != nil {
				// No merge operation at this time.
				ctx.ModuleErrorf("Found multiple JavaInfoProviders in deps: %s, %s", foundName, dep.Name())
			}
			foundName = dep.Name()
			javaInfo = &JavaInfo{
				HeaderJars:                             provider.HeaderJars,
				RepackagedHeaderJars:                   provider.RepackagedHeaderJars,
				TransitiveLibsHeaderJarsForR8:          provider.TransitiveLibsHeaderJarsForR8,
				TransitiveStaticLibsHeaderJarsForR8:    provider.TransitiveStaticLibsHeaderJarsForR8,
				TransitiveStaticLibsHeaderJars:         provider.TransitiveStaticLibsHeaderJars,
				TransitiveStaticLibsImplementationJars: provider.TransitiveStaticLibsImplementationJars,
				TransitiveStaticLibsResourceJars:       provider.TransitiveStaticLibsResourceJars,
				ImplementationAndResourcesJars:         outputFiles,
				ImplementationJars:                     outputFiles,
				ResourceJars:                           provider.ResourceJars,
				LocalHeaderJars:                        provider.LocalHeaderJars,
				AidlIncludeDirs:                        provider.AidlIncludeDirs,
				SrcJarArgs:                             provider.SrcJarArgs,
				SrcJarDeps:                             provider.SrcJarDeps,
				TransitiveSrcFiles:                     provider.TransitiveSrcFiles,
				ExportedPlugins:                        provider.ExportedPlugins,
				ExportedPluginClasses:                  provider.ExportedPluginClasses,
				ExportedPluginDisableTurbine:           provider.ExportedPluginDisableTurbine,
				JacocoReportClassesFile:                provider.JacocoReportClassesFile,
				StubsLinkType:                          provider.StubsLinkType,
				AconfigIntermediateCacheOutputPaths:    provider.AconfigIntermediateCacheOutputPaths,
				SdkVersion:                             provider.SdkVersion,
				OutputFile:                             outputFiles[0],
				AndroidLibraryDependencyInfo:           provider.AndroidLibraryDependencyInfo,
				UsesLibraryDependencyInfo:              provider.UsesLibraryDependencyInfo,
				SdkLibraryComponentDependencyInfo:      provider.SdkLibraryComponentDependencyInfo,
				ProvidesUsesLibInfo:                    provider.ProvidesUsesLibInfo,
				ModuleWithUsesLibraryInfo:              provider.ModuleWithUsesLibraryInfo,
				ModuleWithSdkDepInfo:                   provider.ModuleWithSdkDepInfo,
			}
		}
	})
	if javaInfo != nil {
		// Our input provided JavaInfo, so we do as well.
		android.SetProvider(ctx, JavaInfoProvider, javaInfo)
		ctx.SetOutputFiles(outputFiles, ".jar")
		ctx.SetOutputFiles(javaInfo.HeaderJars, ".hjar")
	} else {
		ctx.ModuleErrorf("No JavaInfoProvider found")
	}
}

func (g *JavaGenRuleModule) AndroidMk() android.AndroidMkData {
	return g.Module.AndroidMk()
}

func (g *JavaGenRuleModule) IDEInfo(ctx android.BaseModuleContext, dpInfo *android.IdeInfo) {
	g.Module.IDEInfo(ctx, dpInfo)
}

func (g *JavaGenRuleModule) ShouldSupportSdkVersion(ctx android.BaseModuleContext,
	sdkVersion android.ApiLevel) error {
	return g.Module.ShouldSupportSdkVersion(ctx, sdkVersion)
}
