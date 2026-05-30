package java

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"android/soong/android"
)

// jvmTunedModule is satisfied by any Java compilation module (Javadoc, Module, Droidstubs)
// that supports dynamically injected JVM tuning flags.
type jvmTunedModule interface {
	jvmFlags() *[]string
}

var (
	readTuningMapOnce sync.Once
	tuningMap         map[string][]string
)

// The performance profile is stored hermetically inside the java build tree
const tuningMapFile = "build/soong/java/jvm_tuning_map.json"

func loadTuningMap(ctx android.BottomUpMutatorContext) {
	// Use ExistentPathForSource to dynamically track jvm_tuning_map.json as a build dependency.
	// Ninja files will ONLY regenerate when this JSON file content changes.
	path := android.ExistentPathForSource(ctx, tuningMapFile)
	if !path.Valid() {
		// Fail silently if the profile is missing (fallback to default AOSP flags safely)
		return
	}

	absPath := filepath.Join(android.AbsSrcDirForExistingUseCases(), path.Path().String())
	bytes, err := os.ReadFile(absPath)
	if err != nil {
		ctx.ModuleErrorf("failed to read JVM tuning map: %s", err)
		return
	}

	var parseMap map[string][]string
	if err := json.Unmarshal(bytes, &parseMap); err != nil {
		ctx.ModuleErrorf("failed to parse JVM tuning map JSON: %s", err)
		return
	}

	tuningMap = parseMap
}

// hyperoptimizerMutator matches the module name against the performance profile
// and injects the tailored, optimized JVM flags into its CommonProperties.
func hyperoptimizerMutator(ctx android.BottomUpMutatorContext) {
	if jm, ok := ctx.Module().(jvmTunedModule); ok {
		readTuningMapOnce.Do(func() {
			loadTuningMap(ctx)
		})

		if tuningMap != nil {
			if flags, found := tuningMap[ctx.ModuleName()]; found {
				jvmFlags := jm.jvmFlags()
				for _, f := range flags {
					f = strings.TrimSpace(f)
					if f == "" {
						continue
					}
					if !strings.HasPrefix(f, "-J") {
						f = "-J" + f
					}
					*jvmFlags = append(*jvmFlags, f)
				}
			}
		}
	}
}
