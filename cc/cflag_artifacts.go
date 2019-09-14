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
				module := fmt.Sprintf("%s/%s:%s %s",
					ctx.ModuleDir(ccModule),
					ctx.BlueprintFile(ccModule),
					ctx.ModuleName(ccModule),
					ctx.ModuleSubDir(ccModule))
				for _, flag := range TrackedCFlags {
					if inList(flag, cflags) {
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
		cleanedName := strings.Replace(flag, "=", "_", -1)
		var filecontent strings.Builder
		sort.Strings(modulesWithCFlag[flag])
		fmt.Fprintf(&filecontent, "# Modules using %s\\n", flag)
		for _, module := range modulesWithCFlag[flag] {
			fmt.Fprintf(&filecontent, "%s\\n", module)
		}
		sort.Strings(modulesWithCFlag["!"+flag])
		fmt.Fprintf(&filecontent, "# Modules not using %s\\n", flag)
		for _, module := range modulesWithCFlag["!"+flag] {
			fmt.Fprintf(&filecontent, "%s\\n", module)
		}

		filename := "module_cflags" + cleanedName + ".txt"
		filepath := android.PathForOutput(ctx, filename)
		s.outputs = append(s.outputs, filepath.String())
		ctx.Build(pctx, android.BuildParams{
			Rule:        android.WriteFile,
			Description: filename,
			Output:      filepath,
			Args: map[string]string{
				"content": filecontent.String(),
			},
		})
	}
}

func cflagArtifactsTextFactory() android.Singleton {
	return &cflagArtifactsText{}
}

func (s *cflagArtifactsText) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict("SOONG_MODULES_CFLAG_ARTIFACTS", strings.Join(s.outputs, " "))
}
