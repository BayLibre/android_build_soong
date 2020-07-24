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
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"android/soong/android"
	"android/soong/java/config"

	"github.com/google/blueprint/pathtools"
)

func init() {
	android.RegisterPreSingletonType("sdk_versions", sdkPreSingletonFactory)
	android.RegisterSingletonType("sdk", sdkSingletonFactory)
	android.RegisterMakeVarsProvider(pctx, sdkMakeVars)
}

var sdkVersionsKey = android.NewOnceKey("sdkVersionsKey")
var sdkFrameworkAidlPathKey = android.NewOnceKey("sdkFrameworkAidlPathKey")
var nonUpdatableFrameworkAidlPathKey = android.NewOnceKey("nonUpdatableFrameworkAidlPathKey")
var apiFingerprintPathKey = android.NewOnceKey("apiFingerprintPathKey")

type sdkContext interface {
	// sdkVersion returns sdkSpec that corresponds to the sdk_version property of the current module
	sdkVersion(android.EarlyModuleContext) sdkSpec
	// systemModules returns the system_modules property of the current module, or an empty string if it is not set.
	systemModules() string
	// minSdkVersion returns sdkSpec that corresponds to the min_sdk_version property of the current module,
	// or from sdk_version if it is not set.
	minSdkVersion(android.EarlyModuleContext) sdkSpec
	// targetSdkVersion returns the sdkSpec that corresponds to the target_sdk_version property of the current module,
	// or from sdk_version if it is not set.
	targetSdkVersion(android.EarlyModuleContext) sdkSpec
}

func UseApiFingerprint(ctx android.BaseModuleContext) bool {
	if ctx.Config().UnbundledBuild() &&
		!ctx.Config().AlwaysUsePrebuiltSdks() &&
		ctx.Config().IsEnvTrue("UNBUNDLED_BUILD_TARGET_SDK_WITH_API_FINGERPRINT") {
		return true
	}
	return false
}

// sdkKind represents a particular category of an SDK spec like public, system, test, etc.
type sdkKind int

const (
	sdkInvalid sdkKind = iota
	sdkNone
	sdkCore
	sdkCorePlatform
	sdkPublic
	sdkSystem
	sdkTest
	sdkModule
	sdkSystemServer
	sdkPrivate
)

// String returns the string representation of this sdkKind
func (k sdkKind) String() string {
	switch k {
	case sdkPrivate:
		return "private"
	case sdkNone:
		return "none"
	case sdkPublic:
		return "public"
	case sdkSystem:
		return "system"
	case sdkTest:
		return "test"
	case sdkCore:
		return "core"
	case sdkCorePlatform:
		return "core_platform"
	case sdkModule:
		return "module-lib"
	case sdkSystemServer:
		return "system-server"
	default:
		return "invalid"
	}
}

// sdkSpec represents the kind and the version of an SDK for a module to build against
type sdkSpec struct {
	kind    sdkKind
	version android.ApiLevel
	raw     string
}

func (s sdkSpec) String() string {
	return fmt.Sprintf("%s_%s", s.kind, s.version)
}

// valid checks if this sdkSpec is well-formed. Note however that true doesn't mean that the
// specified SDK actually exists.
func (s sdkSpec) valid() bool {
	return s.kind != sdkInvalid
}

// specified checks if this sdkSpec is well-formed and is not "".
func (s sdkSpec) specified() bool {
	return s.valid() && s.kind != sdkPrivate
}

// whether the API surface is managed and versioned, i.e. has .txt file that
// get frozen on SDK freeze and changes get reviewed by API council.
func (s sdkSpec) stable() bool {
	if !s.specified() {
		return false
	}
	switch s.kind {
	case sdkNone:
		// there is nothing to manage and version in this case; de facto stable API.
		return true
	case sdkCore, sdkPublic, sdkSystem, sdkModule, sdkSystemServer:
		return true
	case sdkCorePlatform, sdkTest, sdkPrivate:
		return false
	default:
		panic(fmt.Errorf("unknown sdkKind=%v", s.kind))
	}
	return false
}

// prebuiltSdkAvailableForUnbundledBuilt tells whether this sdkSpec can have a prebuilt SDK
// that can be used for unbundled builds.
func (s sdkSpec) prebuiltSdkAvailableForUnbundledBuild() bool {
	// "", "none", and "core_platform" are not available for unbundled build
	// as we don't/can't have prebuilt stub for the versions
	return s.kind != sdkPrivate && s.kind != sdkNone && s.kind != sdkCorePlatform
}

