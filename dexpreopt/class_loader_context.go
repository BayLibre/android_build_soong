// Copyright 2020 Google Inc. All rights reserved.
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

package dexpreopt

import (
	"fmt"
	"strconv"
	"strings"

	"android/soong/android"
)

// Class loader context (CLC) is a tree of libraries that represent transitive closure of all
// <uses-library> dependencies of the dexpreopted module (library or app). The toplevel elements of
// a CLC tree are the libraries that are used directly by the dexpreopted module (they are specified
// as <uses-library> tags in its manifest). Each node of a CLC tree is a <uses-library> which may
// have its own <uses-library> sub-nodes.
//
// Because <uses-library> dependencies are, in general, a graph and not necessarily a tree, CLC may
// contain subtrees for the same library multiple times. In other words, CLC is the dependency graph
// "unfolded" to a tree.
//
// Example: A has <uses-library> tags B, C and D; C has <uses-library tags> B and D;
//          D has <uses-library> E; B and E have no <uses-library> dependencies. The CLC is:
//    A
//    ├── B
//    ├── C
//    │   ├── B
//    │   └── D
//    │       └── E
//    └── D
//        └── E
//
// Sometimes <uses-library> tags are missing from the source manifest of the library or app. This
// may happen if one of the transitive dependencies of the library/app starts using another
// <uses-library>, and the library/app's manifest isn't updated to include it. In such cases Soong
// adds the missing <uses-library> tags automatically. It computes the transitive closure of SDK
// library dependencies, stopping at shared library boundaries.
//
// CLC defines the lookup order of libraries when resolving Java classes used by the library/app.
// The lookup order is important because libraries may contain duplicate classes, and the class is
// resolved to the first match.
//
// For dexpreopt, CLC is important because the build-time CLC must be identical to the run-time CLC.
// If the run-time CLC is different, compiled code should be rejected. In order to check the
// equality of build-time and run-time CLC, the dex2oat compiler records build-time CLC in the
// *.odex files (in the "classpath" field of the OAT file header). To find the stored CLC, use the
// following command: `oatdump --oat-file=<FILE> | grep '^classpath = '`.
//
// <uses-library> can be either "required" or "optional" (this is specified as a tag attribute in
// the manifest). Required libraries are always used by the dexpreopted library/app at runtime.
// Optional libraries are used only if they are present at runtime (if not, the library/app can run
// without them). From the dexpreopt standpoint, if an optional <uses-library> is present at build
// time, it will be added to CLC and passed to dex2oat, otherwise it will be skipped and there will
// be no build-time error (CLC will be constructed without the missing library). If there is a
// mismatch between built-time and run-time (library is present in one case, but not the other),
// then the CLCs won't match and the compiled code will be rejected at runtime (this is impossible
// to guess at build-time). A required <uses-library> must be present at build time, its absence is
// a build error.
//
// In order to construct CLC for dex2oat and manifest_fixer, build system needs to know all
// <uses-library> dependencies of the dexpreopted library/app (including transitive ones). For each
// dependency it needs to know the following information:
//
//   - the real name of the <uses-library> (may be different from the module name)
//   - build-time (on host) and run-time (on device) paths to the DEX jar file of the library
//   - whether this library is optional or required
//   - whether this library is a shared library
//   - all <uses-library> dependencies
//
// Since the build system doesn't have access to the manifest contents (it cannot read manifests at
// the time of build rule generation), it is necessary to copy this information to the Android.bp
// and Android.mk files. For blueprints, the relevant properties are `uses_libs` and
// `options_uses_libs`. For makefiles, relevant variables are `LOCAL_USES_LIBRARIES` and
// `LOCAL_OPTIONAL_USES_LIBRARIES`. It is preferable to avoid specifying these properties explicilty
// when they can be computed automatically by Soong (as the transitive closure of SDK library
// dependencies).
//
// Some of the Java libraries that are used as <uses-library> are not SDK libraries. In order for
// the build system to handle them automatically like SDK libraries, it is possible to set a
// property `provides_uses_lib` or variable `LOCAL_PROVIDES_USES_LIBRARY` on the blueprint/makefile
// module of such library. This property can also be used to specify real library name is cases
// when it differs from the module name.
//
// Because the information from the manifests has to be duplicated in the build files, there is a
// danger that it may get out of sync. To guard against that, build system generates a rule that
// checks the metadata in the build files agains the contents of a manifest (verify_uses_libraries).
// The manifest can be available as a source file, or as part of a prebuilt APK. Note that reading
// the manifests at the Ninja stage of the build is fine, unlike the build rule generation phase.
//
// ClassLoaderContext is a structure that represents CLC.
//
type ClassLoaderContext struct {
	// The name of the library.
	Name string

	// On-host build path to the library dex file (used in dex2oat argument --class-loader-context).
	Host android.Path

	// On-device install path (used in dex2oat argument --stored-class-loader-context).
	Device string

	// Nested sub-CLC for dependencies.
	Subcontexts []*ClassLoaderContext

	// If this is a shared library (affects if its dependencies are passed to the manifest_fixer).
	IsSharedLibrary bool
}

