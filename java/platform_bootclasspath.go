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
	"fmt"
	"sort"
	"strings"

	"android/soong/android"
	"android/soong/dexpreopt"
)

func init() {
	registerPlatformBootclasspathBuildComponents(android.InitRegistrationContext)
}

func registerPlatformBootclasspathBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("platform_bootclasspath", platformBootclasspathFactory)
}

// The tags used for the dependencies between the platform bootclasspath and any configured boot
// jars.
var (
	platformBootclasspathArtBootJarDepTag          = bootclasspathDependencyTag{name: "art-boot-jar"}
	platformBootclasspathNonUpdatableBootJarDepTag = bootclasspathDependencyTag{name: "non-updatable-boot-jar"}
	platformBootclasspathUpdatableBootJarDepTag    = bootclasspathDependencyTag{name: "updatable-boot-jar"}
)

type platformBootclasspathModule struct {
	android.ModuleBase
	ClasspathFragmentBase

	properties platformBootclasspathProperties

	// The apex:module pairs obtained from the configured modules.
	//
	// Currently only for testing.
	configuredModules []android.Module

	// The apex:module pairs obtained from the fragments.
	//
	// Currently only for testing.
	fragments []android.Module

	// Path to the monolithic hiddenapi-flags.csv file.
	hiddenAPIFlagsCSV android.OutputPath

	// Path to the monolithic hiddenapi-index.csv file.
	hiddenAPIIndexCSV android.OutputPath

	// Path to the monolithic hiddenapi-unsupported.csv file.
	hiddenAPIMetadataCSV android.OutputPath
}

type platformBootclasspathProperties struct {
	BootclasspathFragmentsDepsProperties

	Hidden_api HiddenAPIFlagFileProperties
}

func platformBootclasspathFactory() android.Module {
	m := &platformBootclasspathModule{}
	m.AddProperties(&m.properties)
	// TODO(satayev): split systemserver and apex jars into separate configs.
	initClasspathFragment(m)
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibCommon)
	return m
}

var _ android.OutputFileProducer = (*platformBootclasspathModule)(nil)

func (b *platformBootclasspathModule) AndroidMkEntries() (entries []android.AndroidMkEntries) {
	entries = append(entries, android.AndroidMkEntries{
		Class: "FAKE",
		// Need at least one output file in order for this to take effect.
		OutputFile: android.OptionalPathForPath(b.hiddenAPIFlagsCSV),
		Include:    "$(BUILD_PHONY_PACKAGE)",
	})
	entries = append(entries, b.classpathFragmentBase().getAndroidMkEntries()...)
	return
}

// Make the hidden API files available from the platform-bootclasspath module.
func (b *platformBootclasspathModule) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case "hiddenapi-flags.csv":
		return android.Paths{b.hiddenAPIFlagsCSV}, nil
	case "hiddenapi-index.csv":
		return android.Paths{b.hiddenAPIIndexCSV}, nil
	case "hiddenapi-metadata.csv":
		return android.Paths{b.hiddenAPIMetadataCSV}, nil
	}

	return nil, fmt.Errorf("unknown tag %s", tag)
}

func (b *platformBootclasspathModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	b.hiddenAPIDepsMutator(ctx)

	if SkipDexpreoptBootJars(ctx) {
		return
	}

	// Add a dependency onto the dex2oat tool which is needed for creating the boot image. The
	// path is retrieved from the dependency by GetGlobalSoongConfig(ctx).
	dexpreopt.RegisterToolDeps(ctx)
}

func (b *platformBootclasspathModule) hiddenAPIDepsMutator(ctx android.BottomUpMutatorContext) {
	if ctx.Config().IsEnvTrue("UNSAFE_DISABLE_HIDDENAPI_FLAGS") {
		return
	}

	// Add dependencies onto the stub lib modules.
	sdkKindToStubLibModules := hiddenAPIComputeMonolithicStubLibModules(ctx.Config())
	hiddenAPIAddStubLibDependencies(ctx, sdkKindToStubLibModules)
}