func (s sdkSpec) forVendorPartition(ctx android.EarlyModuleContext) sdkSpec {
	// If BOARD_CURRENT_API_LEVEL_FOR_VENDOR_MODULES has a numeric value,
	// use it instead of "current" for the vendor partition.
	currentSdkVersion := ctx.DeviceConfig().CurrentApiLevelForVendorModules(ctx)
	if currentSdkVersion.IsCurrent() {
		return s
	}

	if s.kind == sdkPublic || s.kind == sdkSystem {
		if s.version.IsCurrent() {
			return sdkSpec{s.kind, currentSdkVersion, s.raw}
		}
	}
	return s
}

// usePrebuilt determines whether prebuilt SDK should be used for this sdkSpec with the given context.
func (s sdkSpec) usePrebuilt(ctx android.EarlyModuleContext) bool {
	if s.version.IsCurrent() {
		// "current" can be built from source and be from prebuilt SDK
		return ctx.Config().AlwaysUsePrebuiltSdks()
	} else if !s.version.IsPreview() {
		if s.kind != sdkPublic && s.kind != sdkSystem && s.kind != sdkTest {
			panic(fmt.Errorf("prebuilt SDK is not not available for sdkKind=%q", s.kind))
			return false
		}
		// numbered SDKs are always from prebuilt
		return true
	}
	// "", "none", "core_platform" fall here
	return false
}

// Returns the version that the module should actually target. All preview API
// levels are coerced to the default targetSdkVersion for the build.
func (s sdkSpec) effectiveVersion(ctx android.EarlyModuleContext) (android.ApiLevel, error) {
	if !s.valid() {
		return s.version, fmt.Errorf("invalid sdk version %q", s.raw)
	}

	if ctx.DeviceSpecific() || ctx.SocSpecific() {
		s = s.forVendorPartition(ctx)
	}
	if !s.version.IsPreview() {
		return s.version, nil
	}
	return ctx.Config().DefaultAppTargetSdk(ctx), nil
}

func (s sdkSpec) defaultJavaLanguageVersion(ctx android.EarlyModuleContext) javaVersion {
	sdk, err := s.effectiveVersion(ctx)
	if err != nil {
		ctx.PropertyErrorf("sdk_version", "%s", err)
	}
	if sdk.GreaterThanOrEqualTo(android.FirstJava9Version) {
		return JAVA_VERSION_9
	} else if sdk.GreaterThanOrEqualTo(android.FirstJava8Version) {
		return JAVA_VERSION_8
	} else {
		return JAVA_VERSION_7
	}
}

func sdkSpecFrom(ctx android.EarlyModuleContext, str string) sdkSpec {
	switch str {
	// special cases first
	case "":
		return sdkSpec{sdkPrivate, android.NoneApiLevel, str}
	case "none":
		return sdkSpec{sdkNone, android.NoneApiLevel, str}
	case "core_platform":
		return sdkSpec{sdkCorePlatform, android.NoneApiLevel, str}
	default:
		// the syntax is [kind_]version
		sep := strings.LastIndex(str, "_")

		var kindString string
		if sep == 0 {
			return sdkSpec{sdkInvalid, android.NoneApiLevel, str}
		} else if sep == -1 {
			kindString = ""
		} else {
			kindString = str[0:sep]
		}
		versionString := str[sep+1 : len(str)]

		var kind sdkKind
		switch kindString {
		case "":
			kind = sdkPublic
		case "core":
			kind = sdkCore
		case "system":
			kind = sdkSystem
		case "test":
			kind = sdkTest
		case "module":
			kind = sdkModule
		case "system_server":
			kind = sdkSystemServer
		default:
			return sdkSpec{sdkInvalid, android.NoneApiLevel, str}
		}

		version, err := android.ApiLevelFromUser(ctx, versionString)
		if err != nil {
			return sdkSpec{sdkInvalid, android.NoneApiLevel, str}
		}

		if version.IsPreview() && !version.IsCurrent() {
			// Java contexts do not allow specifying codenames. Only final API
			// numbers and "current" are allowed.
			return sdkSpec{sdkInvalid, android.NoneApiLevel, str}
		}

		return sdkSpec{kind, *version, str}
	}
}

