package cc

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"android/soong/android"
)

func init() {
	android.RegisterSingletonType("cflag_artifacts_text", cflagArtifactsTextFactory)
}

var (
	TrackedCFlags = []string{
		"-Wall",
		"-Werror",
		"-Wextra",
		"-Wthread-safety",
		"-O3",
	}

	TrackedCFlagsDir = []string{
		"device/google/",
		"vendor/google/",
	}
)

const FileBP = 50

// Stores output files.
type cflagArtifactsText struct {
	interOutputs []android.OutputPath
	outputs      []android.OutputPath
}

// allowedDir verifies if the directory/project is part of the TrackedCFlagsDir
// filter.
func allowedDir(subdir string) bool {
	subdir += "/"
	for _, prefix := range TrackedCFlagsDir {
		if strings.HasPrefix(subdir, prefix) {
			return true
		}
	}
	return false
}

// incrementFile is used to generate an output path object with the passed in flag
// and part number.
// e.g. FLAG + part # -> out/soong/cflags/module_cflags-FLAG.txt.0
func (s *cflagArtifactsText) incrementFile(ctx android.SingletonContext,
	flag string, part int) (string, android.OutputPath) {

	filename := fmt.Sprintf("module_cflags%s.txt.%d", flag, part)
	filepath := android.PathForOutput(ctx, "cflags", filename)
	s.interOutputs = append(s.interOutputs, filepath)
	return filename, filepath
}

// GenCFlagArtifactParts is used to generate the build rules which produce the
// intermediary files for each desired C Flag artifact
// e.g. module_cflags-FLAG.txt.0, module_cflags-FLAG.txt.1, ...
func (s *cflagArtifactsText) GenCFlagArtifactParts(ctx android.SingletonContext,
	flag string, using bool, modules []string, part int) int {

	cleanedName := strings.Replace(flag, "=", "_", -1)
	filename, filepath := s.incrementFile(ctx, cleanedName, part)
	rule := android.NewRuleBuilder()
	rule.Command().Textf("rm -f %s", filepath.String())

	if using {
		rule.Command().
			Textf("echo '# Modules using %s'", flag).
			FlagWithOutput(">> ", filepath)
	} else {
		rule.Command().
			Textf("echo '# Modules not using %s'", flag).
			FlagWithOutput(">> ", filepath)
	}

	length := len(modules)

	if length == 0 {
		rule.Build(pctx, ctx, filename, "gen "+filename)
		part++
	}

	// Following loop splits the module list for each tracked C Flag into
	// chunks of length FileBP (file breakpoint) and generates a partial artifact
	// (intermediary file) build rule for each split.
	for i := 0; i < length; i += FileBP {
		end := i + FileBP

		if end > length {
			end = length
		}

		rule.Command().
			Textf("for m in %s; do echo $m", strings.Join(modules[i:end], " ")).
			FlagWithOutput(">> ", filepath).
			Text("; done")
		rule.Build(pctx, ctx, filename, "gen "+filename)

		part++
		if i+FileBP < length {
			filename, filepath = s.incrementFile(ctx, cleanedName, part)
			rule = android.NewRuleBuilder()
			rule.Command().Textf("rm -f %s", filepath.String())
		}
	}

	return part
}

// GenCFlagArtifacts is used to generate build rules which combine the
// intermediary files of a specific tracked flag into a single C Flag artifact
// for each tracked flag.
// e.g. module_cflags-FLAG.txt.0 + module_cflags-FLAG.txt.1 = module_cflags-FLAG.txt
func (s *cflagArtifactsText) GenCFlagArtifacts(ctx android.SingletonContext) {
	re := regexp.MustCompile(`(module_cflags.+)\.\d+$`)
	currentArtifact := ""
	length := len(s.interOutputs)
	index := 0

	// Scans through s.interOutputs and creates a build rule for each tracked C
	// Flag that concatenates the associated intermediary file into a single
	// artifact.
	for index < length {
		// Extract the base file name.
		// module_cflags-FLAG.txt.0 -> [module_cflags-FLAG.txt.0 module_cflags-FLAG.txt]
		match := re.FindStringSubmatch(s.interOutputs[index].String())
		var targets []string
		currentArtifact = match[1]

		// Get related intermediary files.
		for index < length && match[1] == currentArtifact {
			// targets should contain only intermediary files associated with current
			// C Flag artifact being tracked.
			targets = append(targets, "cflags/"+match[0])
			index++
			if index < length {
				match = re.FindStringSubmatch(s.interOutputs[index].String())
			}
		}
		if len(targets) > 0 {
			// Generate build rule to combine related intermediary files into a
			// C Flag artifact
			base := re.FindStringSubmatch(targets[0])
			rule := android.NewRuleBuilder()
			outputpath := android.PathForOutput(ctx, "cflags", base[1])
			rule.Command().
				Text("cat").
				Inputs(android.PathsForOutput(ctx, targets).Paths()).
				FlagWithOutput("> ", outputpath)
			rule.Build(pctx, ctx, base[1], "gen "+base[1])
			s.outputs = append(s.outputs, outputpath)
		}
	}
}

func (s *cflagArtifactsText) GenerateBuildActions(ctx android.SingletonContext) {
	modulesWithCFlag := make(map[string][]string)

	// Scan through all modules, selecting the ones that are part of the filter,
	// and then storing into a map which tracks whether or not tracked C flag is
	// used or not.
	ctx.VisitAllModules(func(module android.Module) {
		if ccModule, ok := module.(*Module); ok {
			if allowedDir(ctx.ModuleDir(ccModule)) {
				cflags := ccModule.flags.CFlags
				cppflags := ccModule.flags.CppFlags
				module := fmt.Sprintf("\"%s:%s (%s)\"",
					ctx.BlueprintFile(ccModule),
					ctx.ModuleName(ccModule),
					ctx.ModuleSubDir(ccModule))
				for _, flag := range TrackedCFlags {
					if inList(flag, cflags) || inList(flag, cppflags) {
						modulesWithCFlag[flag] = append(modulesWithCFlag[flag], module)
					} else {
						modulesWithCFlag["!"+flag] = append(modulesWithCFlag["!"+flag], module)
					}
				}
			}
		}
	})

	// Traversing map and setting up rules to produce intermediary files which
	// contain parts of each expected C Flag artifact.
	for _, flag := range TrackedCFlags {
		sort.Strings(modulesWithCFlag[flag])
		part := s.GenCFlagArtifactParts(ctx, flag, true, modulesWithCFlag[flag], 0)
		sort.Strings(modulesWithCFlag["!"+flag])
		s.GenCFlagArtifactParts(ctx, flag, false, modulesWithCFlag["!"+flag], part)
	}

	// Combine intermediary files into a single C Flag artifact.
	s.GenCFlagArtifacts(ctx)
}

func cflagArtifactsTextFactory() android.Singleton {
	return &cflagArtifactsText{}
}

func (s *cflagArtifactsText) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict("SOONG_MODULES_CFLAG_ARTIFACTS", strings.Trim(fmt.Sprintf("%+v", s.outputs), "[]"))
}