func (b *platformBootclasspathModule) BootclasspathDepsMutator(ctx android.BottomUpMutatorContext) {
	// Add dependencies on all the modules configured in the "art" boot image.
	artImageConfig := genBootImageConfigs(ctx)[artBootImageName]
	addDependenciesOntoBootImageModules(ctx, artImageConfig.modules, platformBootclasspathArtBootJarDepTag)

	// Add dependencies on all the non-updatable module configured in the "boot" boot image. That does
	// not include modules configured in the "art" boot image.
	bootImageConfig := b.getImageConfig(ctx)
	addDependenciesOntoBootImageModules(ctx, bootImageConfig.modules, platformBootclasspathNonUpdatableBootJarDepTag)

	// Add dependencies on all the updatable modules.
	updatableModules := dexpreopt.GetGlobalConfig(ctx).UpdatableBootJars
	addDependenciesOntoBootImageModules(ctx, updatableModules, platformBootclasspathUpdatableBootJarDepTag)

	// Add dependencies on all the fragments.
	b.properties.BootclasspathFragmentsDepsProperties.addDependenciesOntoFragments(ctx)
}

func addDependenciesOntoBootImageModules(ctx android.BottomUpMutatorContext, modules android.ConfiguredJarList, tag bootclasspathDependencyTag) {
	for i := 0; i < modules.Len(); i++ {
		apex := modules.Apex(i)
		name := modules.Jar(i)

		addDependencyOntoApexModulePair(ctx, apex, name, tag)
	}
}

func (b *platformBootclasspathModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	b.classpathFragmentBase().generateAndroidBuildActions(ctx)

	var artModules []android.Module
	var nonUpdatableModules []android.Module
	var updatableModules []android.Module
	ctx.VisitDirectDepsIf(isActiveModule, func(module android.Module) {
		tag := ctx.OtherModuleDependencyTag(module)
		switch tag {
		case platformBootclasspathArtBootJarDepTag:
			artModules = append(artModules, module)
		case platformBootclasspathNonUpdatableBootJarDepTag:
			nonUpdatableModules = append(nonUpdatableModules, module)
		case platformBootclasspathUpdatableBootJarDepTag:
			updatableModules = append(updatableModules, module)
		case bootclasspathFragmentDepTag:
			b.fragments = append(b.fragments, module)
		}
	})

	var allModules []android.Module
	allModules = append(allModules, artModules...)
	allModules = append(allModules, nonUpdatableModules...)
	allModules = append(allModules, updatableModules...)
	b.configuredModules = allModules

	// Check the configuration of the boot modules.
	// ART modules are checked by the art-bootclasspath-fragment.
	b.checkNonUpdatableModules(ctx, nonUpdatableModules)
	b.checkUpdatableModules(ctx, updatableModules)

	b.generateHiddenAPIBuildActions(ctx, b.configuredModules, b.fragments)

	// Nothing to do if skipping the dexpreopt of boot image jars.
	if SkipDexpreoptBootJars(ctx) {
		return
	}

	b.generateBootImageBuildActions(ctx, updatableModules)
}

// checkNonUpdatableModules ensures that the non-updatable modules that this depends on satisfy the
// constraints that apply to them.
func (b *platformBootclasspathModule) checkNonUpdatableModules(ctx android.ModuleContext, modules []android.Module) {
	for _, m := range modules {
		apexInfo := ctx.OtherModuleProvider(m, android.ApexInfoProvider).(android.ApexInfo)
		fromUpdatableApex := apexInfo.Updatable
		if fromUpdatableApex {
			// error: this jar is part of an updatable apex
			ctx.ModuleErrorf("module %q from updatable apexes %q is not allowed in the framework boot image", ctx.OtherModuleName(m), apexInfo.InApexes)
		} else {
			// ok: this jar is part of the platform or a non-updatable apex
		}
	}
}

// checkUpdatableModules ensures that the updatable modules that this depends on satisfy the
// constraints that apply to them.
func (b *platformBootclasspathModule) checkUpdatableModules(ctx android.ModuleContext, modules []android.Module) {
	for _, m := range modules {
		apexInfo := ctx.OtherModuleProvider(m, android.ApexInfoProvider).(android.ApexInfo)
		fromUpdatableApex := apexInfo.Updatable
		if fromUpdatableApex {
			// ok: this jar is part of an updatable apex
		} else {
			name := ctx.OtherModuleName(m)
			if apexInfo.IsForPlatform() {
				// error: this jar is part of the platform
				ctx.ModuleErrorf("module %q from platform is not allowed in the updatable boot jars list", name)
			} else {
				// TODO(b/177892522): Treat this as an error.
				// Cannot do that at the moment because framework-wifi and framework-tethering are in the
				// PRODUCT_UPDATABLE_BOOT_JARS but not marked as updatable in AOSP.
			}
		}
	}
}

