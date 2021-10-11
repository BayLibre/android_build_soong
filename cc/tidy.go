// Copyright 2016 Google Inc. All rights reserved.
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

package cc

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/cc/config"
)

type TidyProperties struct {
	// whether to run clang-tidy over C-like sources.
	Tidy *bool

	// Extra flags to pass to clang-tidy
	Tidy_flags []string

	// Extra checks to enable or disable in clang-tidy
	Tidy_checks []string

	// Checks that should be treated as errors.
	Tidy_checks_as_errors []string
}

type tidyFeature struct {
	Properties TidyProperties
}

var quotedFlagRegexp, _ = regexp.Compile(`^-?-[^=]+=('|").*('|")$`)

// When passing flag -name=value, if user add quotes around 'value',
// the quotation marks will be preserved by NinjaAndShellEscapeList
// and the 'value' string with quotes won't work like the intended value.
// So here we report an error if -*='*' is found.
func checkNinjaAndShellEscapeList(ctx ModuleContext, prop string, slice []string) []string {
	for _, s := range slice {
		if quotedFlagRegexp.MatchString(s) {
			ctx.PropertyErrorf(prop, "Extra quotes in: %s", s)
		}
	}
	return proptools.NinjaAndShellEscapeList(slice)
}

func (tidy *tidyFeature) props() []interface{} {
	return []interface{}{&tidy.Properties}
}

func (tidy *tidyFeature) flags(ctx ModuleContext, flags Flags) Flags {
	CheckBadTidyFlags(ctx, "tidy_flags", tidy.Properties.Tidy_flags)
	CheckBadTidyChecks(ctx, "tidy_checks", tidy.Properties.Tidy_checks)

	// Check if tidy is explicitly disabled for this module
	if tidy.Properties.Tidy != nil && !*tidy.Properties.Tidy {
		return flags
	}

	// If not explicitly set, check the global tidy flag
	if tidy.Properties.Tidy == nil && !ctx.Config().ClangTidy() {
		return flags
	}

	flags.Tidy = true

	// Add global WITH_TIDY_FLAGS and local tidy_flags.
	withTidyFlags := ctx.Config().Getenv("WITH_TIDY_FLAGS")
	if len(withTidyFlags) > 0 {
		flags.TidyFlags = append(flags.TidyFlags, withTidyFlags)
	}
	esc := checkNinjaAndShellEscapeList
	flags.TidyFlags = append(flags.TidyFlags, esc(ctx, "tidy_flags", tidy.Properties.Tidy_flags)...)
	// If TidyFlags does not contain -header-filter, add default header filter.
	// Find the substring because the flag could also appear as --header-filter=...
	// and with or without single or double quotes.
	if !android.SubstringInList(flags.TidyFlags, "-header-filter=") {
		defaultDirs := ctx.Config().Getenv("DEFAULT_TIDY_HEADER_DIRS")
		headerFilter := "-header-filter="
		if defaultDirs == "" {
			headerFilter += ctx.ModuleDir() + "/"
		} else {
			headerFilter += "\"(" + ctx.ModuleDir() + "/|" + defaultDirs + ")\""
		}
		flags.TidyFlags = append(flags.TidyFlags, headerFilter)
	}

	// If clang-tidy is not enabled globally, add the -quiet flag.
	if !ctx.Config().ClangTidy() {
		flags.TidyFlags = append(flags.TidyFlags, "-quiet")
		flags.TidyFlags = append(flags.TidyFlags, "-extra-arg-before=-fno-caret-diagnostics")
	}

	extraArgFlags := []string{
		// We might be using the static analyzer through clang tidy.
		// https://bugs.llvm.org/show_bug.cgi?id=32914
		"-D__clang_analyzer__",

		// A recent change in clang-tidy (r328258) enabled destructor inlining, which
		// appears to cause a number of false positives. Until that's resolved, this turns
		// off the effects of r328258.
		// https://bugs.llvm.org/show_bug.cgi?id=37459
		"-Xclang", "-analyzer-config", "-Xclang", "c++-temp-dtor-inlining=false",
	}

	for _, f := range extraArgFlags {
		flags.TidyFlags = append(flags.TidyFlags, "-extra-arg-before="+f)
	}

	tidyChecks := "-checks="
	if checks := ctx.Config().TidyChecks(); len(checks) > 0 {
		tidyChecks += checks
	} else {
		tidyChecks += config.TidyChecksForDir(ctx.ModuleDir())
	}
	if len(tidy.Properties.Tidy_checks) > 0 {
		tidyChecks = tidyChecks + "," + strings.Join(esc(ctx, "tidy_checks",
			config.ClangRewriteTidyChecks(tidy.Properties.Tidy_checks)), ",")
	}
	if ctx.Windows() {
		// https://b.corp.google.com/issues/120614316
		// mingw32 has cert-dcl16-c warning in NO_ERROR,
		// which is used in many Android files.
		tidyChecks = tidyChecks + ",-cert-dcl16-c"
	}
	// https://b.corp.google.com/issues/153464409
	// many local projects enable cert-* checks, which
	// trigger bugprone-reserved-identifier.
	tidyChecks = tidyChecks + ",-bugprone-reserved-identifier*,-cert-dcl51-cpp,-cert-dcl37-c"
	// http://b/153757728
	tidyChecks = tidyChecks + ",-readability-qualified-auto"
	// http://b/155034563
	tidyChecks = tidyChecks + ",-bugprone-signed-char-misuse"
	// http://b/155034972
	tidyChecks = tidyChecks + ",-bugprone-branch-clone"
	// http://b/193716442
	tidyChecks = tidyChecks + ",-bugprone-implicit-widening-of-multiplication-result"
	// Too many existing functions trigger this rule, and fixing it requires large code
	// refactoring. The cost of maintaining this tidy rule outweighs the benefit it brings.
	tidyChecks = tidyChecks + ",-bugprone-easily-swappable-parameters"
	flags.TidyFlags = append(flags.TidyFlags, tidyChecks)

	if ctx.Config().IsEnvTrue("WITH_TIDY") {
		// WITH_TIDY=1 enables clang-tidy globally. There could be many unexpected
		// warnings from new checks and many local tidy_checks_as_errors and
		// -warnings-as-errors can break a global build.
		// So allow all clang-tidy warnings.
		inserted := false
		for i, s := range flags.TidyFlags {
			if strings.Contains(s, "-warnings-as-errors=") {
				// clang-tidy accepts only one -warnings-as-errors
				// replace the old one
				re := regexp.MustCompile(`'?-?-warnings-as-errors=[^ ]* *`)
				newFlag := re.ReplaceAllString(s, "")
				if newFlag == "" {
					flags.TidyFlags[i] = "-warnings-as-errors=-*"
				} else {
					flags.TidyFlags[i] = newFlag + " -warnings-as-errors=-*"
				}
				inserted = true
				break
			}
		}
		if !inserted {
			flags.TidyFlags = append(flags.TidyFlags, "-warnings-as-errors=-*")
		}
	} else if len(tidy.Properties.Tidy_checks_as_errors) > 0 {
		tidyChecksAsErrors := "-warnings-as-errors=" + strings.Join(esc(ctx, "tidy_checks_as_errors", tidy.Properties.Tidy_checks_as_errors), ",")
		flags.TidyFlags = append(flags.TidyFlags, tidyChecksAsErrors)
	}
	return flags
}

