// Copyright 2018 Google Inc. All rights reserved.
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
	"android/soong/android"
	"android/soong/dexpreopt"
)

type dexpreopterInterface interface {
	IsInstallable() bool // Structs that embed dexpreopter must implement this.
	dexpreoptDisabled(ctx android.BaseModuleContext) bool
}

type dexpreopter struct {
	dexpreoptProperties DexpreoptProperties

	installPath         android.InstallPath
	uncompressedDex     bool
	isSDKLibrary        bool
	isApp               bool
	isTest              bool
	isPresignedPrebuilt bool

	manifestFile        android.Path
	statusFile          android.WritablePath
	enforceUsesLibs     bool
	classLoaderContexts dexpreopt.ClassLoaderContextMap

	builtInstalled string

	// The config is used for two purposes:
	// - Passing dexpreopt information about libraries from Soong to Make. This is needed when
	//   a <uses-library> is defined in Android.bp, but used in Android.mk (see dex_preopt_config_merger.py).
	//   Note that dexpreopt.config might be needed even if dexpreopt is disabled for the library itself.
	// - Dexpreopt post-processing (using dexpreopt artifacts from a prebuilt system image to incrementally
	//   dexpreopt another partition).
	configPath android.WritablePath
}

type DexpreoptProperties struct {
	Dex_preopt struct {
		// If false, prevent dexpreopting.  Defaults to true.
		Enabled *bool

		// If true, generate an app image (.art file) for this module.
		App_image *bool

		// If true, use a checked-in profile to guide optimization.  Defaults to false unless
		// a matching profile is set or a profile is found in PRODUCT_DEX_PREOPT_PROFILE_DIR
		// that matches the name of this module, in which case it is defaulted to true.
		Profile_guided *bool

		// If set, provides the path to profile relative to the Android.bp file.  If not set,
		// defaults to searching for a file that matches the name of this module in the default
		// profile location set by PRODUCT_DEX_PREOPT_PROFILE_DIR, or empty if not found.
		Profile *string `android:"path"`
	}
}

func init() {
	dexpreopt.DexpreoptRunningInSoong = true
}

func (d *dexpreopter) dexpreoptDisabled(ctx android.BaseModuleContext) bool {
	global := dexpreopt.GetGlobalConfig(ctx)

	if global.DisablePreopt {
		return true
	}

	if inList(ctx.ModuleName(), global.DisablePreoptModules) {
		return true
	}

	if d.isTest {
		return true
	}

	if !BoolDefault(d.dexpreoptProperties.Dex_preopt.Enabled, true) {
		return true
	}

	if !ctx.Module().(dexpreopterInterface).IsInstallable() {
		return true
	}

	if ctx.Host() {
		return true
	}

	// Don't preopt APEX variant module
	if apexInfo := ctx.Provider(android.ApexInfoProvider).(android.ApexInfo); !apexInfo.IsForPlatform() {
		return true
	}

	// TODO: contains no java code

	return false
}

func dexpreoptToolDepsMutator(ctx android.BottomUpMutatorContext) {
	if d, ok := ctx.Module().(dexpreopterInterface); !ok || d.dexpreoptDisabled(ctx) {
		return
	}
	dexpreopt.RegisterToolDeps(ctx)
}

func odexOnSystemOther(ctx android.ModuleContext, installPath android.InstallPath) bool {
	return dexpreopt.OdexOnSystemOtherByName(ctx.ModuleName(), android.InstallPathToOnDevicePath(ctx, installPath), dexpreopt.GetGlobalConfig(ctx))
}