// Conditional CLC is for compatibility libraries which didn't exist prior to a certain SDK version
// (say, N), but classes in them were in the bootclasspath jars, etc., and in version N they have
// been separated into a standalone <uses-library>. Compatibility libraries should only be in the
// CLC if the library/app that uses them has `targetSdkVersion` less than N in the manifest.
//
// Currently only apps (but not libraries) use conditional CLC.
//
// Target SDK version information is unavailable to the build system at rule generation time, so
// the build system doesn't know whether conditional CLC is needed for a given app or not. So it
// generates a build rule that includes conditional CLC for all versions, extracts the target SDK
// version from the manifest, and filters the CLCs based on that version. Exact final CLC that is
// passed to dex2oat is unknown to the build system, and gets known only at Ninja stage.
//
// ClassLoaderContextMap is a map from SDK version to CLC. There is a special entry with key
// AnySdkVersion that stores unconditional CLC that is added regardless of the target SDK version.
//
type ClassLoaderContextMap map[int][]*ClassLoaderContext

// Compatibility libraries. Some are optional, and some are required: this is the default that
// affects how they are handled by the Soong logic that automatically adds implicit SDK libraries
// to the manifest_fixer, but an explicit `uses_libs`/`optional_uses_libs` can override this.
var OrgApacheHttpLegacy = "org.apache.http.legacy"
var AndroidTestBase = "android.test.base"
var AndroidTestMock = "android.test.mock"
var AndroidHidlBase = "android.hidl.base-V1.0-java"
var AndroidHidlManager = "android.hidl.manager-V1.0-java"

var OptionalCompatUsesLibs28 = []string{
	OrgApacheHttpLegacy,
}
var OptionalCompatUsesLibs30 = []string{
	AndroidTestBase,
	AndroidTestMock,
}
var CompatUsesLibs29 = []string{
	AndroidHidlBase,
	AndroidHidlManager,
}
var OptionalCompatUsesLibs = append(android.CopyOf(OptionalCompatUsesLibs28), OptionalCompatUsesLibs30...)
var CompatUsesLibs = android.CopyOf(CompatUsesLibs29)

const UnknownInstallLibraryPath = "error"

// AnySdkVersion means that the class loader context is needed regardless of the targetSdkVersion
// of the app. The numeric value affects the key order in the map and, as a result, the order of
// arguments passed to construct_context.py (high value means that the unconditional context goes
// last). We use the converntional "current" SDK level (10000), but any big number would do as well.
const AnySdkVersion int = android.FutureApiLevelInt