func (b *platformBootclasspathModule) getImageConfig(ctx android.EarlyModuleContext) *bootImageConfig {
	return defaultBootImageConfig(ctx)
}

// generateHiddenAPIBuildActions generates all the hidden API related build rules.
func (b *platformBootclasspathModule) generateHiddenAPIBuildActions(ctx android.ModuleContext, modules []android.Module, fragments []android.Module) {

	// Save the paths to the monolithic files for retrieval via OutputFiles().
	b.hiddenAPIFlagsCSV = hiddenAPISingletonPaths(ctx).flags
	b.hiddenAPIIndexCSV = hiddenAPISingletonPaths(ctx).index
	b.hiddenAPIMetadataCSV = hiddenAPISingletonPaths(ctx).metadata

	// Don't run any hiddenapi rules if UNSAFE_DISABLE_HIDDENAPI_FLAGS=true. This is a performance
	// optimization that can be used to reduce the incremental build time but as its name suggests it
	// can be unsafe to use, e.g. when the changes affect anything that goes on the bootclasspath.
	if ctx.Config().IsEnvTrue("UNSAFE_DISABLE_HIDDENAPI_FLAGS") {
		paths := android.OutputPaths{b.hiddenAPIFlagsCSV, b.hiddenAPIIndexCSV, b.hiddenAPIMetadataCSV}
		for _, path := range paths {
			ctx.Build(pctx, android.BuildParams{
				Rule:   android.Touch,
				Output: path,
			})
		}
		return
	}

	hiddenAPISupportingModules := []hiddenAPISupportingModule{}
	for _, module := range modules {
		if h, ok := module.(hiddenAPISupportingModule); ok {
			if h.bootDexJar() == nil {
				ctx.ModuleErrorf("module %s does not provide a bootDexJar file", module)
			}
			if h.flagsCSV() == nil {
				ctx.ModuleErrorf("module %s does not provide a flagsCSV file", module)
			}
			if h.indexCSV() == nil {
				ctx.ModuleErrorf("module %s does not provide an indexCSV file", module)
			}
			if h.metadataCSV() == nil {
				ctx.ModuleErrorf("module %s does not provide a metadataCSV file", module)
			}

			if ctx.Failed() {
				continue
			}

			hiddenAPISupportingModules = append(hiddenAPISupportingModules, h)
		} else {
			ctx.ModuleErrorf("module %s of type %s does not support hidden API processing", module, ctx.OtherModuleType(module))
		}
	}

	moduleSpecificFlagsPaths := android.Paths{}
	for _, module := range hiddenAPISupportingModules {
		moduleSpecificFlagsPaths = append(moduleSpecificFlagsPaths, module.flagsCSV())
	}

	flagFileInfo := b.properties.Hidden_api.hiddenAPIFlagFileInfo(ctx)
	for _, fragment := range fragments {
		if ctx.OtherModuleHasProvider(fragment, hiddenAPIFlagFileInfoProvider) {
			info := ctx.OtherModuleProvider(fragment, hiddenAPIFlagFileInfoProvider).(hiddenAPIFlagFileInfo)
			flagFileInfo.append(info)
		}
	}

	// Store the information for testing.
	ctx.SetProvider(hiddenAPIFlagFileInfoProvider, flagFileInfo)

	outputPath := hiddenAPISingletonPaths(ctx).flags
	baseFlagsPath := hiddenAPISingletonPaths(ctx).stubFlags
	ruleToGenerateHiddenApiFlags(ctx, outputPath, baseFlagsPath, moduleSpecificFlagsPaths, flagFileInfo)

	b.generateHiddenAPIStubFlagsRules(ctx, hiddenAPISupportingModules)
	b.generateHiddenAPIIndexRules(ctx, hiddenAPISupportingModules)
	b.generatedHiddenAPIMetadataRules(ctx, hiddenAPISupportingModules)
}