func (d *dexpreopter) dexpreopt(ctx android.ModuleContext, dexJarFile android.WritablePath) {
	// TODO(b/148690468): The check on d.installPath is to bail out in cases where
	// the dexpreopter struct hasn't been fully initialized before we're called,
	// e.g. in aar.go. This keeps the behaviour that dexpreopting is effectively
	// disabled, even if installable is true.
	if d.installPath.Base() == "." {
		return
	}

	dexLocation := android.InstallPathToOnDevicePath(ctx, d.installPath)

	providesUsesLib := ctx.ModuleName()
	if ulib, ok := ctx.Module().(ProvidesUsesLib); ok {
		name := ulib.ProvidesUsesLib()
		if name != nil {
			providesUsesLib = *name
		}
	}

	// If it is test, make config files regardless of its dexpreopt setting.
	// The config files are required for apps defined in make which depend on the lib.
	if d.isTest && d.dexpreoptDisabled(ctx) {
		return
	}

	global := dexpreopt.GetGlobalConfig(ctx)

	isSystemServerJar := global.SystemServerJars.ContainsJar(ctx.ModuleName())

	bootImage := defaultBootImageConfig(ctx)
	if global.UseArtImage {
		bootImage = artBootImageConfig(ctx)
	}

	// System server jars are an exception: they are dexpreopted without updatable bootclasspath.
	dexFiles, dexLocations := bcpForDexpreopt(ctx, global.PreoptWithUpdatableBcp && !isSystemServerJar)

	targets := ctx.MultiTargets()
	if len(targets) == 0 {
		// assume this is a java library, dexpreopt for all arches for now
		for _, target := range ctx.Config().Targets[android.Android] {
			if target.NativeBridge == android.NativeBridgeDisabled {
				targets = append(targets, target)
			}
		}
		if isSystemServerJar && !d.isSDKLibrary {
			// If the module is not an SDK library and it's a system server jar, only preopt the primary arch.
			targets = targets[:1]
		}
	}

	var archs []android.ArchType
	var images android.Paths
	var imagesDeps []android.OutputPaths
	for _, target := range targets {
		archs = append(archs, target.Arch.ArchType)
		variant := bootImage.getVariant(target)
		images = append(images, variant.imagePathOnHost)
		imagesDeps = append(imagesDeps, variant.imagesDeps)
	}
	// The image locations for all Android variants are identical.
	hostImageLocations, deviceImageLocations := bootImage.getAnyAndroidVariant().imageLocations()

	var profileClassListing android.OptionalPath
	var profileBootListing android.OptionalPath
	profileIsTextListing := false
	if BoolDefault(d.dexpreoptProperties.Dex_preopt.Profile_guided, true) {
		// If dex_preopt.profile_guided is not set, default it based on the existence of the
		// dexprepot.profile option or the profile class listing.
		if String(d.dexpreoptProperties.Dex_preopt.Profile) != "" {
			profileClassListing = android.OptionalPathForPath(
				android.PathForModuleSrc(ctx, String(d.dexpreoptProperties.Dex_preopt.Profile)))
			profileBootListing = android.ExistentPathForSource(ctx,
				ctx.ModuleDir(), String(d.dexpreoptProperties.Dex_preopt.Profile)+"-boot")
			profileIsTextListing = true
		} else if global.ProfileDir != "" {
			profileClassListing = android.ExistentPathForSource(ctx,
				global.ProfileDir, ctx.ModuleName()+".prof")
		}
	}

	// Full dexpreopt config, used to create dexpreopt build rules.
	dexpreoptConfig := &dexpreopt.ModuleConfig{
		Name:            ctx.ModuleName(),
		DexLocation:     dexLocation,
		BuildPath:       android.PathForModuleOut(ctx, "dexpreopt", ctx.ModuleName()+".jar").OutputPath,
		DexPath:         dexJarFile,
		ManifestPath:    android.OptionalPathForPath(d.manifestFile),
		UncompressedDex: d.uncompressedDex,
		HasApkLibraries: false,
		PreoptFlags:     nil,

		ProfileClassListing:  profileClassListing,
		ProfileIsTextListing: profileIsTextListing,
		ProfileBootListing:   profileBootListing,

		EnforceUsesLibrariesStatusFile: dexpreopt.UsesLibrariesStatusFile(ctx),
		EnforceUsesLibraries:           d.enforceUsesLibs,
		ProvidesUsesLibrary:            providesUsesLib,
		ClassLoaderContexts:            d.classLoaderContexts,

		Archs:                           archs,
		DexPreoptImagesDeps:             imagesDeps,
		DexPreoptImageLocationsOnHost:   hostImageLocations,
		DexPreoptImageLocationsOnDevice: deviceImageLocations,

		PreoptBootClassPathDexFiles:     dexFiles.Paths(),
		PreoptBootClassPathDexLocations: dexLocations,

		PreoptExtractedApk: false,

		NoCreateAppImage:    !BoolDefault(d.dexpreoptProperties.Dex_preopt.App_image, true),
		ForceCreateAppImage: BoolDefault(d.dexpreoptProperties.Dex_preopt.App_image, false),

		PresignedPrebuilt: d.isPresignedPrebuilt,
	}

	d.configPath = android.PathForModuleOut(ctx, "dexpreopt", "dexpreopt.config")
	dexpreopt.WriteModuleConfig(ctx, dexpreoptConfig, d.configPath)

	if d.dexpreoptDisabled(ctx) {
		return
	}

	globalSoong := dexpreopt.GetGlobalSoongConfig(ctx)

	dexpreoptRule, err := dexpreopt.GenerateDexpreoptRule(ctx, globalSoong, global, dexpreoptConfig)
	if err != nil {
		ctx.ModuleErrorf("error generating dexpreopt rule: %s", err.Error())
		return
	}

	dexpreoptRule.Build("dexpreopt", "dexpreopt")

	d.builtInstalled = dexpreoptRule.Installs().String()
}

