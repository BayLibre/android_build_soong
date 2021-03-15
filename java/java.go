// Copyright 2015 Google Inc. All rights reserved.
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

// This file contains various interfaces, structs, and functions that facilitate interactions within
// or across packages.

import (
	"fmt"
	"path/filepath"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/dexpreopt"
	"android/soong/java/config"
)

func init() {
	RegisterJavaSdkMemberTypes()
}

func RegisterJavaSdkMemberTypes() {
	// Register sdk member types.
	android.RegisterSdkMemberType(javaHeaderLibsSdkMemberType)

	// Export implementation classes jar as part of the sdk.
	exportImplementationClassesJar := func(_ android.SdkMemberContext, j *Library) android.Path {
		implementationJars := j.ImplementationAndResourcesJars()
		if len(implementationJars) != 1 {
			panic(fmt.Errorf("there must be only one implementation jar from %q", j.Name()))
		}
		return implementationJars[0]
	}

	// Register java implementation libraries for use only in module_exports (not sdk).
	android.RegisterSdkMemberType(&librarySdkMemberType{
		android.SdkMemberTypeBase{
			PropertyName: "java_libs",
		},
		exportImplementationClassesJar,
		sdkSnapshotFilePathForJar,
		copyEverythingToSnapshot,
	})

	// Register java boot libraries for use in sdk.
	//
	// The build has some implicit dependencies (via the boot jars configuration) on a number of
	// modules, e.g. core-oj, apache-xml, that are part of the java boot class path and which are
	// provided by mainline modules (e.g. art, conscrypt, runtime-i18n) but which are not otherwise
	// used outside those mainline modules.
	//
	// As they are not needed outside the mainline modules adding them to the sdk/module-exports as
	// either java_libs, or java_header_libs would end up exporting more information than was strictly
	// necessary. The java_boot_libs property to allow those modules to be exported as part of the
	// sdk/module_exports without exposing any unnecessary information.
	android.RegisterSdkMemberType(&librarySdkMemberType{
		android.SdkMemberTypeBase{
			PropertyName: "java_boot_libs",
			SupportsSdk:  true,
		},
		// Temporarily export implementation classes jar for java_boot_libs as it is required for the
		// hiddenapi processing.
		// TODO(b/179354495): Revert once hiddenapi processing has been modularized.
		exportImplementationClassesJar,
		sdkSnapshotFilePathForJar,
		onlyCopyJarToSnapshot,
	})

	// Register java test libraries for use only in module_exports (not sdk).
	android.RegisterSdkMemberType(&testSdkMemberType{
		SdkMemberTypeBase: android.SdkMemberTypeBase{
			PropertyName: "java_tests",
		},
	})

}

// JavaInfo contains information about a java module for use by modules that depend on it.
type JavaInfo struct {
	// HeaderJars is a list of jars that can be passed as the javac classpath in order to link
	// against this module.  If empty, ImplementationJars should be used instead.
	HeaderJars android.Paths

	// ImplementationAndResourceJars is a list of jars that contain the implementations of classes
	// in the module as well as any resources included in the module.
	ImplementationAndResourcesJars android.Paths

	// ImplementationJars is a list of jars that contain the implementations of classes in the
	//module.
	ImplementationJars android.Paths

	// ResourceJars is a list of jars that contain the resources included in the module.
	ResourceJars android.Paths

	// AidlIncludeDirs is a list of directories that should be passed to the aidl tool when
	// depending on this module.
	AidlIncludeDirs android.Paths

	// SrcJarArgs is a list of arguments to pass to soong_zip to package the sources of this
	// module.
	SrcJarArgs []string

	// SrcJarDeps is a list of paths to depend on when packaging the sources of this module.
	SrcJarDeps android.Paths

	// ExportedPlugins is a list of paths that should be used as annotation processors for any
	// module that depends on this module.
	ExportedPlugins android.Paths

	// ExportedPluginClasses is a list of classes that should be run as annotation processors for
	// any module that depends on this module.
	ExportedPluginClasses []string

	// ExportedPluginDisableTurbine is true if this module's annotation processors generate APIs,
	// requiring disbling turbine for any modules that depend on it.
	ExportedPluginDisableTurbine bool

	// JacocoReportClassesFile is the path to a jar containing uninstrumented classes that will be
	// instrumented by jacoco.
	JacocoReportClassesFile android.Path
}

var JavaInfoProvider = blueprint.NewProvider(JavaInfo{})