// Add class loader context for the given library to the map entry for the given SDK version.
func (clcMap ClassLoaderContextMap) addContext(ctx android.ModuleInstallPathContext, sdkVer int, lib string,
	shared bool, hostPath, installPath android.Path, strict bool, nestedClcMap ClassLoaderContextMap) error {

	// If missing dependencies are allowed, the build shouldn't fail when a <uses-library> is
	// not found. However, this is likely to result is disabling dexpreopt, as it won't be
	// possible to construct class loader context without on-host and on-device library paths.
	strict = strict && !ctx.Config().AllowMissingDependencies()

	if hostPath == nil && strict {
		return fmt.Errorf("unknown build path to <uses-library> \"%s\"", lib)
	}

	devicePath := UnknownInstallLibraryPath
	if installPath == nil {
		if android.InList(lib, CompatUsesLibs) || android.InList(lib, OptionalCompatUsesLibs) {
			// Assume that compatibility libraries are installed in /system/framework.
			installPath = android.PathForModuleInstall(ctx, "framework", lib+".jar")
		} else if strict {
			return fmt.Errorf("unknown install path to <uses-library> \"%s\"", lib)
		} else {
			// For some stub libraries the only known thing is the name of their implementation
			// library, but the library itself is unavailable (missing or part of a prebuilt). In
			// such cases we still need to add the library to <uses-library> tags in the manifest,
			// but we cannot use it for dexpreopt.
		}
	}
	if installPath != nil {
		devicePath = android.InstallPathToOnDevicePath(ctx, installPath.(android.InstallPath))
	}

	// Nested class loader context shouldn't have conditional part (it is allowed only at the top level).
	for ver, _ := range nestedClcMap {
		if ver != AnySdkVersion {
			clcStr, _ := ComputeClassLoaderContext(nestedClcMap)
			return fmt.Errorf("nested class loader context shouldn't have conditional part: %s", clcStr)
		}
	}
	subcontexts := nestedClcMap[AnySdkVersion]

	// If the library with this name is already present as one of the unconditional top-level
	// components, do not re-add it.
	for _, clc := range clcMap[sdkVer] {
		if clc.Name == lib {
			return nil
		}
	}

	clcMap[sdkVer] = append(clcMap[sdkVer], &ClassLoaderContext{
		Name:            lib,
		Host:            hostPath,
		Device:          devicePath,
		Subcontexts:     subcontexts,
		IsSharedLibrary: shared,
	})
	return nil
}

// Wrapper around addContext that reports errors.
func (clcMap ClassLoaderContextMap) addContextOrReportError(ctx android.ModuleInstallPathContext, sdkVer int, lib string,
	shared bool, hostPath, installPath android.Path, strict bool, nestedClcMap ClassLoaderContextMap) {

	err := clcMap.addContext(ctx, sdkVer, lib, shared, hostPath, installPath, strict, nestedClcMap)
	if err != nil {
		ctx.ModuleErrorf(err.Error())
	}
}

// Add class loader context. Fail on unknown build/install paths.
func (clcMap ClassLoaderContextMap) AddContext(ctx android.ModuleInstallPathContext, lib string,
	shared bool, hostPath, installPath android.Path) {

	clcMap.addContextOrReportError(ctx, AnySdkVersion, lib, shared, hostPath, installPath, true, nil)
}

// Add class loader context if the library exists. Don't fail on unknown build/install paths.
func (clcMap ClassLoaderContextMap) MaybeAddContext(ctx android.ModuleInstallPathContext, lib *string,
	shared bool, hostPath, installPath android.Path) {

	if lib != nil {
		clcMap.addContextOrReportError(ctx, AnySdkVersion, *lib, shared, hostPath, installPath, false, nil)
	}
}

// Add class loader context for the given SDK version. Fail on unknown build/install paths.
func (clcMap ClassLoaderContextMap) AddContextForSdk(ctx android.ModuleInstallPathContext, sdkVer int,
	lib string, shared bool, hostPath, installPath android.Path, nestedClcMap ClassLoaderContextMap) {

	clcMap.addContextOrReportError(ctx, sdkVer, lib, shared, hostPath, installPath, true, nestedClcMap)
}

