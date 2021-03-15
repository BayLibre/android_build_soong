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
	"strings"

	"android/soong/android"
	"android/soong/dexpreopt"
	"android/soong/tradefed"
	"github.com/google/blueprint/proptools"
)

// This file contains the module implementations for java_import, java_import_host,
// java_test_import, and dex_import.

func init() {
	RegisterImportBuildComponents(android.InitRegistrationContext)
}

func RegisterImportBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("java_import", ImportFactory)
	ctx.RegisterModuleType("java_import_host", ImportFactoryHost)
	ctx.RegisterModuleType("java_test_import", JavaTestImportFactory)
	ctx.RegisterModuleType("dex_import", DexImportFactory)
}

//
// Java prebuilts
//

type ImportProperties struct {
	Jars []string `android:"path,arch_variant"`

	Sdk_version *string

	Installable *bool

	// List of shared java libs that this module has dependencies to
	Libs []string

	// List of files to remove from the jar file(s)
	Exclude_files []string

	// List of directories to remove from the jar file(s)
	Exclude_dirs []string

	// if set to true, run Jetifier against .jar file. Defaults to false.
	Jetifier *bool

	// set the name of the output
	Stem *string

	Aidl struct {
		// directories that should be added as include directories for any aidl sources of modules
		// that depend on this module, as well as to aidl for this module.
		Export_include_dirs []string
	}
}

type Import struct {
	android.ModuleBase
	android.DefaultableModuleBase
	android.ApexModuleBase
	prebuilt android.Prebuilt
	android.SdkBase

	// Functionality common to Module and Import.
	embeddableInModuleAndImport

	hiddenAPI
	dexer
	dexpreopter

	properties ImportProperties

	// output file containing classes.dex and resources
	dexJarFile android.Path

	combinedClasspathFile android.Path
	classLoaderContexts   dexpreopt.ClassLoaderContextMap
	exportAidlIncludeDirs android.Paths

	hideApexVariantFromMake bool
}

func (j *Import) sdkVersion() sdkSpec {
	return sdkSpecFrom(String(j.properties.Sdk_version))
}

func (j *Import) makeSdkVersion() string {
	return j.sdkVersion().raw
}

func (j *Import) systemModules() string {
	return "none"
}

func (j *Import) minSdkVersion() sdkSpec {
	return j.sdkVersion()
}

func (j *Import) targetSdkVersion() sdkSpec {
	return j.sdkVersion()
}

func (j *Import) MinSdkVersion() string {
	return j.minSdkVersion().version.String()
}

func (j *Import) Prebuilt() *android.Prebuilt {
	return &j.prebuilt
}

func (j *Import) PrebuiltSrcs() []string {
	return j.properties.Jars
}

func (j *Import) Name() string {
	return j.prebuilt.Name(j.ModuleBase.Name())
}

func (j *Import) Stem() string {
	return proptools.StringDefault(j.properties.Stem, j.ModuleBase.Name())
}

func (a *Import) JacocoReportClassesFile() android.Path {
	return nil
}

func (j *Import) LintDepSets() LintDepSets {
	return LintDepSets{}
}

func (j *Import) DepsMutator(ctx android.BottomUpMutatorContext) {
	ctx.AddVariationDependencies(nil, libTag, j.properties.Libs...)

	if ctx.Device() && Bool(j.dexProperties.Compile_dex) {
		sdkDeps(ctx, sdkContext(j), j.dexer)
	}
}

