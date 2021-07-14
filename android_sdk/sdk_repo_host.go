// Copyright (C) 2021 The Android Open Source Project
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

package android_sdk

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/cc/config"
)

var pctx = android.NewPackageContext("android/soong/android_sdk")

func init() {
	registerBuildComponents(android.InitRegistrationContext)
}

func registerBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("android_sdk_repo_host", SdkRepoHostFactory)
}

type sdkRepoHost struct {
	android.ModuleBase
	android.PackagingBase

	properties sdkRepoHostProperties

	outputBaseName string
	outputFile     android.OptionalPath
}

type sdkRepoHostProperties struct {
	// Base directory relative to root, to which deps are installed, e.g. "system". Default is "."
	// (root).
	Base_dir *string

	// List of zip files to merge into the SDK repo.
	Merge_zips []string `android:"arch_variant,path"`

	Deps_remap []string `android:"arch_variant"`

	Srcs []string `android:"arch_variant,path"`

	Strip_files []string `android:"arch_variant"`
}

// android_sdk_repo_host defines an Android SDK repo containing host tools
func SdkRepoHostFactory() android.Module {
	return newSdkRepoHostModule()
}

func newSdkRepoHostModule() *sdkRepoHost {
	s := &sdkRepoHost{}
	s.AddProperties(&s.properties)
	android.InitPackageModule(s)
	android.InitAndroidMultiTargetsArchModule(s, android.HostSupported, android.MultilibCommon)
	return s
}

var dependencyTag = struct {
	blueprint.BaseDependencyTag
	android.PackagingItemAlwaysDepTag
}{}

func (s *sdkRepoHost) DepsMutator(ctx android.BottomUpMutatorContext) {
	s.AddDeps(ctx, dependencyTag)
}

func (s *sdkRepoHost) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	packageSpecs := s.GatherPackagingSpecs(ctx)

	remapPackageSpecs(ctx, packageSpecs, s.properties.Deps_remap)

	noticeMap := map[string]android.Paths{}
	for path, pkgSpec := range packageSpecs {
		licenseFiles := pkgSpec.EffectiveLicenseFiles()
		if len(licenseFiles) > 0 {
			noticeMap[path] = pkgSpec.EffectiveLicenseFiles()
		}
	}
	notices := android.BuildNotices(ctx, noticeMap)

	builder := android.NewRuleBuilder(pctx, ctx)
	dir := android.PathForModuleOut(ctx, "tmp")
	builder.Command().Text("rm").Flag("-rf").Text(dir.String())
	builder.Command().Text("mkdir").Flag("-p").Text(dir.String())

	s.CopySpecsToDir(ctx, builder, packageSpecs, dir)
	builder.Command().Text("cp").
		Input(notices.TxtOutput.Path()).
		Text(filepath.Join(dir.String(), "NOTICE.txt"))

	for _, zip := range android.PathsForModuleSrc(ctx, s.properties.Merge_zips) {
		builder.Command().
			Text("unzip").
			Flag("-DD").
			Flag("-q").
			FlagWithArg("-d ", dir.String()).
			Input(zip)
	}

	for _, src := range android.PathsForModuleSrc(ctx, s.properties.Srcs) {
		builder.Command().
			Text("cp").Input(src).Flag(dir.Join(ctx, src.Rel()).String())
	}

	// Note: this stripping logic was copied over from the old Make implementation
	// It's not using the same flags as the regular stripping support, nor does it
	// support the array of per-module stripping options. It would be nice if we
	// pulled the stripped versions from the CC modules, but that doesn't exist
	// for host tools today. (And not all the things we strip are CC modules today)
	if ctx.Darwin() {
		macStrip := config.MacStripPath(ctx)
		for _, strip := range s.properties.Strip_files {
			builder.Command().
				Text(macStrip).Flag("-x").
				Flag(dir.Join(ctx, strip).String())
		}
	} else {
		llvmStrip := config.ClangTool(ctx, "llvm-strip")
		for _, strip := range s.properties.Strip_files {
			cmd := builder.Command().Tool(llvmStrip)
			if !ctx.Windows() {
				cmd.Flag("-x")
			}
			cmd.Flag(dir.Join(ctx, strip).String())
		}
	}

	// Fix up the line endings of all text files. This also removes executable permissions.
	builder.Command().
		Text("find").
		Flag(dir.String()).
		Flag("-name '*.aidl' -o -name '*.css' -o -name '*.html' -o -name '*.java'").
		Flag("-o -name '*.js' -o -name '*.prop' -o -name '*.template'").
		Flag("-o -name '*.txt' -o -name '*.windows' -o -name '*.xml'").
		// Using -n 500 for xargs to limit the max number of arguments per call to line_endings
		// to 500. This avoids line_endings failing with "arguments too long".
		Text("| xargs -n 500 ").
		BuiltTool("line_endings").
		// TODO: should we specify dos for windows SDKs? We haven't been doing that
		Flag("unix")

	builder.Command().
		Text("find").
		Flag(dir.String()).
		Flag("'(' -name '.*' -o -name '*~' -o -name 'Makefile' -o -name 'Android.mk' ')'").
		Text("| xargs rm -rf")
	builder.Command().
		Text("find").
		Flag(dir.String()).
		Flag("-name '_*' ! -name '__*'").
		Text("| xargs rm -rf")

	outputZipFile := android.PathForModuleOut(ctx, "output.zip")
	builder.Command().
		BuiltTool("soong_zip").
		FlagWithOutput("-o ", outputZipFile).
		FlagWithArg("-P ", proptools.StringDefault(s.properties.Base_dir, ".")).
		FlagWithArg("-C ", dir.String()).
		FlagWithArg("-D ", dir.String())
	builder.Command().Text("rm").Flag("-rf").Text(dir.String())

	builder.Build("build_sdk_repo", "Creating sdk-repo-"+s.BaseModuleName())

	osName := ctx.Os().String()
	if osName == "linux_glibc" {
		osName = "linux"
	}
	name := fmt.Sprintf("sdk-repo-%s-%s", osName, s.BaseModuleName())

	s.outputBaseName = name
	s.outputFile = android.OptionalPathForPath(outputZipFile)
	ctx.InstallFile(android.PathForModuleInstall(ctx, "sdk-repo"), name+".zip", outputZipFile)
}