func init() {
	android.RegisterSingletonType("tidy_phony_targets", TidyPhonySingleton)
}

// This TidyPhonySingleton generates both tidy-* and obj-* phony targets for C/C++ files.
func TidyPhonySingleton() android.Singleton {
	return &tidyPhonySingleton{}
}

type tidyPhonySingleton struct{}

// Given a final module, generate its tidy/obj phony targets, for all, osName, and subset variants.
func generateTidyObjModuleTarget(ctx android.SingletonContext, module android.Module,
	tidyModulesInDirGroup, objModulesInDirGroup map[string]map[string]android.WritablePaths) {
	allObjFileGroups := make(map[string]android.WritablePaths)     // variant group name => obj file Paths
	allTidyFileGroups := make(map[string]android.WritablePaths)    // variant group name => tidy file Paths
	subsetTidyFileGroups := make(map[string]android.WritablePaths) // subset group name => tidy file Paths
	// For tidy phony targets, generate subset groups.
	// (1) Group Os().Name + "_subset" is generated to include a subset
	//     of the group Os().Name. Current implementation has the first variant
	//     as the subset, but in the future a subset might include more variants.
	// (2) The "subset" variant is a union of all (Os().Name + "_subset") groups.
	m := module.Base()
	m.ObjTargetGroups = make(map[string]android.WritablePath)
	m.TidyTargetGroups = make(map[string]android.WritablePath)

	ctx.VisitAllModuleVariants(module, func(variant android.Module) {
		a := variant.Base()
		if ctx.Config().KatiEnabled() && android.ShouldSkipAndroidMkProcessing(a) {
			return
		}
		for group, files := range a.ObjFileGroups {
			allObjFileGroups[group] = append(allObjFileGroups[group], files...)
		}
		for group, files := range a.TidyFileGroups {
			allTidyFileGroups[group] = append(allTidyFileGroups[group], files...)
			if group != "" {
				subsetGroup := group + "_subset"
				// Now include only the first variant in the subsetGroup.
				// When clang-tidy gets faster, we might include more variants.
				if _, found := subsetTidyFileGroups[subsetGroup]; !found {
					subsetTidyFileGroups[subsetGroup] = files
				}
			}
		}
	})

	if len(subsetTidyFileGroups) > 0 {
		// Add a "subset" group that includes all "*_subset" phony targets.
		var allSubsets android.WritablePaths
		for group, _ := range subsetTidyFileGroups {
			allSubsets = append(allSubsets, android.PathForPhony(ctx, objTidyModuleGroupName(module, group, "tidy")))
		}
		subsetTidyFileGroups["subset"] = allSubsets
	}

	for group, files := range allObjFileGroups {
		groupName := objTidyModuleGroupName(module, group, "obj")
		ctx.Phony(groupName, files.Paths()...)
		m.ObjTargetGroups[group] = android.PathForPhony(ctx, groupName)
	}
	for _, tidyFileGroup := range []map[string]android.WritablePaths{allTidyFileGroups, subsetTidyFileGroups} {
		for group, files := range tidyFileGroup {
			groupName := objTidyModuleGroupName(module, group, "tidy")
			ctx.Phony(groupName, files.Paths()...)
			m.TidyTargetGroups[group] = android.PathForPhony(ctx, groupName)
		}
	}
}