func (j *Import) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Initialize the hiddenapi structure.
	j.initHiddenAPI(ctx, j.BaseModuleName())

	if !ctx.Provider(android.ApexInfoProvider).(android.ApexInfo).IsForPlatform() {
		j.hideApexVariantFromMake = true
	}

	jars := android.PathsForModuleSrc(ctx, j.properties.Jars)

	jarName := j.Stem() + ".jar"
	outputFile := android.PathForModuleOut(ctx, "combined", jarName)
	TransformJarsToJar(ctx, outputFile, "for prebuilts", jars, android.OptionalPath{},
		false, j.properties.Exclude_files, j.properties.Exclude_dirs)
	if Bool(j.properties.Jetifier) {
		inputFile := outputFile
		outputFile = android.PathForModuleOut(ctx, "jetifier", jarName)
		TransformJetifier(ctx, outputFile, inputFile)
	}
	j.combinedClasspathFile = outputFile
	j.classLoaderContexts = make(dexpreopt.ClassLoaderContextMap)

	var flags javaBuilderFlags
	var deapexerModule android.Module

	ctx.VisitDirectDeps(func(module android.Module) {
		tag := ctx.OtherModuleDependencyTag(module)

		if ctx.OtherModuleHasProvider(module, JavaInfoProvider) {
			dep := ctx.OtherModuleProvider(module, JavaInfoProvider).(JavaInfo)
			switch tag {
			case libTag, staticLibTag:
				flags.classpath = append(flags.classpath, dep.HeaderJars...)
			case bootClasspathTag:
				flags.bootClasspath = append(flags.bootClasspath, dep.HeaderJars...)
			}
		} else if dep, ok := module.(SdkLibraryDependency); ok {
			switch tag {
			case libTag:
				flags.classpath = append(flags.classpath, dep.SdkHeaderJars(ctx, j.sdkVersion())...)
			}
		}

		addCLCFromDep(ctx, module, j.classLoaderContexts)

		// Save away the `deapexer` module on which this depends, if any.
		if tag == android.DeapexerTag {
			deapexerModule = module
		}
	})

	if Bool(j.properties.Installable) {
		ctx.InstallFile(android.PathForModuleInstall(ctx, "framework"),
			jarName, outputFile)
	}

	j.exportAidlIncludeDirs = android.PathsForModuleSrc(ctx, j.properties.Aidl.Export_include_dirs)

	if ctx.Device() {
		// If this is a variant created for a prebuilt_apex then use the dex implementation jar
		// obtained from the associated deapexer module.
		ai := ctx.Provider(android.ApexInfoProvider).(android.ApexInfo)
		if ai.ForPrebuiltApex {
			if deapexerModule == nil {
				// This should never happen as a variant for a prebuilt_apex is only created if the
				// deapxer module has been configured to export the dex implementation jar for this module.
				ctx.ModuleErrorf("internal error: module %q does not depend on a `deapexer` module for prebuilt_apex %q",
					j.Name(), ai.ApexVariationName)
			}

			// Get the path of the dex implementation jar from the `deapexer` module.
			di := ctx.OtherModuleProvider(deapexerModule, android.DeapexerProvider).(android.DeapexerInfo)
			if dexOutputPath := di.PrebuiltExportPath(j.BaseModuleName(), ".dexjar"); dexOutputPath != nil {
				j.dexJarFile = dexOutputPath
				j.hiddenAPIExtractInformation(ctx, dexOutputPath, outputFile)
			} else {
				// This should never happen as a variant for a prebuilt_apex is only created if the
				// prebuilt_apex has been configured to export the java library dex file.
				ctx.ModuleErrorf("internal error: no dex implementation jar available from prebuilt_apex %q", deapexerModule.Name())
			}
		} else if Bool(j.dexProperties.Compile_dex) {
			sdkDep := decodeSdkDep(ctx, sdkContext(j))
			if sdkDep.invalidVersion {
				ctx.AddMissingDependencies(sdkDep.bootclasspath)
				ctx.AddMissingDependencies(sdkDep.java9Classpath)
			} else if sdkDep.useFiles {
				// sdkDep.jar is actually equivalent to turbine header.jar.
				flags.classpath = append(flags.classpath, sdkDep.jars...)
			}

			// Dex compilation

			j.dexpreopter.installPath = android.PathForModuleInstall(ctx, "framework", jarName)
			if j.dexProperties.Uncompress_dex == nil {
				// If the value was not force-set by the user, use reasonable default based on the module.
				j.dexProperties.Uncompress_dex = proptools.BoolPtr(shouldUncompressDex(ctx, &j.dexpreopter))
			}
			j.dexpreopter.uncompressedDex = *j.dexProperties.Uncompress_dex

			var dexOutputFile android.OutputPath
			dexOutputFile = j.dexer.compileDex(ctx, flags, j.minSdkVersion(), outputFile, jarName)
			if ctx.Failed() {
				return
			}

			// Hidden API CSV generation and dex encoding
			dexOutputFile = j.hiddenAPIExtractAndEncode(ctx, dexOutputFile, outputFile,
				proptools.Bool(j.dexProperties.Uncompress_dex))

			j.dexJarFile = dexOutputFile
		}
	}

	ctx.SetProvider(JavaInfoProvider, JavaInfo{
		HeaderJars:                     android.PathsIfNonNil(j.combinedClasspathFile),
		ImplementationAndResourcesJars: android.PathsIfNonNil(j.combinedClasspathFile),
		ImplementationJars:             android.PathsIfNonNil(j.combinedClasspathFile),
		AidlIncludeDirs:                j.exportAidlIncludeDirs,
	})
}