// Merge the other class loader context map into this one, do not override existing entries.
// The implicitRootLib parameter is the name of the library for which the other class loader
// context map was constructed. If the implicitRootLib is itself a <uses-library>, it should be
// already present in the class loader context (with the other context as its subcontext) -- in
// that case do not re-add the other context. Otherwise add the other context at the top-level.
func (clcMap ClassLoaderContextMap) AddContextMap(otherClcMap ClassLoaderContextMap, implicitRootLib string) {
	if otherClcMap == nil {
		return
	}

	// If the implicit root of the merged map is already present as one of top-level subtrees, do
	// not merge it second time.
	for _, clc := range clcMap[AnySdkVersion] {
		if clc.Name == implicitRootLib {
			return
		}
	}

	for sdkVer, otherClcs := range otherClcMap {
		for _, otherClc := range otherClcs {
			alreadyHave := false
			for _, clc := range clcMap[sdkVer] {
				if clc.Name == otherClc.Name {
					alreadyHave = true
					break
				}
			}
			if !alreadyHave {
				clcMap[sdkVer] = append(clcMap[sdkVer], otherClc)
			}
		}
	}
}

// List of libraries in the unconditional class loader context, excluding dependencies of shared
// libraries. These libraries should be in the <uses-library> tags in the manifest. Some of them may
// be present in the original manifest, others are added by the manifest_fixer.
func (clcMap ClassLoaderContextMap) UsesLibs() (ulibs []string) {
	if clcMap != nil {
		// compatibility libraries (those in conditional context) are not added to <uses-library> tags
		ulibs = usesLibsRec(clcMap[AnySdkVersion])
		ulibs = android.FirstUniqueStrings(ulibs)
	}
	return ulibs
}

// Helper function for UsesLibs() that handles recursion.
func usesLibsRec(clcs []*ClassLoaderContext) (ulibs []string) {
	for _, clc := range clcs {
		ulibs = append(ulibs, clc.Name)
		// <uses-library> tags in the manifest should not include dependencies of shared libraries,
		// because PackageManager already tracks all such dependencies and automatically adds their
		// class loader contexts as subcontext of the shared library.
		if !clc.IsSharedLibrary {
			ulibs = append(ulibs, usesLibsRec(clc.Subcontexts)...)
		}
	}
	return ulibs
}

// Now that the full unconditional context is known, reconstruct conditional context.
// Apply filters for individual libraries, mirroring what the PackageManager does when it
// constructs class loader context on device.
//
// TODO(b/132357300): remove "android.hidl.manager" and "android.hidl.base" for non-system apps.
//
func fixClassLoaderContext(clcMap ClassLoaderContextMap) {
	usesLibs := clcMap.UsesLibs()

	for sdkVer, clcs := range clcMap {
		if sdkVer == AnySdkVersion {
			continue
		}
		fixedClcs := []*ClassLoaderContext{}
		for _, clc := range clcs {
			if android.InList(clc.Name, usesLibs) {
				// skip compatibility libraries that are already included in unconditional context
			} else if clc.Name == AndroidTestMock && !android.InList("android.test.runner", usesLibs) {
				// android.test.mock is only needed as a compatibility library (in conditional class
				// loader context) if android.test.runner is used, otherwise skip it
			} else {
				fixedClcs = append(fixedClcs, clc)
			}
			clcMap[sdkVer] = fixedClcs
		}
	}
}