// Provides build path and install path to DEX jars.
type UsesLibraryDependency interface {
	DexJarBuildPath() android.Path
	DexJarInstallPath() android.Path
	ClassLoaderContexts() dexpreopt.ClassLoaderContextMap
}

type UsesLibraryProperties struct {
	// A list of shared library modules that will be listed in uses-library tags in the AndroidManifest.xml file.
	Uses_libs []string

	// A list of shared library modules that will be listed in uses-library tags in the AndroidManifest.xml file with
	// required=false.
	Optional_uses_libs []string

	// If true, the list of uses_libs and optional_uses_libs modules must match the AndroidManifest.xml file.  Defaults
	// to true if either uses_libs or optional_uses_libs is set.  Will unconditionally default to true in the future.
	Enforce_uses_libs *bool

	// Optional name of the <uses-library> provided by this module. This is needed for non-SDK
	// libraries, because SDK ones are automatically picked up by Soong. The <uses-library> name
	// normally is the same as the module name, but there are exceptions.
	Provides_uses_lib *string
}

// usesLibrary provides properties and helper functions for AndroidApp and AndroidAppImport to verify that the
// <uses-library> tags that end up in the manifest of an APK match the ones known to the build system through the
// uses_libs and optional_uses_libs properties.  The build system's values are used by dexpreopt to preopt apps
// with knowledge of their shared libraries.
type usesLibrary struct {
	usesLibraryProperties UsesLibraryProperties

	// Whether to enforce verify_uses_library check.
	enforce bool
}

func (u *usesLibrary) addLib(lib string, optional bool) {
	if !android.InList(lib, u.usesLibraryProperties.Uses_libs) && !android.InList(lib, u.usesLibraryProperties.Optional_uses_libs) {
		if optional {
			u.usesLibraryProperties.Optional_uses_libs = append(u.usesLibraryProperties.Optional_uses_libs, lib)
		} else {
			u.usesLibraryProperties.Uses_libs = append(u.usesLibraryProperties.Uses_libs, lib)
		}
	}
}