func (m *tidyPhonySingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// For tidy-* directory phony targets, there are different variant groups.
	// tidyModulesInDirGroup[G][D] is for group G, directory D, with Paths
	// of all phony targets to be included into direct dependents of tidy-D_G.
	tidyModulesInDirGroup := make(map[string]map[string]android.WritablePaths)
	// Also for obj-* directory phony targets.
	objModulesInDirGroup := make(map[string]map[string]android.WritablePaths)

	// Generate tidy/obj module targets from the 'final' modules.
	ctx.VisitAllModules(func(module android.Module) {
		if module == ctx.FinalModule(module) {
			generateTidyObjModuleTarget(ctx, module, tidyModulesInDirGroup, objModulesInDirGroup)
		}
		a := module.Base()
		appendToModulesInDirGroup(a.ObjTargetGroups, a, objModulesInDirGroup)
		appendToModulesInDirGroup(a.TidyTargetGroups, a, tidyModulesInDirGroup)
	})

	suffix := ""
	if ctx.Config().KatiEnabled() {
		suffix = "-soong"
	}
	generateObjTidyPhonyTargets(ctx, suffix, "obj", objModulesInDirGroup)
	generateObjTidyPhonyTargets(ctx, suffix, "tidy", tidyModulesInDirGroup)
}

// The name for an obj/tidy module variant group phony target is Name_group-obj/tidy,
func objTidyModuleGroupName(module android.Module, group string, suffix string) string {
	if group == "" {
		return module.Name() + "-" + suffix
	}
	return module.Name() + "_" + group + "-" + suffix
}

// Generate obj-* or tidy-* phony targets.
func generateObjTidyPhonyTargets(ctx android.SingletonContext, suffix string, prefix string, objTidyModulesInDirGroup map[string]map[string]android.WritablePaths) {
	// For each variant group, create a <prefix>-<directory>_group target that
	// depends on all subdirectories and modules in the directory.
	for group, modulesInDir := range objTidyModulesInDirGroup {
		groupSuffix := ""
		if group != "" {
			groupSuffix = "_" + group
		}
		mmTarget := func(dir string) string {
			return prefix + "-" + strings.Replace(filepath.Clean(dir), "/", "-", -1) + groupSuffix
		}
		dirs, topDirs := android.AddAncestors(ctx, modulesInDir, mmTarget)
		// Create a <prefix>-soong_group target that depends on all <prefix>-dir_group of top level dirs.
		var topDirPaths android.Paths
		for _, dir := range topDirs {
			topDirPaths = append(topDirPaths, android.PathForPhony(ctx, mmTarget(dir)))
		}
		ctx.Phony(prefix+suffix+groupSuffix, topDirPaths...)
		// Create a <prefix>-dir_group target that depends on all targets in modulesInDir[dir]
		for _, dir := range dirs {
			if dir != "." && dir != "" {
				ctx.Phony(mmTarget(dir), modulesInDir[dir].Paths()...)
			}
		}
	}
}

// Append (obj|tidy)TargetGroups[group] into (obj|tidy)ModulesInDirGroups[group][blueprintDir].
func appendToModulesInDirGroup(targetGroups map[string]android.WritablePath, a *android.ModuleBase, modulesInDirGroup map[string]map[string]android.WritablePaths) {
	// blueprintDir := android.BlueprintDir(a)
	blueprintDir := a.BlueprintDir()
	for group, phonyPath := range targetGroups {
		if _, found := modulesInDirGroup[group]; !found {
			modulesInDirGroup[group] = make(map[string]android.WritablePaths)
		}
		modulesInDirGroup[group][blueprintDir] = append(modulesInDirGroup[group][blueprintDir], phonyPath)
	}
}