// SyspropPublicStubInfo contains info about the sysprop public stub library that corresponds to
// the sysprop implementation library.
type SyspropPublicStubInfo struct {
	// JavaInfo is the JavaInfoProvider of the sysprop public stub library that corresponds to
	// the sysprop implementation library.
	JavaInfo JavaInfo
}

var SyspropPublicStubInfoProvider = blueprint.NewProvider(SyspropPublicStubInfo{})

// Methods that need to be implemented for a module that is added to apex java_libs property.
type ApexDependency interface {
	HeaderJars() android.Paths
	ImplementationAndResourcesJars() android.Paths
}

// Provides build path and install path to DEX jars.
type UsesLibraryDependency interface {
	DexJarBuildPath() android.Path
	DexJarInstallPath() android.Path
	ClassLoaderContexts() dexpreopt.ClassLoaderContextMap
}

type dependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

// installDependencyTag is a dependency tag that is annotated to cause the installed files of the
// dependency to be installed when the parent module is installed.
type installDependencyTag struct {
	blueprint.BaseDependencyTag
	android.InstallAlwaysNeededDependencyTag
	name string
}

type usesLibraryDependencyTag struct {
	dependencyTag
	sdkVersion int // SDK version in which the library appared as a standalone library.
}

func makeUsesLibraryDependencyTag(sdkVersion int) usesLibraryDependencyTag {
	return usesLibraryDependencyTag{
		dependencyTag: dependencyTag{name: fmt.Sprintf("uses-library-%d", sdkVersion)},
		sdkVersion:    sdkVersion,
	}
}

func IsJniDepTag(depTag blueprint.DependencyTag) bool {
	return depTag == jniLibTag
}

var (
	dataNativeBinsTag       = dependencyTag{name: "dataNativeBins"}
	staticLibTag            = dependencyTag{name: "staticlib"}
	libTag                  = dependencyTag{name: "javalib"}
	java9LibTag             = dependencyTag{name: "java9lib"}
	pluginTag               = dependencyTag{name: "plugin"}
	errorpronePluginTag     = dependencyTag{name: "errorprone-plugin"}
	exportedPluginTag       = dependencyTag{name: "exported-plugin"}
	bootClasspathTag        = dependencyTag{name: "bootclasspath"}
	systemModulesTag        = dependencyTag{name: "system modules"}
	frameworkResTag         = dependencyTag{name: "framework-res"}
	kotlinStdlibTag         = dependencyTag{name: "kotlin-stdlib"}
	kotlinAnnotationsTag    = dependencyTag{name: "kotlin-annotations"}
	proguardRaiseTag        = dependencyTag{name: "proguard-raise"}
	certificateTag          = dependencyTag{name: "certificate"}
	instrumentationForTag   = dependencyTag{name: "instrumentation_for"}
	extraLintCheckTag       = dependencyTag{name: "extra-lint-check"}
	jniLibTag               = dependencyTag{name: "jnilib"}
	syspropPublicStubDepTag = dependencyTag{name: "sysprop public stub"}
	jniInstallTag           = installDependencyTag{name: "jni install"}
	binaryInstallTag        = installDependencyTag{name: "binary install"}
	usesLibTag              = makeUsesLibraryDependencyTag(dexpreopt.AnySdkVersion)
	usesLibCompat28Tag      = makeUsesLibraryDependencyTag(28)
	usesLibCompat29Tag      = makeUsesLibraryDependencyTag(29)
	usesLibCompat30Tag      = makeUsesLibraryDependencyTag(30)
)

func IsLibDepTag(depTag blueprint.DependencyTag) bool {
	return depTag == libTag
}

func IsStaticLibDepTag(depTag blueprint.DependencyTag) bool {
	return depTag == staticLibTag
}

type sdkDep struct {
	useModule, useFiles, invalidVersion bool

	// The modules that will be added to the bootclasspath when targeting 1.8 or lower
	bootclasspath []string

	// The default system modules to use. Will be an empty string if no system
	// modules are to be used.
	systemModules string

	// The modules that will be added to the classpath regardless of the Java language level targeted
	classpath []string

	// The modules that will be added ot the classpath when targeting 1.9 or higher
	// (normally these will be on the bootclasspath when targeting 1.8 or lower)
	java9Classpath []string

	frameworkResModule string

	jars android.Paths
	aidl android.OptionalPath

	noStandardLibs, noFrameworksLibs bool
}

func (s sdkDep) hasStandardLibs() bool {
	return !s.noStandardLibs
}