func (b *platformBootclasspathModule) generateHiddenAPIStubFlagsRules(ctx android.ModuleContext, modules []hiddenAPISupportingModule) {
	bootDexJars := android.Paths{}
	for _, module := range modules {
		bootDexJars = append(bootDexJars, module.bootDexJar())
	}

	sdkKindToStubPaths := hiddenAPIGatherStubLibDexJarPaths(ctx)

	outputPath := hiddenAPISingletonPaths(ctx).stubFlags
	rule := ruleToGenerateHiddenAPIStubFlagsFile(ctx, outputPath, bootDexJars, sdkKindToStubPaths)
	rule.Build("platform-bootclasspath-monolithic-hiddenapi-stub-flags", "monolithic hidden API stub flags")
}

func (b *platformBootclasspathModule) generateHiddenAPIIndexRules(ctx android.ModuleContext, modules []hiddenAPISupportingModule) {
	indexes := android.Paths{}
	for _, module := range modules {
		indexes = append(indexes, module.indexCSV())
	}

	rule := android.NewRuleBuilder(pctx, ctx)
	rule.Command().
		BuiltTool("merge_csv").
		Flag("--key_field signature").
		FlagWithArg("--header=", "signature,file,startline,startcol,endline,endcol,properties").
		FlagWithOutput("--output=", hiddenAPISingletonPaths(ctx).index).
		Inputs(indexes)
	rule.Build("platform-bootclasspath-monolithic-hiddenapi-index", "monolithic hidden API index")
}

func (b *platformBootclasspathModule) generatedHiddenAPIMetadataRules(ctx android.ModuleContext, modules []hiddenAPISupportingModule) {
	metadataCSVFiles := android.Paths{}
	for _, module := range modules {
		metadataCSVFiles = append(metadataCSVFiles, module.metadataCSV())
	}

	rule := android.NewRuleBuilder(pctx, ctx)

	outputPath := hiddenAPISingletonPaths(ctx).metadata

	rule.Command().
		BuiltTool("merge_csv").
		Flag("--key_field signature").
		FlagWithOutput("--output=", outputPath).
		Inputs(metadataCSVFiles)

	rule.Build("platform-bootclasspath-monolithic-hiddenapi-metadata", "monolithic hidden API metadata")
}

// generateBootImageBuildActions generates ninja rules related to the boot image creation.
func (b *platformBootclasspathModule) generateBootImageBuildActions(ctx android.ModuleContext, updatableModules []android.Module) {
	// Force the GlobalSoongConfig to be created and cached for use by the dex_bootjars
	// GenerateSingletonBuildActions method as it cannot create it for itself.
	dexpreopt.GetGlobalSoongConfig(ctx)

	imageConfig := b.getImageConfig(ctx)
	if imageConfig == nil {
		return
	}

	global := dexpreopt.GetGlobalConfig(ctx)

	// Skip recompiling the boot image for the second sanitization phase. We'll get separate paths
	// and invalidate first-stage artifacts which are crucial to SANITIZE_LITE builds.
	// Note: this is technically incorrect. Compiled code contains stack checks which may depend
	//       on ASAN settings.
	if len(ctx.Config().SanitizeDevice()) == 1 &&
		ctx.Config().SanitizeDevice()[0] == "address" &&
		global.SanitizeLite {
		return
	}

	// Generate the framework profile rule
	bootFrameworkProfileRule(ctx, imageConfig)

	// Generate the updatable bootclasspath packages rule.
	generateUpdatableBcpPackagesRule(ctx, imageConfig, updatableModules)

	dumpOatRules(ctx, imageConfig)
}

