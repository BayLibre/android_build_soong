package cc

import (
	"fmt"
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

// Stores output directories
type cflagArtifactsText struct {
	outputs []string
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

func (s *cflagArtifactsText) GenerateBuildActions(ctx android.SingletonContext) {
	modulesWithCFlag := make(map[string][]string)

	// Scan through all modules, selecting the ones that are part of the filter, and
	// storing into a map whether or not tracked C flag is used or not.
	ctx.VisitAllModules(func(module android.Module) {
		if ccModule, ok := module.(*Module); ok {
			if allowedDir(ctx.ModuleDir(ccModule)) {
				cflags := ccModule.flags.CFlags
				cppflags := ccModule.flags.CppFlags
				module := fmt.Sprintf("%s:%s %s",
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

	// Traversing map and setting up output files to display results of c flag scan.
	for _, flag := range TrackedCFlags {
		artifactRule := android.NewRuleBuilder()
		cleanedName := strings.Replace(flag, "=", "_", -1)
		filename := "module_cflags" + cleanedName + ".txt"
		filepath := android.PathForOutput(ctx, filename)
		s.outputs = append(s.outputs, filepath.String())

		artifactRule.Command().
			Text("rm").
			FlagWithOutput("-rf ", filepath)

		sort.Strings(modulesWithCFlag[flag])
		artifactRule.Command().
			Textf("echo '# Modules using %s'", flag).
			FlagWithOutput(">> ", filepath)
		for _, module := range modulesWithCFlag[flag] {
			artifactRule.Command().
				Textf("echo '%s'", module).
				FlagWithOutput(">> ", filepath)
		}

		sort.Strings(modulesWithCFlag["!"+flag])
		artifactRule.Command().
			Textf("echo '# Modules not using %s'", flag).
			FlagWithOutput(">> ", filepath)

		for _, module := range modulesWithCFlag["!"+flag] {
			artifactRule.Command().
				Textf("echo '%s'", module).
				FlagWithOutput(">> ", filepath)
		}

		artifactRule.Build(pctx, ctx, filename, "gen "+filename)
	}

}

func cflagArtifactsTextFactory() android.Singleton {
	return &cflagArtifactsText{}
}

func (s *cflagArtifactsText) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict("SOONG_MODULES_CFLAG_ARTIFACTS", strings.Join(s.outputs, " "))
}