func (s sdkDep) hasFrameworkLibs() bool {
	return !s.noStandardLibs && !s.noFrameworksLibs
}

type jniLib struct {
	name           string
	path           android.Path
	target         android.Target
	coverageFile   android.OptionalPath
	unstrippedFile android.Path
}

func sdkDeps(ctx android.BottomUpMutatorContext, sdkContext sdkContext, d dexer) {
	sdkDep := decodeSdkDep(ctx, sdkContext)
	if sdkDep.useModule {
		ctx.AddVariationDependencies(nil, bootClasspathTag, sdkDep.bootclasspath...)
		ctx.AddVariationDependencies(nil, java9LibTag, sdkDep.java9Classpath...)
		ctx.AddVariationDependencies(nil, libTag, sdkDep.classpath...)
		if d.effectiveOptimizeEnabled() && sdkDep.hasStandardLibs() {
			ctx.AddVariationDependencies(nil, proguardRaiseTag, config.LegacyCorePlatformBootclasspathLibraries...)
		}
		if d.effectiveOptimizeEnabled() && sdkDep.hasFrameworkLibs() {
			ctx.AddVariationDependencies(nil, proguardRaiseTag, config.FrameworkLibraries...)
		}
	}
	if sdkDep.systemModules != "" {
		ctx.AddVariationDependencies(nil, systemModulesTag, sdkDep.systemModules)
	}
}

type deps struct {
	classpath               classpath
	java9Classpath          classpath
	bootClasspath           classpath
	processorPath           classpath
	errorProneProcessorPath classpath
	processorClasses        []string
	staticJars              android.Paths
	staticHeaderJars        android.Paths
	staticResourceJars      android.Paths
	aidlIncludeDirs         android.Paths
	srcs                    android.Paths
	srcJars                 android.Paths
	systemModules           *systemModules
	aidlPreprocess          android.OptionalPath
	kotlinStdlib            android.Paths
	kotlinAnnotations       android.Paths

	disableTurbine bool
}

func checkProducesJars(ctx android.ModuleContext, dep android.SourceFileProducer) {
	for _, f := range dep.Srcs() {
		if f.Ext() != ".jar" {
			ctx.ModuleErrorf("genrule %q must generate files ending with .jar to be used as a libs or static_libs dependency",
				ctx.OtherModuleName(dep.(blueprint.Module)))
		}
	}
}

func getJavaVersion(ctx android.ModuleContext, javaVersion string, sdkContext sdkContext) javaVersion {
	if javaVersion != "" {
		return normalizeJavaVersion(ctx, javaVersion)
	} else if ctx.Device() {
		return sdkContext.sdkVersion().defaultJavaLanguageVersion(ctx)
	} else {
		return JAVA_VERSION_9
	}
}

type javaVersion int

const (
	JAVA_VERSION_UNSUPPORTED = 0
	JAVA_VERSION_6           = 6
	JAVA_VERSION_7           = 7
	JAVA_VERSION_8           = 8
	JAVA_VERSION_9           = 9
)

func (v javaVersion) String() string {
	switch v {
	case JAVA_VERSION_6:
		return "1.6"
	case JAVA_VERSION_7:
		return "1.7"
	case JAVA_VERSION_8:
		return "1.8"
	case JAVA_VERSION_9:
		return "1.9"
	default:
		return "unsupported"
	}
}

// Returns true if javac targeting this version uses system modules instead of a bootclasspath.
func (v javaVersion) usesJavaModules() bool {
	return v >= 9
}

func normalizeJavaVersion(ctx android.BaseModuleContext, javaVersion string) javaVersion {
	switch javaVersion {
	case "1.6", "6":
		return JAVA_VERSION_6
	case "1.7", "7":
		return JAVA_VERSION_7
	case "1.8", "8":
		return JAVA_VERSION_8
	case "1.9", "9":
		return JAVA_VERSION_9
	case "10", "11":
		ctx.PropertyErrorf("java_version", "Java language levels above 9 are not supported")
		return JAVA_VERSION_UNSUPPORTED
	default:
		ctx.PropertyErrorf("java_version", "Unrecognized Java language level")
		return JAVA_VERSION_UNSUPPORTED
	}
}

const (
	aidlIncludeDir   = "aidl"
	javaDir          = "java"
	jarFileSuffix    = ".jar"
	testConfigSuffix = "-AndroidTest.xml"
)

// path to the jar file of a java library. Relative to <sdk_root>/<api_dir>
func sdkSnapshotFilePathForJar(osPrefix, name string) string {
	return sdkSnapshotFilePathForMember(osPrefix, name, jarFileSuffix)
}

