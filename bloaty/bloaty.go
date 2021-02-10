// Package bloaty implements a singleton that measures binary (e.g. ELF
// executable, shared library or Rust rlib) section sizes at build time.
package bloaty

import (
	"strings"
	"sync"

	"android/soong/android"

	"github.com/google/blueprint"
)

const bloatyDescriptorExt = ".bloaty.csv"

var (
	sizeMeasuredPathsLock sync.Mutex
	sizeMeasuredPathsKey  = android.NewOnceKey("sizeMeasuredPaths")
	pctx                  = android.NewPackageContext("android/soong/bloaty")

	// bloaty is used to measure a binary section sizes.
	bloaty = pctx.AndroidStaticRule("bloaty",
		blueprint.RuleParams{
			Command:     "${bloaty} -n 0 --csv ${in} > ${out}",
			CommandDeps: []string{"${bloaty}"},
		})

	// The bloaty merger script is used to combine the outputs from bloaty
	// into a single protobuf.
	bloatyMerger = pctx.AndroidStaticRule("bloatyMerger",
		blueprint.RuleParams{
			Command:     "${bloatyMerger} ${in} ${out}",
			CommandDeps: []string{"${bloatyMerger}"},
		})
)

func init() {
	pctx.HostBinToolVariable("bloaty", "bloaty")
	pctx.HostBinToolVariable("bloatyMerger", "bloaty_merger")
	android.RegisterSingletonType("file_metrics", fileSizesSingleton)
}

// sizeMeasuredMap returns the global slice that contains the paths for which a
// bloaty output will be generated.
func sizeMeasuredSlice(config android.Config) *android.Paths {
	return config.Once(sizeMeasuredPathsKey, func() interface{} {
		paths := make(android.Paths, 0, 1024)
		return &paths
	}).(*android.Paths)
}

// MeasureSizeForPath should be called by binary producers (e.g. in builder.go).
func MeasureSizeForPath(ctx android.ModuleContext, filePath android.WritablePath) {
	sizeFile := android.PathForModuleOut(ctx, filePath.Base()+bloatyDescriptorExt)
	ctx.Build(pctx, android.BuildParams{
		Rule:        bloaty,
		Description: "bloaty " + filePath.Rel(),
		Output:      sizeFile,
		Inputs:      []android.Path{filePath},
	})

	// Keep track of the added file to mark it as dependency for the singleton merging step.
	sizeMeasuredPathsLock.Lock()
	defer sizeMeasuredPathsLock.Unlock()
	s := sizeMeasuredSlice(ctx.Config())
	*s = append(*s, sizeFile)
}

type sizesSingleton struct{}

func fileSizesSingleton() android.Singleton {
	return &sizesSingleton{}
}

func (singleton *sizesSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// In case no file was measured, make sure to call Once at least
	// once, to avoid panicking in the Get below.
	sizeMeasuredSlice(ctx.Config())

	// Generate the list of files that are measured. This is to be able to
	// handle any arbitrary number of files (as opposed to passing the
	// files directly as input to bloaty_merger).
	listPath := android.PathForOutput(ctx, "binary_sizes.lst")
	deps := ctx.Config().Get(sizeMeasuredPathsKey).(*android.Paths)
	android.WriteFileRule(ctx, listPath, strings.Join(deps.Strings(), "\n"))

	ctx.Build(pctx, android.BuildParams{
		Rule:      bloatyMerger,
		Input:     listPath,
		Implicits: *deps,
		Output:    android.PathForOutput(ctx, "binary_sizes.pb"),
	})
}
