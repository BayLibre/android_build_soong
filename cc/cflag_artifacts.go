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

	FileBP = 50
)

// Stores output directories
type cflagArtifactsText struct {
	interOutputs []string
	outputs      []string
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

// incrementFile is used to generate output path object and filename string with
// passed in part number for output.
// e.g. file.txt.0
func (s *cflagArtifactsText) incrementFile(ctx android.SingletonContext, flag string, part int) (string, android.OutputPath) {
	filename := fmt.Sprintf("module_cflags%s.txt.%d", flag, part)
	filepath := android.PathForOutput(ctx, "cflags", filename)
	s.interOutputs = append(s.interOutputs, filepath.String())
	return filename, filepath
}

// GenCFlagArtifactParts is used to generate the build rules which produce the
// intermediary files for each desired C Flag artifact
// e.g. flag.txt.0, flag.txt.1, flag.txt.2...
func (s *cflagArtifactsText) GenCFlagArtifactParts(ctx android.SingletonContext, flag string, cleanedName string, using bool, modules []string, part int) int {
	filename, filepath := s.incrementFile(ctx, cleanedName, part)
	rule := android.NewRuleBuilder()

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

	// Cuts module list into chunks of length FileBP (file breakpoint) and
	// generates a partial artifact (intermediary file) build rule for each chunk.
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
		}
	}

	return part
}

// GenCFlagArtifacts is used to generate the build rules which combine the
// intermediary files into the each of the final C Flag artifact
// e.g. flag.txt.0 + flag.txt.1 + flag.txt.2 -> flag.txt
func (s *cflagArtifactsText) GenCFlagArtifacts(ctx android.SingletonContext) {
	re := regexp.MustCompile(`(module_cflags.+)\.\d+$`)
	currentArtifact := ""
	sort.Strings(s.interOutputs)
	index := 0

	// Takes each intermediary output and generates a build rule that combines
	// each related intermediary file into a single out/dist artifact.
	for index < len(s.interOutputs) {
		match := re.FindStringSubmatch(s.interOutputs[index])
		var targets []string
		currentArtifact = match[1]

		// Get related intermediary files.
		for index < len(s.interOutputs) && match[1] == currentArtifact {
			targets = append(targets, "cflags/"+match[0])
			index++
			if index < len(s.interOutputs) {
				match = re.FindStringSubmatch(s.interOutputs[index])
			}
		}
		if len(targets) > 0 {
			// Generate build rule to combine related intermediary file.
			// Build rule also cleans up intermediary files.
			base := re.FindStringSubmatch(targets[0])
			rule := android.NewRuleBuilder()
			outputpath := android.PathForOutput(ctx, "cflags", base[1])
			rule.Command().
				Text("cat").
				Inputs(android.PathsForOutput(ctx, targets).Paths()).
				FlagWithOutput("> ", outputpath)
			rule.Build(pctx, ctx, base[1], "gen "+base[1])
			s.outputs = append(s.outputs, outputpath.String())
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
				module := fmt.Sprintf("%s:%s~%s",
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
		cleanedName := strings.Replace(flag, "=", "_", -1)
		part := 0
		sort.Strings(modulesWithCFlag[flag])
		part = s.GenCFlagArtifactParts(ctx, flag, cleanedName, true, modulesWithCFlag[flag], part)
		sort.Strings(modulesWithCFlag["!"+flag])
		part = s.GenCFlagArtifactParts(ctx, flag, cleanedName, false, modulesWithCFlag["!"+flag], part)
	}

	// Combine intermediary files into a single C Flag artifact.
	s.GenCFlagArtifacts(ctx)
}

func cflagArtifactsTextFactory() android.Singleton {
	return &cflagArtifactsText{}
}

func (s *cflagArtifactsText) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict("SOONG_MODULES_CFLAG_ARTIFACTS", strings.Join(s.outputs, " "))
}