func sdkSnapshotFilePathForMember(osPrefix, name string, suffix string) string {
	return filepath.Join(javaDir, osPrefix, name+suffix)
}

type librarySdkMemberType struct {
	android.SdkMemberTypeBase

	// Function to retrieve the appropriate output jar (implementation or header) from
	// the library.
	jarToExportGetter func(ctx android.SdkMemberContext, j *Library) android.Path

	// Function to compute the snapshot relative path to which the named library's
	// jar should be copied.
	snapshotPathGetter func(osPrefix, name string) string

	// True if only the jar should be copied to the snapshot, false if the jar plus any additional
	// files like aidl files should also be copied.
	onlyCopyJarToSnapshot bool
}

const (
	onlyCopyJarToSnapshot    = true
	copyEverythingToSnapshot = false
)

func (mt *librarySdkMemberType) AddDependencies(mctx android.BottomUpMutatorContext, dependencyTag blueprint.DependencyTag, names []string) {
	mctx.AddVariationDependencies(nil, dependencyTag, names...)
}

func (mt *librarySdkMemberType) IsInstance(module android.Module) bool {
	_, ok := module.(*Library)
	return ok
}

func (mt *librarySdkMemberType) AddPrebuiltModule(ctx android.SdkMemberContext, member android.SdkMember) android.BpModule {
	return ctx.SnapshotBuilder().AddPrebuiltModule(member, "java_import")
}

func (mt *librarySdkMemberType) CreateVariantPropertiesStruct() android.SdkMemberProperties {
	return &librarySdkMemberProperties{}
}

type librarySdkMemberProperties struct {
	android.SdkMemberPropertiesBase

	JarToExport     android.Path `android:"arch_variant"`
	AidlIncludeDirs android.Paths
}

func (p *librarySdkMemberProperties) PopulateFromVariant(ctx android.SdkMemberContext, variant android.Module) {
	j := variant.(*Library)

	p.JarToExport = ctx.MemberType().(*librarySdkMemberType).jarToExportGetter(ctx, j)

	p.AidlIncludeDirs = j.AidlIncludeDirs()
}

func (p *librarySdkMemberProperties) AddToPropertySet(ctx android.SdkMemberContext, propertySet android.BpPropertySet) {
	builder := ctx.SnapshotBuilder()

	memberType := ctx.MemberType().(*librarySdkMemberType)

	exportedJar := p.JarToExport
	if exportedJar != nil {
		// Delegate the creation of the snapshot relative path to the member type.
		snapshotRelativeJavaLibPath := memberType.snapshotPathGetter(p.OsPrefix(), ctx.Name())

		// Copy the exported jar to the snapshot.
		builder.CopyToSnapshot(exportedJar, snapshotRelativeJavaLibPath)

		propertySet.AddProperty("jars", []string{snapshotRelativeJavaLibPath})
	}

	// Do not copy anything else to the snapshot.
	if memberType.onlyCopyJarToSnapshot {
		return
	}

	aidlIncludeDirs := p.AidlIncludeDirs
	if len(aidlIncludeDirs) != 0 {
		sdkModuleContext := ctx.SdkModuleContext()
		for _, dir := range aidlIncludeDirs {
			// TODO(jiyong): copy parcelable declarations only
			aidlFiles, _ := sdkModuleContext.GlobWithDeps(dir.String()+"/**/*.aidl", nil)
			for _, file := range aidlFiles {
				builder.CopyToSnapshot(android.PathForSource(sdkModuleContext, file), filepath.Join(aidlIncludeDir, file))
			}
		}

		// TODO(b/151933053) - add aidl include dirs property
	}
}

var javaHeaderLibsSdkMemberType android.SdkMemberType = &librarySdkMemberType{
	android.SdkMemberTypeBase{
		PropertyName: "java_header_libs",
		SupportsSdk:  true,
	},
	func(_ android.SdkMemberContext, j *Library) android.Path {
		headerJars := j.HeaderJars()
		if len(headerJars) != 1 {
			panic(fmt.Errorf("there must be only one header jar from %q", j.Name()))
		}

		return headerJars[0]
	},
	sdkSnapshotFilePathForJar,
	copyEverythingToSnapshot,
}

type testSdkMemberType struct {
	android.SdkMemberTypeBase
}

func (mt *testSdkMemberType) AddDependencies(mctx android.BottomUpMutatorContext, dependencyTag blueprint.DependencyTag, names []string) {
	mctx.AddVariationDependencies(nil, dependencyTag, names...)
}