func (u *usesLibrary) deps(ctx android.BottomUpMutatorContext, hasFrameworkLibs bool) {
	if !ctx.Config().UnbundledBuild() || ctx.Config().UnbundledBuildImage() {
		ctx.AddVariationDependencies(nil, usesLibReqTag, u.usesLibraryProperties.Uses_libs...)
		ctx.AddVariationDependencies(nil, usesLibOptTag, u.presentOptionalUsesLibs(ctx)...)
		// Only add these extra dependencies if the module depends on framework libs. This avoids
		// creating a cyclic dependency:
		//     e.g. framework-res -> org.apache.http.legacy -> ... -> framework-res.
		if hasFrameworkLibs {
			// Dexpreopt needs paths to the dex jars of these libraries in order to construct
			// class loader context for dex2oat. Add them as a dependency with a special tag.
			ctx.AddVariationDependencies(nil, usesLibCompat29ReqTag, dexpreopt.CompatUsesLibs29...)
			ctx.AddVariationDependencies(nil, usesLibCompat28OptTag, dexpreopt.OptionalCompatUsesLibs28...)
			ctx.AddVariationDependencies(nil, usesLibCompat30OptTag, dexpreopt.OptionalCompatUsesLibs30...)
		}
	}
}

// presentOptionalUsesLibs returns optional_uses_libs after filtering out MissingUsesLibraries, which don't exist in the
// build.
func (u *usesLibrary) presentOptionalUsesLibs(ctx android.BaseModuleContext) []string {
	optionalUsesLibs, _ := android.FilterList(u.usesLibraryProperties.Optional_uses_libs, ctx.Config().MissingUsesLibraries())
	return optionalUsesLibs
}

// Helper function to replace string in a list.
func replaceInList(list []string, oldstr, newstr string) {
	for i, str := range list {
		if str == oldstr {
			list[i] = newstr
		}
	}
}

// Returns a map of module names of shared library dependencies to the paths to their dex jars on
// host and on device.
func (u *usesLibrary) classLoaderContextForUsesLibDeps(ctx android.ModuleContext) dexpreopt.ClassLoaderContextMap {
	clcMap := make(dexpreopt.ClassLoaderContextMap)

	// Skip when UnbundledBuild() is true, but UnbundledBuildImage() is false. With
	// UnbundledBuildImage() it is necessary to generate dexpreopt.config for post-dexpreopting.
	if ctx.Config().UnbundledBuild() && !ctx.Config().UnbundledBuildImage() {
		return clcMap
	}

	ctx.VisitDirectDeps(func(m android.Module) {
		tag, isUsesLibTag := ctx.OtherModuleDependencyTag(m).(usesLibraryDependencyTag)
		if !isUsesLibTag {
			return
		}

		dep := android.RemoveOptionalPrebuiltPrefix(ctx.OtherModuleName(m))

		// Skip stub libraries. A dependency on the implementation library has been added earlier,
		// so it will be added to CLC, but the stub shouldn't be. Stub libraries can be distingushed
		// from implementation libraries by their name, which is different as it has a suffix.
		if comp, ok := m.(SdkLibraryComponentDependency); ok {
			if impl := comp.OptionalSdkLibraryImplementation(); impl != nil && *impl != dep {
				return
			}
		}

		if lib, ok := m.(UsesLibraryDependency); ok {
			libName := dep
			if ulib, ok := m.(ProvidesUsesLib); ok && ulib.ProvidesUsesLib() != nil {
				libName = *ulib.ProvidesUsesLib()
				// Replace module name with library name in `uses_libs`/`optional_uses_libs` in
				// order to pass verify_uses_libraries check (which compares these properties
				// against library names written in the manifest).
				replaceInList(u.usesLibraryProperties.Uses_libs, dep, libName)
				replaceInList(u.usesLibraryProperties.Optional_uses_libs, dep, libName)
			}
			clcMap.AddContext(ctx, tag.sdkVersion, libName, tag.optional,
				lib.DexJarBuildPath(), lib.DexJarInstallPath(), lib.ClassLoaderContexts())
		} else if ctx.Config().AllowMissingDependencies() {
			ctx.AddMissingDependencies([]string{dep})
		} else {
			ctx.ModuleErrorf("module %q in uses_libs or optional_uses_libs must be a java library", dep)
		}
	})
	return clcMap
}

