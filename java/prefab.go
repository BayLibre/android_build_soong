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

package java

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"android/soong/android"
	"android/soong/cc"
)

// https://google.github.io/prefab/#package-structure
type prefabPackageMetadata struct {
	SchemaVersion int      `json:"schema_version"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Dependencies  []string `json:"dependencies"`
}

// The schema supports more fields than this, but we only need the export
// libraries.
type prefabModuleMetadata struct {
	ExportLibraries []string `json:"export_libraries"`
}

type prefabAndroidMetadata struct {
	Abi string `json:"abi"`
	Api int    `json:"api"`
	Ndk int    `json:"ndk"`
	Stl string `json:"stl"`
}

type prefabProperties struct {
	// Name of the package that will be used to address the modules. For
	// example, if the package is "foo" and contains a module "bar", CMake users
	// will link the target "foo::bar".
	Package_name *string

	// The version of the package. Used to populate build system version
	// information such as CMake's <package name>-config-version.cmake file.
	Version string

	// The STL used by the AAR. Must be one of the following:
	//
	// * c++_shared
	// * c++_static
	// * none
	// * system
	Stl string

	// Shared libraries to expose to consumers.
	Shared_libraries []string

	// Static libraries to expose to consumers.
	Static_libraries []string

	// TODO: Do header libraries need to be named separately?

	// Other prefab packages this package depends on.
	Dependencies []string
}

func (p *prefabProperties) HasModules() bool {
	return len(p.Shared_libraries) > 0 || len(p.Static_libraries) > 0
}

func (p *prefabProperties) PackageName(m android.Module) string {
	if p.Package_name != nil {
		return *p.Package_name
	}

	return m.Name()
}

func findNativeLibrary(ctx android.ModuleContext, name string,
	target android.Target) (*cc.Module, error) {

	var dep *cc.Module
	ctx.VisitDirectDeps(func(m android.Module) {
		if m.Name() != name {
			return
		}

		c, ok := m.(*cc.Module)
		if !ok {
			return
		}

		if c.Target().String() != target.String() {
			return
		}

		if dep != nil {
			panic(fmt.Sprintf("unexpected multiple dependencies for %s %s",
				name, target.String()))
		}

		dep = c
	})

	if dep == nil {
		return nil, fmt.Errorf("could not find a matching dependency for %s %s",
			name, target.String())
	}

	return dep, nil
}

func buildPackageMetadata(ctx android.ModuleContext, rule *android.RuleBuilder,
	props prefabProperties, outDir android.OutputPath) {

	packageJsonPath := outDir.Join(ctx, "prefab.json")

	depsList := props.Dependencies
	if depsList == nil {
		depsList = []string{}
	}

	packageMetadata, err := json.Marshal(&prefabPackageMetadata{
		SchemaVersion: 1,
		Name:          props.PackageName(ctx.Module()),
		Version:       props.Version,
		Dependencies:  depsList,
	})
	if err != nil {
		ctx.ModuleErrorf("failed to marshal package metadata: %s", err)
		return
	}
	rule.Command().WriteFile(packageJsonPath, string(packageMetadata))
}

func ndkAbiNameForTarget(ctx android.ModuleContext, target android.Target) string {
	switch target.Arch.ArchType {
	case android.Arm:
		return "armeabi-v7a"
	case android.Arm64:
		return "arm64-v8a"
	case android.X86:
		return "x86"
	case android.X86_64:
		return "x86_64"
	}

	ctx.ModuleErrorf("Unsupported architecture: %s", target.Arch.ArchType.Name)
	return ""
}

func buildModuleMetadata(ctx android.ModuleContext, rule *android.RuleBuilder,
	name string, moduleDir android.OutputPath) {

	moduleJsonPath := moduleDir.Join(ctx, "module.json")

	moduleMetadata, err := json.Marshal(&prefabModuleMetadata{
		// TODO: Might be possible to infer from the modules.
		// Nothing we export needs this yet.
		ExportLibraries: []string{},
	})
	if err != nil {
		ctx.ModuleErrorf(
			"failed to marshal module metadata for %s: %s", name, err)
		return
	}
	rule.Command().WriteFile(moduleJsonPath, string(moduleMetadata))
}

func minSdkVersionForAar(ctx android.ModuleContext,
	target android.Target) (int, error) {

	sdkStr, err := ctx.PrimaryModule().(*AndroidLibrary).minSdkVersion().
		effectiveVersionString(ctx)
	if err != nil {
		return 0, err
	}
	return cc.NormalizeNdkApiLevelInt(ctx, sdkStr, target.Arch)
}

func determinePrebuiltNdkMajorVersion(ctx android.ModuleContext) int {
	link := ctx.Readlink(android.PathForSource(ctx, "prebuilts/ndk/current"))
	if link[0] != 'r' {
		panic(
			fmt.Sprintf("expected NDK directory %q to begin with \"r\"", link))
	}

	asInt, err := strconv.Atoi(link[1:])
	if err != nil {
		panic(err)
	}

	return asInt
}

var prebuiltNdkMajorVersion_ *int

func prebuiltNdkMajorVersion(ctx android.ModuleContext) int {
	if prebuiltNdkMajorVersion_ == nil {
		version := determinePrebuiltNdkMajorVersion(ctx)
		prebuiltNdkMajorVersion_ = &version
	}
	return *prebuiltNdkMajorVersion_
}

func buildAbiMetadata(ctx android.ModuleContext, rule *android.RuleBuilder,
	module *cc.Module, target android.Target, stl string,
	abiDir android.OutputPath) {

	abiJsonPath := abiDir.Join(ctx, "abi.json")

	api, err := minSdkVersionForAar(ctx, target)
	if err != nil {
		ctx.ModuleErrorf("%s", err)
		return
	}

	abiMetadata, err := json.Marshal(&prefabAndroidMetadata{
		Abi: ndkAbiNameForTarget(ctx, target),
		Api: api,
		Ndk: prebuiltNdkMajorVersion(ctx),
		Stl: stl,
	})
	if err != nil {
		ctx.ModuleErrorf("failed to marshal ABI metadata for %s %s: %s",
			target.String(), module.Name(), err)
	}
	rule.Command().WriteFile(abiJsonPath, string(abiMetadata))
}

func generateAbi(ctx android.ModuleContext, rule *android.RuleBuilder,
	module *cc.Module, target android.Target, stl string,
	lib android.OptionalPath, abiDir android.OutputPath) {

	// Header only libraries do not need lib directories.
	if lib.Valid() {
		buildAbiMetadata(ctx, rule, module, target, stl, abiDir)
		libOut := abiDir.Join(ctx, filepath.Base(lib.String()))
		rule.Command().Text("cp").Input(lib.Path()).Output(libOut)
	}
}

func dedupPaths(paths android.Paths) android.Paths {
	seen := make(map[android.Path]struct{})
	for _, file := range paths {
		seen[file] = struct{}{}
	}
	deduped := android.Paths{}
	for key := range seen {
		deduped = append(deduped, key)
	}
	return deduped
}

func installHeaders(ctx android.ModuleContext, rule *android.RuleBuilder,
	exportedIncludes android.Paths, moduleDir android.OutputPath) {

	installDir := moduleDir.Join(ctx, "include")

	// We get the exported includes from every variant. Dedup to avoid making
	// multiple rules for each.
	dedupedExports := dedupPaths(exportedIncludes)
	for _, exportedDir := range dedupedExports {
		command := rule.Command().
			Text("cp -r").Inputf("%s/*", exportedDir).SboxPath(installDir)
		files := ctx.Glob(
			filepath.Join(exportedDir.String(), "**", "*"), []string{})
		for _, file := range files {
			relPath, err := filepath.Rel(exportedDir.String(), file.String())
			if err != nil {
				ctx.ModuleErrorf("%s", err)
				return
			}

			outPath := installDir.Join(ctx, relPath)
			if ctx.IsDir(file) {
				// No need to handle these.
			} else if ctx.IsRegular(file) {
				command.Implicit(file)
				command.ImplicitOutput(outPath)
			} else {
				ctx.ModuleErrorf("Unhandled file type: %q", file.String())
				return
			}
		}
	}
}

func pathListsEqual(a, b android.Paths) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func exportedHeadersVaryByTarget(ctx android.ModuleContext,
	name string) (bool, error) {

	var headers android.Paths
	for _, target := range ctx.MultiTargets() {
		module, err := findNativeLibrary(ctx, name, target)
		if err != nil {
			return false, err
		}

		if headers == nil {
			headers = module.ExportedIncludeDirs()
		} else {
			if !pathListsEqual(headers, module.ExportedIncludeDirs()) {
				return true, nil
			}
		}
	}

	return false, nil
}

func generateModule(ctx android.ModuleContext, rule *android.RuleBuilder,
	name string, shared bool, stl string, moduleDir android.OutputPath) {

	buildModuleMetadata(ctx, rule, name, moduleDir)

	var propName string
	if shared {
		propName = "prefab.export_shared_libraries"
	} else {
		propName = "prefab.export_static_libraries"
	}

	perTargetHeaders, err := exportedHeadersVaryByTarget(ctx, name)
	if err != nil {
		ctx.PropertyErrorf(propName, "%s", err)
		return
	}

	if !perTargetHeaders {
		module, err := findNativeLibrary(ctx, name, ctx.MultiTargets()[0])
		if err != nil {
			ctx.PropertyErrorf(propName, "%s", err)
			return
		}
		installHeaders(ctx, rule, module.ExportedIncludeDirs(), moduleDir)
	}

	libsDir := moduleDir.Join(ctx, "libs")
	for _, target := range ctx.MultiTargets() {
		module, err := findNativeLibrary(ctx, name, target)
		if err != nil {
			ctx.PropertyErrorf(propName, "%s", err)
			continue
		}

		if module.SdkVersion() == "" {
			ctx.PropertyErrorf(propName, "exports non-NDK library %q", name)
			continue
		}

		depVersion, err := module.NdkApiLevelInt(ctx, target.Arch)
		if err != nil {
			ctx.ModuleErrorf("%s", err)
			continue
		}

		aarVersion, err := minSdkVersionForAar(ctx, target)
		if err != nil {
			ctx.ModuleErrorf("%s", err)
			continue
		}

		if depVersion > aarVersion {
			ctx.ModuleErrorf(
				"exported library %q has minSdkVersion %d which is too high "+
					"for the AAR's minSdkVersion %d", module.Name(), depVersion,
				aarVersion)
			return
		}

		// The target string is very long compared to the arch name, and Windows
		// consumers may run into path length issues, so use the generic arch
		// name.
		abiDir := libsDir.Join(ctx,
			fmt.Sprintf("android.%s", target.Arch.ArchType.Name))
		if perTargetHeaders {
			installHeaders(ctx, rule, module.ExportedIncludeDirs(), abiDir)
		}
		libPath := module.OutputFile()
		generateAbi(ctx, rule, module, target, stl, libPath, abiDir)
	}

	return
}

func generateModules(ctx android.ModuleContext, rule *android.RuleBuilder,
	names []string, shared bool, stl string, modulesDir android.OutputPath) {

	for _, name := range names {
		generateModule(ctx, rule, name, shared, stl, modulesDir.Join(ctx, name))
	}
}

func verify(ctx android.ModuleContext, props prefabProperties) bool {
	switch props.Stl {
	case "c++_shared":
	case "c++_static":
	case "none":
	case "system":
		break
	default:
		ctx.PropertyErrorf("prefab.stl", "unsupported STL: %q", props.Stl)
		return false
	}

	if props.Version == "" {
		ctx.PropertyErrorf("prefab.version", "packages must define a version")
		return false
	}

	// TODO: Verify:
	//
	// * All package names are unique across the build.
	// * Any modules exposed as both shared and static have deconflicted names.
	// * AAR dependencies are of matching or lower minSdkVersion.
	return true
}

func buildPrefabPackage(ctx android.ModuleContext,
	installDir android.OutputPath, props prefabProperties) android.Paths {

	contents := android.Paths{}
	if !verify(ctx, props) {
		return contents
	}

	rule := android.NewRuleBuilder().Sbox(installDir)
	var outDir android.OutputPath = rule.OutputDir().(android.OutputPath)

	buildPackageMetadata(ctx, rule, props, outDir)

	modulesDir := outDir.Join(ctx, "modules")

	generateModules(
		ctx, rule, props.Shared_libraries, true, props.Stl, modulesDir)
	generateModules(
		ctx, rule, props.Static_libraries, false, props.Stl, modulesDir)

	rule.Build(pctx, ctx, "prefab", "Assemble prefab package")

	for _, output := range rule.Outputs() {
		contents = append(contents, output)
	}
	return contents
}