func (mt *testSdkMemberType) IsInstance(module android.Module) bool {
	_, ok := module.(*Test)
	return ok
}

func (mt *testSdkMemberType) AddPrebuiltModule(ctx android.SdkMemberContext, member android.SdkMember) android.BpModule {
	return ctx.SnapshotBuilder().AddPrebuiltModule(member, "java_test_import")
}

func (mt *testSdkMemberType) CreateVariantPropertiesStruct() android.SdkMemberProperties {
	return &testSdkMemberProperties{}
}

type testSdkMemberProperties struct {
	android.SdkMemberPropertiesBase

	JarToExport android.Path
	TestConfig  android.Path
}

func (p *testSdkMemberProperties) PopulateFromVariant(ctx android.SdkMemberContext, variant android.Module) {
	test := variant.(*Test)

	implementationJars := test.ImplementationJars()
	if len(implementationJars) != 1 {
		panic(fmt.Errorf("there must be only one implementation jar from %q", test.Name()))
	}

	p.JarToExport = implementationJars[0]
	p.TestConfig = test.testConfig
}

func (p *testSdkMemberProperties) AddToPropertySet(ctx android.SdkMemberContext, propertySet android.BpPropertySet) {
	builder := ctx.SnapshotBuilder()

	exportedJar := p.JarToExport
	if exportedJar != nil {
		snapshotRelativeJavaLibPath := sdkSnapshotFilePathForJar(p.OsPrefix(), ctx.Name())
		builder.CopyToSnapshot(exportedJar, snapshotRelativeJavaLibPath)

		propertySet.AddProperty("jars", []string{snapshotRelativeJavaLibPath})
	}

	testConfig := p.TestConfig
	if testConfig != nil {
		snapshotRelativeTestConfigPath := sdkSnapshotFilePathForMember(p.OsPrefix(), ctx.Name(), testConfigSuffix)
		builder.CopyToSnapshot(testConfig, snapshotRelativeTestConfigPath)
		propertySet.AddProperty("test_config", snapshotRelativeTestConfigPath)
	}
}

var Bool = proptools.Bool
var BoolDefault = proptools.BoolDefault
var String = proptools.String
var inList = android.InList

// Add class loader context (CLC) of a given dependency to the current CLC.
func addCLCFromDep(ctx android.ModuleContext, depModule android.Module,
	clcMap dexpreopt.ClassLoaderContextMap) {

	dep, ok := depModule.(UsesLibraryDependency)
	if !ok {
		return
	}

	// Find out if the dependency is either an SDK library or an ordinary library that is disguised
	// as an SDK library by the means of `provides_uses_lib` property. If yes, the library is itself
	// a <uses-library> and should be added as a node in the CLC tree, and its CLC should be added
	// as subtree of that node. Otherwise the library is not a <uses_library> and should not be
	// added to CLC, but the transitive <uses-library> dependencies from its CLC should be added to
	// the current CLC.
	var implicitSdkLib *string
	comp, isComp := depModule.(SdkLibraryComponentDependency)
	if isComp {
		implicitSdkLib = comp.OptionalImplicitSdkLibrary()
		// OptionalImplicitSdkLibrary() may be nil so need to fall through to ProvidesUsesLib().
	}
	if implicitSdkLib == nil {
		if ulib, ok := depModule.(ProvidesUsesLib); ok {
			implicitSdkLib = ulib.ProvidesUsesLib()
		}
	}

	depTag := ctx.OtherModuleDependencyTag(depModule)
	if depTag == libTag || depTag == usesLibTag {
		// Ok, propagate <uses-library> through non-static library dependencies.
	} else if depTag == staticLibTag {
		// Propagate <uses-library> through static library dependencies, unless it is a component
		// library (such as stubs). Component libraries have a dependency on their SDK library,
		// which should not be pulled just because of a static component library.
		if implicitSdkLib != nil {
			return
		}
	} else {
		// Don't propagate <uses-library> for other dependency tags.
		return
	}

	if implicitSdkLib != nil {
		clcMap.AddContext(ctx, dexpreopt.AnySdkVersion, *implicitSdkLib,
			dep.DexJarBuildPath(), dep.DexJarInstallPath(), dep.ClassLoaderContexts())
	} else {
		depName := ctx.OtherModuleName(depModule)
		clcMap.AddContextMap(dep.ClassLoaderContexts(), depName)
	}
}