func (s sdkSpec) validateSystemSdk(ctx android.EarlyModuleContext) bool {
	// Ensures that the specified system SDK version is one of BOARD_SYSTEMSDK_VERSIONS (for vendor/product Java module)
	// Assuming that BOARD_SYSTEMSDK_VERSIONS := 28 29,
	// sdk_version of the modules in vendor/product that use system sdk must be either system_28, system_29 or system_current
	if s.kind != sdkSystem || s.version.IsPreview() {
		return true
	}
	allowedVersions := ctx.DeviceConfig().PlatformSystemSdkVersions()
	if ctx.DeviceSpecific() || ctx.SocSpecific() || (ctx.ProductSpecific() && ctx.Config().EnforceProductPartitionInterface()) {
		systemSdkVersions := ctx.DeviceConfig().SystemSdkVersions()
		if len(systemSdkVersions) > 0 {
			allowedVersions = systemSdkVersions
		}
	}
	if len(allowedVersions) > 0 && !android.InList(s.version.String(), allowedVersions) {
		ctx.PropertyErrorf("sdk_version", "incompatible sdk version %q. System SDK version should be one of %q",
			s.raw, allowedVersions)
		return false
	}
	return true
}

func decodeSdkDep(ctx android.EarlyModuleContext, sdkContext sdkContext) sdkDep {
	sdkVersion := sdkContext.sdkVersion(ctx)
	if !sdkVersion.valid() {
		ctx.PropertyErrorf("sdk_version", "invalid version %q", sdkVersion.raw)
		return sdkDep{}
	}

	if ctx.DeviceSpecific() || ctx.SocSpecific() {
		sdkVersion = sdkVersion.forVendorPartition(ctx)
	}

	if !sdkVersion.validateSystemSdk(ctx) {
		return sdkDep{}
	}

	if sdkVersion.usePrebuilt(ctx) {
		dir := filepath.Join("prebuilts", "sdk", sdkVersion.version.String(), sdkVersion.kind.String())
		jar := filepath.Join(dir, "android.jar")
		// There's no aidl for other SDKs yet.
		// TODO(77525052): Add aidl files for other SDKs too.
		public_dir := filepath.Join("prebuilts", "sdk", sdkVersion.version.String(), "public")
		aidl := filepath.Join(public_dir, "framework.aidl")
		jarPath := android.ExistentPathForSource(ctx, jar)
		aidlPath := android.ExistentPathForSource(ctx, aidl)
		lambdaStubsPath := android.PathForSource(ctx, config.SdkLambdaStubsPath)

		if (!jarPath.Valid() || !aidlPath.Valid()) && ctx.Config().AllowMissingDependencies() {
			return sdkDep{
				invalidVersion: true,
				bootclasspath:  []string{fmt.Sprintf("sdk_%s_%s_android", sdkVersion.kind, sdkVersion.version.String())},
			}
		}

		if !jarPath.Valid() {
			ctx.PropertyErrorf("sdk_version", "invalid sdk version %q, %q does not exist", sdkVersion.raw, jar)
			return sdkDep{}
		}

		if !aidlPath.Valid() {
			ctx.PropertyErrorf("sdk_version", "invalid sdk version %q, %q does not exist", sdkVersion.raw, aidl)
			return sdkDep{}
		}

		var systemModules string
		if sdkVersion.defaultJavaLanguageVersion(ctx).usesJavaModules() {
			systemModules = "sdk_public_" + sdkVersion.version.String() + "_system_modules"
		}

		return sdkDep{
			useFiles:      true,
			jars:          android.Paths{jarPath.Path(), lambdaStubsPath},
			aidl:          android.OptionalPathForPath(aidlPath.Path()),
			systemModules: systemModules,
		}
	}

	toModule := func(modules []string, res string, aidl android.Path) sdkDep {
		return sdkDep{
			useModule:          true,
			bootclasspath:      append(modules, config.DefaultLambdaStubsLibrary),
			systemModules:      "core-current-stubs-system-modules",
			java9Classpath:     modules,
			frameworkResModule: res,
			aidl:               android.OptionalPathForPath(aidl),
		}
	}

	switch sdkVersion.kind {
	case sdkPrivate:
		return sdkDep{
			useModule:          true,
			systemModules:      corePlatformSystemModules(ctx),
			bootclasspath:      corePlatformBootclasspathLibraries(ctx),
			classpath:          config.FrameworkLibraries,
			frameworkResModule: "framework-res",
		}
	case sdkNone:
		systemModules := sdkContext.systemModules()
		if systemModules == "" {
			ctx.PropertyErrorf("sdk_version",
				`system_modules is required to be set to a non-empty value when sdk_version is "none", did you mean sdk_version: "core_platform"?`)
		} else if systemModules == "none" {
			return sdkDep{
				noStandardLibs: true,
			}
		}

		return sdkDep{
			useModule:      true,
			noStandardLibs: true,
			systemModules:  systemModules,
			bootclasspath:  []string{systemModules},
		}
	case sdkCorePlatform:
		return sdkDep{
			useModule:        true,
			systemModules:    corePlatformSystemModules(ctx),
			bootclasspath:    corePlatformBootclasspathLibraries(ctx),
			noFrameworksLibs: true,
		}
	case sdkPublic:
		return toModule([]string{"android_stubs_current"}, "framework-res", sdkFrameworkAidlPath(ctx))
	case sdkSystem:
		return toModule([]string{"android_system_stubs_current"}, "framework-res", sdkFrameworkAidlPath(ctx))
	case sdkTest:
		return toModule([]string{"android_test_stubs_current"}, "framework-res", sdkFrameworkAidlPath(ctx))
	case sdkCore:
		return sdkDep{
			useModule:        true,
			bootclasspath:    []string{"core.current.stubs", config.DefaultLambdaStubsLibrary},
			systemModules:    "core-current-stubs-system-modules",
			noFrameworksLibs: true,
		}
	case sdkModule:
		// TODO(146757305): provide .apk and .aidl that have more APIs for modules
		return toModule([]string{"android_module_lib_stubs_current"}, "framework-res", nonUpdatableFrameworkAidlPath(ctx))
	case sdkSystemServer:
		// TODO(146757305): provide .apk and .aidl that have more APIs for modules
		return toModule([]string{"android_system_server_stubs_current"}, "framework-res", sdkFrameworkAidlPath(ctx))
	default:
		panic(fmt.Errorf("invalid sdk %q", sdkVersion.raw))
	}
}