func (s *sdkRepoHost) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
			fmt.Fprintln(w, ".PHONY:", name, "sdk_repo", "sdk-repo-"+name)
			fmt.Fprintln(w, "sdk_repo", "sdk-repo-"+name+":", strings.Join(s.FilesToInstall().Strings(), " "))

			fmt.Fprintf(w, "$(call dist-for-goals,sdk_repo sdk-repo-%s,%s:new_%s-$(FILE_NAME_TAG).zip)\n\n", s.BaseModuleName(), s.outputFile.String(), s.outputBaseName)
		},
	}
}

func remapPackageSpecs(ctx android.ModuleContext, specs map[string]android.PackagingSpec, remaps []string) {
	for _, remap := range remaps {
		from_to := strings.SplitN(remap, ":", 2)
		if len(from_to) != 2 {
			ctx.PropertyErrorf("deps_remap", "Must be in the form of 'from:to', not %q", remap)
			continue
		}
		for path, spec := range specs {
			if match, err := pathtools.Match(from_to[0], path); err != nil {
				ctx.PropertyErrorf("deps_remap", "Error parsing %q: %v", from_to[0], err)
			} else if match {
				newPath := from_to[1]
				if pathtools.IsGlob(from_to[0]) {
					rel, err := filepath.Rel(constantPartOfPattern(from_to[0]), path)
					if err != nil {
						panic(err)
					}
					newPath = filepath.Join(from_to[1], rel)
				}
				delete(specs, path)
				spec.SetRelPathInPackage(newPath)
				specs[newPath] = spec
			}
		}
	}
}

func constantPartOfPattern(pattern string) string {
	ret := ""
	for pattern != "" {
		var first string
		first, pattern = splitFirst(pattern)
		if pathtools.IsGlob(first) {
			return ret
		}
		ret = filepath.Join(ret, first)
	}
	return ret
}

func splitFirst(path string) (string, string) {
	i := strings.IndexRune(path, filepath.Separator)
	if i < 0 {
		return path, ""
	}
	return path[:i], path[i+1:]
}