// enforceUsesLibraries returns true of <uses-library> tags should be checked against uses_libs and optional_uses_libs
// properties.  Defaults to true if either of uses_libs or optional_uses_libs is specified.  Will default to true
// unconditionally in the future.
func (u *usesLibrary) enforceUsesLibraries() bool {
	defaultEnforceUsesLibs := len(u.usesLibraryProperties.Uses_libs) > 0 ||
		len(u.usesLibraryProperties.Optional_uses_libs) > 0
	return BoolDefault(u.usesLibraryProperties.Enforce_uses_libs, u.enforce || defaultEnforceUsesLibs)
}

// Freeze the value of `enforce_uses_libs` based on the current values of `uses_libs` and `optional_uses_libs`.
func (u *usesLibrary) freezeEnforceUsesLibraries() {
	enforce := u.enforceUsesLibraries()
	u.usesLibraryProperties.Enforce_uses_libs = &enforce
}

// verifyUsesLibraries checks the <uses-library> tags in the manifest against the ones specified
// in the `uses_libs`/`optional_uses_libs` properties. The input can be either an XML manifest, or
// an APK with the manifest embedded in it (manifest_check will know which one it is by the file
// extension: APKs are supposed to end with '.apk').
func (u *usesLibrary) verifyUsesLibraries(ctx android.ModuleContext, inputFile android.Path,
	outputFile android.WritablePath) android.Path {

	statusFile := dexpreopt.UsesLibrariesStatusFile(ctx)

	// Disable verify_uses_libraries check if dexpreopt is globally disabled. Without dexpreopt the
	// check is not necessary, and although it is good to have, it is difficult to maintain on
	// non-linux build platforms where dexpreopt is generally disabled (the check may fail due to
	// various unrelated reasons, such as a failure to get manifest from an APK).
	global := dexpreopt.GetGlobalConfig(ctx)
	if global.DisablePreopt || global.OnlyPreoptBootImageAndSystemServer {
		return inputFile
	}

	rule := android.NewRuleBuilder(pctx, ctx)
	cmd := rule.Command().BuiltTool("manifest_check").
		Flag("--enforce-uses-libraries").
		Input(inputFile).
		FlagWithOutput("--enforce-uses-libraries-status ", statusFile).
		FlagWithInput("--aapt ", ctx.Config().HostToolPath(ctx, "aapt"))

	if outputFile != nil {
		cmd.FlagWithOutput("-o ", outputFile)
	}

	if dexpreopt.GetGlobalConfig(ctx).RelaxUsesLibraryCheck {
		cmd.Flag("--enforce-uses-libraries-relax")
	}

	for _, lib := range u.usesLibraryProperties.Uses_libs {
		cmd.FlagWithArg("--uses-library ", lib)
	}

	for _, lib := range u.usesLibraryProperties.Optional_uses_libs {
		cmd.FlagWithArg("--optional-uses-library ", lib)
	}

	rule.Build("verify_uses_libraries", "verify <uses-library>")
	return outputFile
}

// verifyUsesLibrariesManifest checks the <uses-library> tags in an AndroidManifest.xml against
// the build system and returns the path to a copy of the manifest.
func (u *usesLibrary) verifyUsesLibrariesManifest(ctx android.ModuleContext, manifest android.Path) android.Path {
	outputFile := android.PathForModuleOut(ctx, "manifest_check", "AndroidManifest.xml")
	return u.verifyUsesLibraries(ctx, manifest, outputFile)
}

// verifyUsesLibrariesAPK checks the <uses-library> tags in the manifest of an APK against the build
// system and returns the path to a copy of the APK.
func (u *usesLibrary) verifyUsesLibrariesAPK(ctx android.ModuleContext, apk android.Path) android.Path {
	u.verifyUsesLibraries(ctx, apk, nil) // for APKs manifest_check does not write output file
	outputFile := android.PathForModuleOut(ctx, "verify_uses_libraries", apk.Base())
	return outputFile
}