// Return true if all build/install library paths are valid (including recursive subcontexts),
// otherwise return false. A build path is valid if it's not nil. An install path is valid if it's
// not equal to a special "error" value.
func validateClassLoaderContext(clcMap ClassLoaderContextMap) (bool, error) {
	for sdkVer, clcs := range clcMap {
		if valid, err := validateClassLoaderContextRec(sdkVer, clcs); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// Helper function for validateClassLoaderContext() that handles recursion.
func validateClassLoaderContextRec(sdkVer int, clcs []*ClassLoaderContext) (bool, error) {
	for _, clc := range clcs {
		if clc.Host == nil || clc.Device == UnknownInstallLibraryPath {
			if sdkVer == AnySdkVersion {
				// Return error if dexpreopt doesn't know paths to one of the <uses-library>
				// dependencies. In the future we may need to relax this and just disable dexpreopt.
				return false, fmt.Errorf("invalid path for <uses-library> \"%s\"", clc.Name)
			} else {
				// No error for compatibility libraries, as Soong doesn't know if they are needed
				// (this depends on the targetSdkVersion in the manifest), but the CLC is invalid.
				return false, nil
			}
		}
		if valid, err := validateClassLoaderContextRec(sdkVer, clc.Subcontexts); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// Return the class loader context as a string, and a slice of build paths for all dependencies.
// Perform a depth-first preorder traversal of the class loader context tree for each SDK version.
// Return the resulting string and a slice of on-host build paths to all library dependencies.
func ComputeClassLoaderContext(clcMap ClassLoaderContextMap) (clcStr string, paths android.Paths) {
	for _, sdkVer := range android.SortedIntKeys(clcMap) { // determinisitc traversal order
		sdkVerStr := fmt.Sprintf("%d", sdkVer)
		if sdkVer == AnySdkVersion {
			sdkVerStr = "any" // a special keyword that means any SDK version
		}
		hostClc, targetClc, hostPaths := computeClassLoaderContextRec(clcMap[sdkVer])
		if hostPaths != nil {
			clcStr += fmt.Sprintf(" --host-context-for-sdk %s %s", sdkVerStr, hostClc)
			clcStr += fmt.Sprintf(" --target-context-for-sdk %s %s", sdkVerStr, targetClc)
		}
		paths = append(paths, hostPaths...)
	}
	return clcStr, android.FirstUniquePaths(paths)
}

// Helper function for ComputeClassLoaderContext() that handles recursion.
func computeClassLoaderContextRec(clcs []*ClassLoaderContext) (string, string, android.Paths) {
	var paths android.Paths
	var clcsHost, clcsTarget []string

	for _, clc := range clcs {
		subClcHost, subClcTarget, subPaths := computeClassLoaderContextRec(clc.Subcontexts)
		if subPaths != nil {
			subClcHost = "{" + subClcHost + "}"
			subClcTarget = "{" + subClcTarget + "}"
		}

		clcsHost = append(clcsHost, "PCL["+clc.Host.String()+"]"+subClcHost)
		clcsTarget = append(clcsTarget, "PCL["+clc.Device+"]"+subClcTarget)

		paths = append(paths, clc.Host)
		paths = append(paths, subPaths...)
	}

	clcHost := strings.Join(clcsHost, "#")
	clcTarget := strings.Join(clcsTarget, "#")

	return clcHost, clcTarget, paths
}

// JSON representation of <uses-library> paths on host and on device.
type jsonLibraryPath struct {
	Host   string
	Device string
}

// Class loader contexts that come from Make (via JSON dexpreopt.config) files have simpler
// structure than Soong class loader contexts: they are flat maps from a <uses-library> name to its
// on-host and on-device paths. There are no nested subcontexts. It is a limitation of the current
// Make implementation.
type jsonClassLoaderContext map[string]jsonLibraryPath

// A map from SDK version (represented with a JSON string) to JSON class loader context.
type jsonClassLoaderContextMap map[string]jsonClassLoaderContext

// Convert JSON class loader context map to ClassLoaderContextMap.
func fromJsonClassLoaderContext(ctx android.PathContext, jClcMap jsonClassLoaderContextMap) ClassLoaderContextMap {
	clcMap := make(ClassLoaderContextMap)
	for sdkVerStr, clc := range jClcMap {
		sdkVer, ok := strconv.Atoi(sdkVerStr)
		if ok != nil {
			if sdkVerStr == "any" {
				sdkVer = AnySdkVersion
			} else {
				android.ReportPathErrorf(ctx, "failed to parse SDK version in dexpreopt.config: '%s'", sdkVerStr)
			}
		}
		for lib, path := range clc {
			clcMap[sdkVer] = append(clcMap[sdkVer], &ClassLoaderContext{
				Name:        lib,
				Host:        constructPath(ctx, path.Host),
				Device:      path.Device,
				Subcontexts: nil,
			})
		}
	}
	return clcMap
}