func sdkPreSingletonFactory() android.Singleton {
	return sdkPreSingleton{}
}

type sdkPreSingleton struct{}

func (sdkPreSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	sdkJars, err := ctx.GlobWithDeps("prebuilts/sdk/*/public/android.jar", nil)
	if err != nil {
		ctx.Errorf("failed to glob prebuilts/sdk/*/public/android.jar: %s", err.Error())
	}

	var sdkVersions []int
	for _, sdkJar := range sdkJars {
		dir := filepath.Base(filepath.Dir(filepath.Dir(sdkJar)))
		v, err := strconv.Atoi(dir)
		if scerr, ok := err.(*strconv.NumError); ok && scerr.Err == strconv.ErrSyntax {
			continue
		} else if err != nil {
			ctx.Errorf("invalid sdk jar %q, %s, %v", sdkJar, err.Error())
		}
		sdkVersions = append(sdkVersions, v)
	}

	sort.Ints(sdkVersions)
	versions := []android.ApiLevel{}
	for _, v := range sdkVersions {
		apiLevel, err := android.ApiLevelFromUser(ctx, strconv.Itoa(v))
		if err != nil {
			ctx.Errorf("Unable to identify API level for prebuilts/sdk/%d: %s",
				v, err)
			continue
		}
		versions = append(versions, *apiLevel)
	}

	ctx.Config().Once(sdkVersionsKey, func() interface{} { return versions })
}

func LatestSdkVersion(ctx android.EarlyModuleContext) android.ApiLevel {
	sdkVersions := ctx.Config().Get(sdkVersionsKey).([]android.ApiLevel)
	if len(sdkVersions) == 0 {
		panic("no SDK versions")
	}
	return sdkVersions[len(sdkVersions)-1]
}

func sdkSingletonFactory() android.Singleton {
	return sdkSingleton{}
}

type sdkSingleton struct{}

func (sdkSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if ctx.Config().AlwaysUsePrebuiltSdks() {
		return
	}

	createSdkFrameworkAidl(ctx)
	createNonUpdatableFrameworkAidl(ctx)
	createAPIFingerprint(ctx)
}

// Create framework.aidl by extracting anything that implements android.os.Parcelable from the SDK stubs modules.
func createSdkFrameworkAidl(ctx android.SingletonContext) {
	stubsModules := []string{
		"android_stubs_current",
		"android_test_stubs_current",
		"android_system_stubs_current",
	}

	combinedAidl := sdkFrameworkAidlPath(ctx)
	tempPath := combinedAidl.ReplaceExtension(ctx, "aidl.tmp")

	rule := createFrameworkAidl(stubsModules, tempPath, ctx)

	commitChangeForRestat(rule, tempPath, combinedAidl)

	rule.Build(pctx, ctx, "framework_aidl", "generate framework.aidl")
}