func (j *Import) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case "", ".jar":
		return android.Paths{j.combinedClasspathFile}, nil
	default:
		return nil, fmt.Errorf("unsupported module reference tag %q", tag)
	}
}

var _ android.OutputFileProducer = (*Import)(nil)

func (j *Import) HeaderJars() android.Paths {
	if j.combinedClasspathFile == nil {
		return nil
	}
	return android.Paths{j.combinedClasspathFile}
}

func (j *Import) ImplementationAndResourcesJars() android.Paths {
	if j.combinedClasspathFile == nil {
		return nil
	}
	return android.Paths{j.combinedClasspathFile}
}

func (j *Import) DexJarBuildPath() android.Path {
	return j.dexJarFile
}

func (j *Import) DexJarInstallPath() android.Path {
	return nil
}

func (j *Import) ClassLoaderContexts() dexpreopt.ClassLoaderContextMap {
	return j.classLoaderContexts
}

var _ android.ApexModule = (*Import)(nil)

// Implements android.ApexModule
func (j *Import) DepIsInSameApex(ctx android.BaseModuleContext, dep android.Module) bool {
	return j.depIsInSameApex(ctx, dep)
}

// Implements android.ApexModule
func (j *Import) ShouldSupportSdkVersion(ctx android.BaseModuleContext,
	sdkVersion android.ApiLevel) error {
	// Do not check for prebuilts against the min_sdk_version of enclosing APEX
	return nil
}

// Add compile time check for interface implementation
var _ android.IDEInfo = (*Import)(nil)
var _ android.IDECustomizedModuleName = (*Import)(nil)

// Collect information for opening IDE project files in java/jdeps.go.
const (
	removedPrefix = "prebuilt_"
)

func (j *Import) IDEInfo(dpInfo *android.IdeInfo) {
	dpInfo.Jars = append(dpInfo.Jars, j.PrebuiltSrcs()...)
}

func (j *Import) IDECustomizedModuleName() string {
	// TODO(b/113562217): Extract the base module name from the Import name, often the Import name
	// has a prefix "prebuilt_". Remove the prefix explicitly if needed until we find a better
	// solution to get the Import name.
	name := j.Name()
	if strings.HasPrefix(name, removedPrefix) {
		name = strings.TrimPrefix(name, removedPrefix)
	}
	return name
}

var _ android.PrebuiltInterface = (*Import)(nil)

func (j *Import) IsInstallable() bool {
	return Bool(j.properties.Installable)
}

var _ dexpreopterInterface = (*Import)(nil)

// java_import imports one or more `.jar` files into the build graph as if they were built by a java_library module.
//
// By default, a java_import has a single variant that expects a `.jar` file containing `.class` files that were
// compiled against an Android classpath.
//
// Specifying `host_supported: true` will produce two variants, one for use as a dependency of device modules and one
// for host modules.
func ImportFactory() android.Module {
	module := &Import{}

	module.AddProperties(
		&module.properties,
		&module.dexer.dexProperties,
	)

	module.initModuleAndImport(&module.ModuleBase)

	module.dexProperties.Optimize.EnabledByDefault = false

	android.InitPrebuiltModule(module, &module.properties.Jars)
	android.InitApexModule(module)
	android.InitSdkAwareModule(module)
	InitJavaModule(module, android.HostAndDeviceSupported)
	return module
}

// java_import_host imports one or more `.jar` files into the build graph as if they were built by a
// java_library_host module.
//
// A java_import_host has a single variant that expects a `.jar` file containing `.class` files that were
// compiled against a host bootclasspath.
func ImportFactoryHost() android.Module {
	module := &Import{}

	module.AddProperties(&module.properties)

	android.InitPrebuiltModule(module, &module.properties.Jars)
	android.InitApexModule(module)
	InitJavaModule(module, android.HostSupported)
	return module
}

type JavaTestImport struct {
	Import

	prebuiltTestProperties prebuiltTestProperties

	testConfig android.Path
	dexJarFile android.Path
}

func (j *JavaTestImport) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	j.testConfig = tradefed.AutoGenJavaTestConfig(ctx, j.prebuiltTestProperties.Test_config, nil,
		j.prebuiltTestProperties.Test_suites, nil, nil)

	j.Import.GenerateAndroidBuildActions(ctx)
}

