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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/cc/config"
	"android/soong/remoteexec"
)

type TidyProperties struct {
	// whether to run clang-tidy over C-like sources.
	Tidy *bool

	// Extra flags to pass to clang-tidy
	Tidy_flags []string

	// A config file passed to clang-tidy as --config-file
	Tidy_config_file *string

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

// Set this const to true when all -warnings-as-errors in tidy_flags
// are replaced with tidy_checks_as_errors.
// Then, that old style usage will be obsolete and an error.
const NoWarningsAsErrorsInTidyFlags = true

// keep this up to date with https://cs.android.com/android/platform/superproject/+/master:build/bazel/rules/cc/clang_tidy.bzl
func (tidy *tidyFeature) flags(ctx ModuleContext, flags Flags) Flags {
	CheckBadTidyFlags(ctx, "tidy_flags", tidy.Properties.Tidy_flags)
	CheckBadTidyChecks(ctx, "tidy_checks", tidy.Properties.Tidy_checks)

	// Check if tidy is explicitly disabled for this module
	if tidy.Properties.Tidy != nil && !*tidy.Properties.Tidy {
		return flags
	}
	// Some projects like external/* and vendor/* have clang-tidy disabled by default,
	// unless they are enabled explicitly with the "tidy:true" property or
	// when TIDY_EXTERNAL_VENDOR is set to true.
	if !proptools.Bool(tidy.Properties.Tidy) &&
		config.NoClangTidyForDir(
			ctx.Config().IsEnvTrue("TIDY_EXTERNAL_VENDOR"),
			ctx.ModuleDir()) {
		return flags
	}
	// If not explicitly disabled, set flags.Tidy to generate .tidy rules.
	// Note that libraries and binaries will depend on .tidy files ONLY if
	// the global WITH_TIDY or module 'tidy' property is true.
	flags.Tidy = true

	// If explicitly enabled, by global WITH_TIDY or local tidy:true property,
	// set flags.NeedTidyFiles to make this module depend on .tidy files.
	// Note that locally set tidy:true is ignored if ALLOW_LOCAL_TIDY_TRUE is not set to true.
	if ctx.Config().IsEnvTrue("WITH_TIDY") || (ctx.Config().IsEnvTrue("ALLOW_LOCAL_TIDY_TRUE") && Bool(tidy.Properties.Tidy)) {
		flags.NeedTidyFiles = true
	}

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
		// Default header filter should include only the module directory,
		// not the out/soong/.../ModuleDir/...
		// Otherwise, there will be too many warnings from generated files in out/...
		// If a module wants to see warnings in the generated source files,
		// it should specify its own -header-filter flag.
		if defaultDirs == "" {
			headerFilter += "^" + ctx.ModuleDir() + "/"
		} else {
			headerFilter += "\"(^" + ctx.ModuleDir() + "/|" + defaultDirs + ")\""
		}
		flags.TidyFlags = append(flags.TidyFlags, headerFilter)
	}

	// Disallow -config-file= in tidy_flags; should use the tidy_config_file property.
	for _, flag := range flags.TidyFlags {
		if len(configFileFlag.FindAllStringSubmatch(flag, -1)) > 0 {
			ctx.PropertyErrorf("tidy_flags", flag+" should be replaced with the tidy_config_file property")
		}
	}
	configFile := String(tidy.Properties.Tidy_config_file)
	if configFile != "" {
		// The file must exist and the path is relative to source root.
		if !android.ExistentPathForSource(ctx, configFile).Valid() {
			ctx.ModuleErrorf("Can't find config file: %s", configFile)
		}
		// NOTE: configFile should not contain spaces, or FindTidyConfigFileInFlags will fail.
		flags.TidyFlags = append(flags.TidyFlags, "-config-file="+configFile)
	}

	// Work around RBE bug in parsing clang-tidy flags, replace "--flag" with "-flag".
	// Some C/C++ modules added local tidy flags like --header-filter= and --extra-arg-before=.
	doubleDash := regexp.MustCompile("^('?)--(.*)$")
	for i, s := range flags.TidyFlags {
		flags.TidyFlags[i] = doubleDash.ReplaceAllString(s, "$1-$2")
	}

	// If clang-tidy is not enabled globally, add the -quiet flag.
	if !ctx.Config().ClangTidy() {
		flags.TidyFlags = append(flags.TidyFlags, "-quiet")
		flags.TidyFlags = append(flags.TidyFlags, "-extra-arg-before=-fno-caret-diagnostics")
	}

	for _, f := range config.TidyExtraArgFlags() {
		flags.TidyFlags = append(flags.TidyFlags, "-extra-arg-before="+f)
	}

	tidyChecks := "-checks="
	if checks := ctx.Config().TidyChecks(); len(checks) > 0 {
		tidyChecks += checks
	} else {
		tidyChecks += config.TidyChecksForDir(ctx.ModuleDir())
	}
	if len(tidy.Properties.Tidy_checks) > 0 {
		// If Tidy_checks contains "-*", ignore all checks before "-*".
		localChecks := tidy.Properties.Tidy_checks
		ignoreGlobalChecks := false
		for n, check := range tidy.Properties.Tidy_checks {
			if check == "-*" {
				ignoreGlobalChecks = true
				localChecks = tidy.Properties.Tidy_checks[n:]
			}
		}
		if ignoreGlobalChecks {
			tidyChecks = "-checks=" + strings.Join(esc(ctx, "tidy_checks",
				config.ClangRewriteTidyChecks(localChecks)), ",")
		} else {
			tidyChecks = tidyChecks + "," + strings.Join(esc(ctx, "tidy_checks",
				config.ClangRewriteTidyChecks(localChecks)), ",")
		}
	}
	tidyChecks = tidyChecks + config.TidyGlobalNoChecks()
	if ctx.Windows() {
		// https://b.corp.google.com/issues/120614316
		// mingw32 has cert-dcl16-c warning in NO_ERROR,
		// which is used in many Android files.
		tidyChecks += ",-cert-dcl16-c"
	}

	flags.TidyFlags = append(flags.TidyFlags, tidyChecks)

	// Embedding -warnings-as-errors in tidy_flags is error-prone.
	// It should be replaced with the tidy_checks_as_errors list.
	for i, s := range flags.TidyFlags {
		if strings.Contains(s, "-warnings-as-errors=") {
			if NoWarningsAsErrorsInTidyFlags {
				ctx.PropertyErrorf("tidy_flags", "should not contain "+s+"; use tidy_checks_as_errors instead.")
			} else {
				fmt.Printf("%s: warning: module %s's tidy_flags should not contain %s, which is replaced with -warnings-as-errors=-*; use tidy_checks_as_errors for your own as-error warnings instead.\n",
					ctx.BlueprintsFile(), ctx.ModuleName(), s)
				flags.TidyFlags[i] = "-warnings-as-errors=-*"
			}
			break // there is at most one -warnings-as-errors
		}
	}
	// Default clang-tidy flags does not contain -warning-as-errors.
	// If a module has tidy_checks_as_errors, add the list to -warnings-as-errors
	// and then append the TidyGlobalNoErrorChecks.
	if len(tidy.Properties.Tidy_checks_as_errors) > 0 {
		tidyChecksAsErrors := "-warnings-as-errors=" +
			strings.Join(esc(ctx, "tidy_checks_as_errors", tidy.Properties.Tidy_checks_as_errors), ",") +
			config.TidyGlobalNoErrorChecks()
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

// Given a final module, add its tidy/obj phony targets to tidy/objModulesInDirGroup.
func collectTidyObjModuleTargets(ctx android.SingletonContext, module android.Module,
	tidyModulesInDirGroup, objModulesInDirGroup map[string]map[string]android.Paths) {
	allObjFileGroups := make(map[string]android.Paths)     // variant group name => obj file Paths
	allTidyFileGroups := make(map[string]android.Paths)    // variant group name => tidy file Paths
	subsetObjFileGroups := make(map[string]android.Paths)  // subset group name => obj file Paths
	subsetTidyFileGroups := make(map[string]android.Paths) // subset group name => tidy file Paths

	// (1) Collect all obj/tidy files into OS-specific groups.
	ctx.VisitAllModuleVariants(module, func(variant android.Module) {
		if ctx.Config().KatiEnabled() && android.ShouldSkipAndroidMkProcessing(variant) {
			return
		}
		if m, ok := variant.(*Module); ok {
			osName := variant.Target().Os.Name
			addToOSGroup(osName, m.objFiles, allObjFileGroups, subsetObjFileGroups)
			addToOSGroup(osName, m.tidyFiles, allTidyFileGroups, subsetTidyFileGroups)
		}
	})

	// (2) Add an all-OS group, with "" or "subset" name, to include all os-specific phony targets.
	addAllOSGroup(ctx, module, allObjFileGroups, "", "obj")
	addAllOSGroup(ctx, module, allTidyFileGroups, "", "tidy")
	addAllOSGroup(ctx, module, subsetObjFileGroups, "subset", "obj")
	addAllOSGroup(ctx, module, subsetTidyFileGroups, "subset", "tidy")

	tidyTargetGroups := make(map[string]android.Path)
	objTargetGroups := make(map[string]android.Path)
	genObjTidyPhonyTargets(ctx, module, "obj", allObjFileGroups, objTargetGroups)
	genObjTidyPhonyTargets(ctx, module, "obj", subsetObjFileGroups, objTargetGroups)
	genObjTidyPhonyTargets(ctx, module, "tidy", allTidyFileGroups, tidyTargetGroups)
	genObjTidyPhonyTargets(ctx, module, "tidy", subsetTidyFileGroups, tidyTargetGroups)

	moduleDir := ctx.ModuleDir(module)
	appendToModulesInDirGroup(tidyTargetGroups, moduleDir, tidyModulesInDirGroup)
	appendToModulesInDirGroup(objTargetGroups, moduleDir, objModulesInDirGroup)
}

func (m *tidyPhonySingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// For tidy-* directory phony targets, there are different variant groups.
	// tidyModulesInDirGroup[G][D] is for group G, directory D, with Paths
	// of all phony targets to be included into direct dependents of tidy-D_G.
	tidyModulesInDirGroup := make(map[string]map[string]android.Paths)
	// Also for obj-* directory phony targets.
	objModulesInDirGroup := make(map[string]map[string]android.Paths)

	// Collect tidy/obj targets from the 'final' modules.
	ctx.VisitAllModules(func(module android.Module) {
		if module == ctx.FinalModule(module) {
			collectTidyObjModuleTargets(ctx, module, tidyModulesInDirGroup, objModulesInDirGroup)
		}
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
func generateObjTidyPhonyTargets(ctx android.SingletonContext, suffix string, prefix string, objTidyModulesInDirGroup map[string]map[string]android.Paths) {
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
				ctx.Phony(mmTarget(dir), modulesInDir[dir]...)
			}
		}
	}
}

// Append (obj|tidy)TargetGroups[group] into (obj|tidy)ModulesInDirGroups[group][moduleDir].
func appendToModulesInDirGroup(targetGroups map[string]android.Path, moduleDir string, modulesInDirGroup map[string]map[string]android.Paths) {
	for group, phonyPath := range targetGroups {
		if _, found := modulesInDirGroup[group]; !found {
			modulesInDirGroup[group] = make(map[string]android.Paths)
		}
		modulesInDirGroup[group][moduleDir] = append(modulesInDirGroup[group][moduleDir], phonyPath)
	}
}

// Add given files to the OS group and subset group.
func addToOSGroup(osName string, files android.Paths, allGroups, subsetGroups map[string]android.Paths) {
	if len(files) > 0 {
		subsetName := osName + "_subset"
		allGroups[osName] = append(allGroups[osName], files...)
		// Now include only the first variant in the subsetGroups.
		// If clang and clang-tidy get faster, we might include more variants.
		if _, found := subsetGroups[subsetName]; !found {
			subsetGroups[subsetName] = files
		}
	}
}

// Add an all-OS group, with groupName, to include all os-specific phony targets.
func addAllOSGroup(ctx android.SingletonContext, module android.Module, phonyTargetGroups map[string]android.Paths, groupName string, objTidyName string) {
	if len(phonyTargetGroups) > 0 {
		var targets android.Paths
		for group, _ := range phonyTargetGroups {
			targets = append(targets, android.PathForPhony(ctx, objTidyModuleGroupName(module, group, objTidyName)))
		}
		phonyTargetGroups[groupName] = targets
	}
}

// Create one phony targets for each group and add them to the targetGroups.
func genObjTidyPhonyTargets(ctx android.SingletonContext, module android.Module, objTidyName string, fileGroups map[string]android.Paths, targetGroups map[string]android.Path) {
	for group, files := range fileGroups {
		groupName := objTidyModuleGroupName(module, group, objTidyName)
		ctx.Phony(groupName, files...)
		targetGroups[group] = android.PathForPhony(ctx, groupName)
	}
}

var (
	// Rules for invoking clang-tidy (a clang-based linter).
	clangTidy, clangTidyRE = pctx.RemoteStaticRules("clangTidy",
		blueprint.RuleParams{
			Depfile: "${out}.d",
			Deps:    blueprint.DepsGCC,
			Command: "CLANG_CMD=$clangCmd TIDY_FILE=$out " +
				"$tidyVars$reTemplate${config.ClangBin}/clang-tidy.sh $in $tidyFlags -- $cFlags",
			CommandDeps: []string{"${config.ClangBin}/clang-tidy.sh", "$ccCmd", "$tidyCmd"},
		},
		&remoteexec.REParams{
			Labels:               map[string]string{"type": "lint", "tool": "clang-tidy", "lang": "cpp"},
			ExecStrategy:         "${config.REClangTidyExecStrategy}",
			Inputs:               []string{"$in", "$implicitInputs"},
			OutputFiles:          []string{"${out}", "${out}.d"},
			ToolchainInputs:      []string{"$ccCmd", "$tidyCmd"},
			EnvironmentVariables: []string{"CLANG_CMD", "TIDY_FILE", "TIDY_TIMEOUT"},
			// Although clang-tidy has an option to "fix" source files, that feature is hardly useable
			// under parallel compilation and RBE. So we assume no OutputFiles here.
			// The clang-tidy fix option is best run locally in single thread.
			// Copying source file back to local caused two problems:
			// (1) New timestamps trigger clang and clang-tidy compilations again.
			// (2) Changing source files caused concurrent clang or clang-tidy jobs to crash.
			Platform: map[string]string{remoteexec.PoolKey: "${config.REClangTidyPool}"},
		}, []string{"cFlags", "ccCmd", "clangCmd", "implicitInputs", "tidyCmd", "tidyFlags", "tidyVars"}, []string{})
)

// Given a module context and its flags, srcFiles, noTidySrcs, timeoutTidySrcs,
// return noTidySrcsMap and tidyVars for tidy calls.
func selectTidyFilesVars(ctx ModuleContext, flags builderFlags, srcFiles, noTidySrcs, timeoutTidySrcs android.Paths) (map[string]bool, string) {
	noTidySrcsMap := make(map[string]bool)
	var tidyVars string
	if flags.tidy {
		for _, path := range noTidySrcs {
			noTidySrcsMap[path.String()] = true
		}
		tidyTimeout := ctx.Config().Getenv("TIDY_TIMEOUT")
		if len(tidyTimeout) > 0 {
			tidyVars += "TIDY_TIMEOUT=" + tidyTimeout + " "
			// add timeoutTidySrcs into noTidySrcsMap if TIDY_TIMEOUT is set
			for _, path := range timeoutTidySrcs {
				noTidySrcsMap[path.String()] = true
			}
		}
	}
	return noTidySrcsMap, tidyVars
}

// Generate ninja rules for the given srcFile in the subdir
// and return the .tidy file path.
func generateTidyRules(ctx ModuleContext, subdir string, srcFile android.Path,
	tidyData *TidyConfigData, ccCmd, ccDesc, tidyVars string,
	shared *SharedFlags, flags builderFlags, moduleFlags string, pathDeps,
	cFlagsDeps android.Paths) android.Path {
	tidyFile := android.ObjPathWithExt(ctx, subdir, srcFile, "tidy")
	tidyCmd := "${config.ClangBin}/clang-tidy"

	rule := clangTidy
	reducedCFlags := moduleFlags
	if ctx.Config().IsEnvTrue("USE_RBE") && ctx.Config().IsEnvTrue("RBE_CLANG_TIDY") {
		rule = clangTidyRE
		// b/248371171, work around RBE input processor problem
		// some cflags rejected by input processor, but usually
		// do not affect included files or clang-tidy
		reducedCFlags = config.TidyReduceCFlags(reducedCFlags)
	}

	sharedCFlags := shareFlags(ctx, shared, "cFlags", reducedCFlags)
	srcRelPath := srcFile.Rel()

	// Add the .tidy rule
	tidyArgs := map[string]string{
		"cFlags":   sharedCFlags,
		"ccCmd":    ccCmd,
		"clangCmd": ccDesc,
		"tidyCmd":  tidyCmd,
		"tidyVars": tidyVars, // short and not shared
	}
	tidyDeps := cFlagsDeps
	tidyConfigFiles, newFlags := FindTidyConfigFiles(tidyData, srcFile.String(), flags.tidyFlags)
	if len(tidyConfigFiles) > 0 {
		// Ignore Android global default clang-tidy checks, but keep
		// only the global disabled checks, which will be appended after
		// the checks specified in a config file.
		newFlags = config.TidyFlagsForSrcFileWithConfig(srcFile, newFlags)
		for _, file := range tidyConfigFiles {
			tidyDeps = append(tidyDeps, android.PathForSource(ctx, file))
		}
		if rule == clangTidyRE {
			tidyArgs["implicitInputs"] = strings.Join(tidyConfigFiles, ",")
		}
	} else {
		newFlags = config.TidyFlagsForSrcFile(srcFile, newFlags)
	}
	tidyArgs["tidyFlags"] = shareFlags(ctx, shared, "tidyFlags", newFlags)
	ctx.Build(pctx, android.BuildParams{
		Rule:        rule,
		Description: "clang-tidy " + srcRelPath,
		Output:      tidyFile,
		Input:       srcFile,
		Implicits:   tidyDeps,
		OrderOnly:   pathDeps,
		Args:        tidyArgs,
	})
	return tidyFile
}

var clangTidyDirs map[string]bool = nil
var mutexForClangTidyDirs sync.Mutex

func GetClangTidyFileDir(file string) string {
	// return a directory path that ends with "/"
	dir, _ := filepath.Split(file)
	if dir == "" {
		return "./"
	}
	return dir
}

// ClangTidyDirs can be mocked in unit tests.
var ClangTidyDirs = func(c android.Config) map[string]bool {
	if clangTidyDirs != nil {
		return clangTidyDirs
	}
	mutexForClangTidyDirs.Lock()
	defer mutexForClangTidyDirs.Unlock()
	if clangTidyDirs != nil {
		return clangTidyDirs
	}
	dirs := make(map[string]bool)
	tidyListFile := c.ClangTidyListFile()
	if tidyListFile != "" {
		file, err := c.Fs().Open(tidyListFile)
		if err != nil {
			panic(fmt.Errorf("cannot open %s", tidyListFile))
		}
		defer file.Close()
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(fmt.Errorf("cannot read %s", tidyListFile))
		}
		for _, line := range strings.Split(string(bytes), "\n") {
			dirs[GetClangTidyFileDir(line)] = true
		}
	}
	clangTidyDirs = dirs
	return dirs
}

// Find all .clang-tidy files in srcFile's dir and parent directories; return the directories.
func findClangTidyDirs(srcFile string, dirCache map[string][]string, tidyDirs map[string]bool) []string {
	// tidyDirs["external/clang/"] is true because external/clang/.clang-tidy exists
	// dirCache["external/clang/lib/Sema/"] should be set to "external/clang/"
	// dirCache["external/clang/lib/"] should be set to "external/clang/"
	// dirCache["bionic"] should be set to ""
	if srcFile == "" {
		return []string{}
	}
	dir := GetClangTidyFileDir(srcFile)
	// The root .clang-tidy directory is "./" and accepted here,
	// but it should be used only in some special tests.
	// It can be used to set up common checks and options for all projects,
	// but we should use the global default check list defined in config/tidy.go
	// and use the default clang-tidy check options.
	if value, ok := dirCache[dir]; ok {
		return value
	}
	result := []string{}
	if _, ok := tidyDirs[dir]; ok {
		result = append(result, dir)
	}
	if dir == "./" {
		return result // stop search after the source root
	}
	parent := GetClangTidyFileDir(dir[:len(dir)-1]) // remove the last slash before split
	result = append(result, findClangTidyDirs(parent, dirCache, tidyDirs)...)
	dirCache[dir] = result
	return result
}

type TidyConfigData struct {
	tidyDirs       map[string]bool     // read-only
	dirCache       map[string][]string // cache of search results
	moduleDir      string              // directory path of the module
	tidyConfigFile string              // "" or value of tidy_config_file
}

func NewTidyConfigData(ctx ModuleContext, flags string) *TidyConfigData {
	// All srcFiles in a module share one tidyDirs and one dirCache to avoid
	// repeated search of the same parent directories.
	return &TidyConfigData{
		tidyDirs:       ClangTidyDirs(ctx.Config()),
		dirCache:       make(map[string][]string),
		moduleDir:      ctx.ModuleDir(),
		tidyConfigFile: FindTidyConfigFileInFlags(flags),
	}
}

var configFileFlag = regexp.MustCompile(`-?-config-file=([^ ]*)`)

// Find the last --config-file=... in flags
func FindTidyConfigFileInFlags(flags string) string {
	matches := configFileFlag.FindAllStringSubmatch(flags, -1)
	if len(matches) > 0 {
		return matches[len(matches)-1][1]
	}
	return ""
}

// Find all .clang-tidy files in srcFile's dir and parent directories.
func findClangTidyFiles(config *TidyConfigData, srcFile string) []string {
	files := []string{}
	for _, dir := range findClangTidyDirs(srcFile, config.dirCache, config.tidyDirs) {
		if dir == "./" {
			files = append(files, ".clang-tidy")
		} else {
			files = append(files, dir+".clang-tidy")
		}
	}
	return files
}

// Given a srcFile, return all .clang-tidy files in srcFile's directory and parent directories.
func FindTidyConfigFiles(config *TidyConfigData, srcFile string, flags string) ([]string, string) {
	// When the config file sets InheritParentConfig to true,
	// it can inherit .clang-tidy files in srcFile's directory and parent directories.
	files := []string{}
	if config.tidyConfigFile != "" {
		files = append(files, config.tidyConfigFile)
	}
	files = append(files, findClangTidyFiles(config, srcFile)...)
	if len(files) > 0 {
		return files, flags // no change of flags
	}
	// If no config file is found up to this point,
	// try to find one .clang-tidy in the Android.bp directory
	// and parent directories. With this Android build feature, users do not need
	// to add tidy_config_file property to modules that include generated files.
	// In this case, this .clang-tidy file must be add to the -config-file flag.
	files = findClangTidyFiles(config, config.moduleDir+"/Android.bp")
	if len(files) > 0 {
		// Even if this .clang-tidy file has InheritParentConfig set to true,
		// it cannot find other .clang-tidy files in the srcFile directory.
		// So, only the files[0] will be used in the dependent file list.
		return files[:1], flags + " -config-file=" + files[0]
	}
	return []string{}, flags
}