// Creates a version of framework.aidl for the non-updatable part of the platform.
func createNonUpdatableFrameworkAidl(ctx android.SingletonContext) {
	stubsModules := []string{"android_module_lib_stubs_current"}

	combinedAidl := nonUpdatableFrameworkAidlPath(ctx)
	tempPath := combinedAidl.ReplaceExtension(ctx, "aidl.tmp")

	rule := createFrameworkAidl(stubsModules, tempPath, ctx)

	commitChangeForRestat(rule, tempPath, combinedAidl)

	rule.Build(pctx, ctx, "framework_non_updatable_aidl", "generate framework_non_updatable.aidl")
}

func createFrameworkAidl(stubsModules []string, path android.OutputPath, ctx android.SingletonContext) *android.RuleBuilder {
	stubsJars := make([]android.Paths, len(stubsModules))

	ctx.VisitAllModules(func(module android.Module) {
		// Collect dex jar paths for the modules listed above.
		if j, ok := module.(Dependency); ok {
			name := ctx.ModuleName(module)
			if i := android.IndexList(name, stubsModules); i != -1 {
				stubsJars[i] = j.HeaderJars()
			}
		}
	})

	var missingDeps []string

	for i := range stubsJars {
		if stubsJars[i] == nil {
			if ctx.Config().AllowMissingDependencies() {
				missingDeps = append(missingDeps, stubsModules[i])
			} else {
				ctx.Errorf("failed to find dex jar path for module %q", stubsModules[i])
			}
		}
	}

	rule := android.NewRuleBuilder()
	rule.MissingDeps(missingDeps)

	var aidls android.Paths
	for _, jars := range stubsJars {
		for _, jar := range jars {
			aidl := android.PathForOutput(ctx, "aidl", pathtools.ReplaceExtension(jar.Base(), "aidl"))

			rule.Command().
				Text("rm -f").Output(aidl)
			rule.Command().
				BuiltTool(ctx, "sdkparcelables").
				Input(jar).
				Output(aidl)

			aidls = append(aidls, aidl)
		}
	}

	rule.Command().
		Text("rm -f").Output(path)
	rule.Command().
		Text("cat").
		Inputs(aidls).
		Text("| sort -u >").
		Output(path)

	return rule
}

func sdkFrameworkAidlPath(ctx android.PathContext) android.OutputPath {
	return ctx.Config().Once(sdkFrameworkAidlPathKey, func() interface{} {
		return android.PathForOutput(ctx, "framework.aidl")
	}).(android.OutputPath)
}

func nonUpdatableFrameworkAidlPath(ctx android.PathContext) android.OutputPath {
	return ctx.Config().Once(nonUpdatableFrameworkAidlPathKey, func() interface{} {
		return android.PathForOutput(ctx, "framework_non_updatable.aidl")
	}).(android.OutputPath)
}

// Create api_fingerprint.txt
func createAPIFingerprint(ctx android.SingletonContext) {
	out := ApiFingerprintPath(ctx)

	rule := android.NewRuleBuilder()

	rule.Command().
		Text("rm -f").Output(out)
	cmd := rule.Command()

	if ctx.Config().PlatformSdkCodename() == "REL" {
		cmd.Text("echo REL >").Output(out)
	} else if !ctx.Config().AlwaysUsePrebuiltSdks() {
		in, err := ctx.GlobWithDeps("frameworks/base/api/*current.txt", nil)
		if err != nil {
			ctx.Errorf("error globbing API files: %s", err)
		}

		cmd.Text("cat").
			Inputs(android.PathsForSource(ctx, in)).
			Text("| md5sum | cut -d' ' -f1 >").
			Output(out)
	} else {
		// Unbundled build
		// TODO: use a prebuilt api_fingerprint.txt from prebuilts/sdk/current.txt once we have one
		cmd.Text("echo").
			Flag(ctx.Config().PlatformPreviewSdkVersion()).
			Text(">").
			Output(out)
	}

	rule.Build(pctx, ctx, "api_fingerprint", "generate api_fingerprint.txt")
}

func ApiFingerprintPath(ctx android.PathContext) android.OutputPath {
	return ctx.Config().Once(apiFingerprintPathKey, func() interface{} {
		return android.PathForOutput(ctx, "api_fingerprint.txt")
	}).(android.OutputPath)
}

func sdkMakeVars(ctx android.MakeVarsContext) {
	if ctx.Config().AlwaysUsePrebuiltSdks() {
		return
	}

	ctx.Strict("FRAMEWORK_AIDL", sdkFrameworkAidlPath(ctx).String())
	ctx.Strict("API_FINGERPRINT", ApiFingerprintPath(ctx).String())
}