// java_test_import imports one or more `.jar` files into the build graph as if they were built by a java_test module
// and makes sure that it is added to the appropriate test suite.
//
// By default, a java_test_import has a single variant that expects a `.jar` file containing `.class` files that were
// compiled against an Android classpath.
//
// Specifying `host_supported: true` will produce two variants, one for use as a dependency of device modules and one
// for host modules.
func JavaTestImportFactory() android.Module {
	module := &JavaTestImport{}

	module.AddProperties(
		&module.Import.properties,
		&module.prebuiltTestProperties)

	module.Import.properties.Installable = proptools.BoolPtr(true)

	android.InitPrebuiltModule(module, &module.properties.Jars)
	android.InitApexModule(module)
	android.InitSdkAwareModule(module)
	InitJavaModule(module, android.HostAndDeviceSupported)
	return module
}

// dex_import module

type DexImportProperties struct {
	Jars []string `android:"path"`

	// set the name of the output
	Stem *string
}

type DexImport struct {
	android.ModuleBase
	android.DefaultableModuleBase
	android.ApexModuleBase
	prebuilt android.Prebuilt

	properties DexImportProperties

	dexJarFile android.Path

	dexpreopter

	hideApexVariantFromMake bool
}

func (j *DexImport) Prebuilt() *android.Prebuilt {
	return &j.prebuilt
}

func (j *DexImport) PrebuiltSrcs() []string {
	return j.properties.Jars
}

func (j *DexImport) Name() string {
	return j.prebuilt.Name(j.ModuleBase.Name())
}

func (j *DexImport) Stem() string {
	return proptools.StringDefault(j.properties.Stem, j.ModuleBase.Name())
}

func (a *DexImport) JacocoReportClassesFile() android.Path {
	return nil
}

func (a *DexImport) LintDepSets() LintDepSets {
	return LintDepSets{}
}

func (j *DexImport) IsInstallable() bool {
	return true
}

func (j *DexImport) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if len(j.properties.Jars) != 1 {
		ctx.PropertyErrorf("jars", "exactly one jar must be provided")
	}

	apexInfo := ctx.Provider(android.ApexInfoProvider).(android.ApexInfo)
	if !apexInfo.IsForPlatform() {
		j.hideApexVariantFromMake = true
	}

	j.dexpreopter.installPath = android.PathForModuleInstall(ctx, "framework", j.Stem()+".jar")
	j.dexpreopter.uncompressedDex = shouldUncompressDex(ctx, &j.dexpreopter)

	inputJar := ctx.ExpandSource(j.properties.Jars[0], "jars")
	dexOutputFile := android.PathForModuleOut(ctx, ctx.ModuleName()+".jar")

	if j.dexpreopter.uncompressedDex {
		rule := android.NewRuleBuilder(pctx, ctx)

		temporary := android.PathForModuleOut(ctx, ctx.ModuleName()+".jar.unaligned")
		rule.Temporary(temporary)

		// use zip2zip to uncompress classes*.dex files
		rule.Command().
			BuiltTool("zip2zip").
			FlagWithInput("-i ", inputJar).
			FlagWithOutput("-o ", temporary).
			FlagWithArg("-0 ", "'classes*.dex'")

		// use zipalign to align uncompressed classes*.dex files
		rule.Command().
			BuiltTool("zipalign").
			Flag("-f").
			Text("4").
			Input(temporary).
			Output(dexOutputFile)

		rule.DeleteTemporaryFiles()

		rule.Build("uncompress_dex", "uncompress dex")
	} else {
		ctx.Build(pctx, android.BuildParams{
			Rule:   android.Cp,
			Input:  inputJar,
			Output: dexOutputFile,
		})
	}

	j.dexJarFile = dexOutputFile

	j.dexpreopt(ctx, dexOutputFile)

	if apexInfo.IsForPlatform() {
		ctx.InstallFile(android.PathForModuleInstall(ctx, "framework"),
			j.Stem()+".jar", dexOutputFile)
	}
}

func (j *DexImport) DexJarBuildPath() android.Path {
	return j.dexJarFile
}

var _ android.ApexModule = (*DexImport)(nil)

// Implements android.ApexModule
func (j *DexImport) ShouldSupportSdkVersion(ctx android.BaseModuleContext,
	sdkVersion android.ApiLevel) error {
	// we don't check prebuilt modules for sdk_version
	return nil
}

// dex_import imports a `.jar` file containing classes.dex files.
//
// A dex_import module cannot be used as a dependency of a java_* or android_* module, it can only be installed
// to the device.
func DexImportFactory() android.Module {
	module := &DexImport{}

	module.AddProperties(&module.properties)

	android.InitPrebuiltModule(module, &module.properties.Jars)
	android.InitApexModule(module)
	InitJavaModule(module, android.DeviceSupported)
	return module
}