// bootFrameworkProfileRule generates the rule to create the boot framework profile and
// returns a path to the generated file.
func bootFrameworkProfileRule(ctx android.ModuleContext, image *bootImageConfig) android.WritablePath {
	globalSoong := dexpreopt.GetGlobalSoongConfig(ctx)
	global := dexpreopt.GetGlobalConfig(ctx)

	if global.DisableGenerateProfile || ctx.Config().UnbundledBuild() {
		return nil
	}

	defaultProfile := "frameworks/base/config/boot-profile.txt"
	bootFrameworkProfile := android.PathForSource(ctx, defaultProfile)

	profile := image.dir.Join(ctx, "boot.bprof")

	rule := android.NewRuleBuilder(pctx, ctx)
	rule.Command().
		Text(`ANDROID_LOG_TAGS="*:e"`).
		Tool(globalSoong.Profman).
		Flag("--output-profile-type=bprof").
		FlagWithInput("--create-profile-from=", bootFrameworkProfile).
		FlagForEachInput("--apk=", image.dexPathsDeps.Paths()).
		FlagForEachArg("--dex-location=", image.getAnyAndroidVariant().dexLocationsDeps).
		FlagWithOutput("--reference-profile-file=", profile)

	rule.Install(profile, "/system/etc/boot-image.bprof")
	rule.Build("bootFrameworkProfile", "profile boot framework jars")
	image.profileInstalls = append(image.profileInstalls, rule.Installs()...)

	return profile
}

// generateUpdatableBcpPackagesRule generates the rule to create the updatable-bcp-packages.txt file
// and returns a path to the generated file.
func generateUpdatableBcpPackagesRule(ctx android.ModuleContext, image *bootImageConfig, updatableModules []android.Module) android.WritablePath {
	// Collect `permitted_packages` for updatable boot jars.
	var updatablePackages []string
	for _, module := range updatableModules {
		if j, ok := module.(PermittedPackagesForUpdatableBootJars); ok {
			pp := j.PermittedPackagesForUpdatableBootJars()
			if len(pp) > 0 {
				updatablePackages = append(updatablePackages, pp...)
			} else {
				ctx.ModuleErrorf("Missing permitted_packages")
			}
		}
	}

	// Sort updatable packages to ensure deterministic ordering.
	sort.Strings(updatablePackages)

	updatableBcpPackagesName := "updatable-bcp-packages.txt"
	updatableBcpPackages := image.dir.Join(ctx, updatableBcpPackagesName)

	// WriteFileRule automatically adds the last end-of-line.
	android.WriteFileRule(ctx, updatableBcpPackages, strings.Join(updatablePackages, "\n"))

	rule := android.NewRuleBuilder(pctx, ctx)
	rule.Install(updatableBcpPackages, "/system/etc/"+updatableBcpPackagesName)
	// TODO: Rename `profileInstalls` to `extraInstalls`?
	// Maybe even move the field out of the bootImageConfig into some higher level type?
	image.profileInstalls = append(image.profileInstalls, rule.Installs()...)

	return updatableBcpPackages
}

func dumpOatRules(ctx android.ModuleContext, image *bootImageConfig) {
	var allPhonies android.Paths
	for _, image := range image.variants {
		arch := image.target.Arch.ArchType
		suffix := arch.String()
		// Host and target might both use x86 arch. We need to ensure the names are unique.
		if image.target.Os.Class == android.Host {
			suffix = "host-" + suffix
		}
		// Create a rule to call oatdump.
		output := android.PathForOutput(ctx, "boot."+suffix+".oatdump.txt")
		rule := android.NewRuleBuilder(pctx, ctx)
		rule.Command().
			// TODO: for now, use the debug version for better error reporting
			BuiltTool("oatdumpd").
			FlagWithInputList("--runtime-arg -Xbootclasspath:", image.dexPathsDeps.Paths(), ":").
			FlagWithList("--runtime-arg -Xbootclasspath-locations:", image.dexLocationsDeps, ":").
			FlagWithArg("--image=", strings.Join(image.imageLocations(), ":")).Implicits(image.imagesDeps.Paths()).
			FlagWithOutput("--output=", output).
			FlagWithArg("--instruction-set=", arch.String())
		rule.Build("dump-oat-boot-"+suffix, "dump oat boot "+arch.String())

		// Create a phony rule that depends on the output file and prints the path.
		phony := android.PathForPhony(ctx, "dump-oat-boot-"+suffix)
		rule = android.NewRuleBuilder(pctx, ctx)
		rule.Command().
			Implicit(output).
			ImplicitOutput(phony).
			Text("echo").FlagWithArg("Output in ", output.String())
		rule.Build("phony-dump-oat-boot-"+suffix, "dump oat boot "+arch.String())

		allPhonies = append(allPhonies, phony)
	}

	phony := android.PathForPhony(ctx, "dump-oat-boot")
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.Phony,
		Output:      phony,
		Inputs:      allPhonies,
		Description: "dump-oat-boot",
	})
}
