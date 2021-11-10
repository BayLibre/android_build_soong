package bp2build

import (
	"fmt"
	"strings"

	"android/soong/android"
	metrics_pkg "android/soong/ui/metrics"
	"android/soong/ui/metrics/bp2build_metrics_proto"
)

// Simple metrics struct to collect information about a Blueprint to BUILD
// conversion process.
type CodegenMetrics struct {
	// Total number of Soong modules converted to generated targets
	generatedModuleCount uint64

	// Total number of Soong modules converted to handcrafted targets
	handCraftedModuleCount uint64

	// Total number of unconverted Soong modules
	unconvertedModuleCount uint64

	// Counts of generated Bazel targets per Bazel rule class
	ruleClassCount map[string]uint64

	// List of modules with unconverted deps
	moduleWithUnconvertedDepsMsgs []string

	// List of converted modules
	convertedModules []string
}

func (metrics *CodegenMetrics) Serialize() bp2build_metrics_proto.Bp2BuildMetrics {
	return bp2build_metrics_proto.Bp2BuildMetrics{
		GeneratedModuleCount:          metrics.generatedModuleCount,
		HandCraftedModuleCount:        metrics.handCraftedModuleCount,
		UnconvertedModuleCount:        metrics.unconvertedModuleCount,
		RuleClassCount:                metrics.ruleClassCount,
		ModuleWithUnconvertedDepsMsgs: metrics.moduleWithUnconvertedDepsMsgs,
		ConvertedModules:              metrics.convertedModules,
	}
}

// Print the codegen metrics to stdout.
func (metrics *CodegenMetrics) Print() {
	generatedTargetCount := uint64(0)
	for _, ruleClass := range android.SortedStringKeys(metrics.ruleClassCount) {
		count := metrics.ruleClassCount[ruleClass]
		fmt.Printf("[bp2build] %s: %d targets\n", ruleClass, count)
		generatedTargetCount += count
	}
	fmt.Printf(
		"[bp2build] Generated %d total BUILD targets and included %d handcrafted BUILD targets from %d Android.bp modules.\n With %d modules with unconverted deps \n\t%s",
		generatedTargetCount,
		metrics.handCraftedModuleCount,
		metrics.TotalModuleCount(),
		len(metrics.moduleWithUnconvertedDepsMsgs),
		strings.Join(metrics.moduleWithUnconvertedDepsMsgs, "\n\t"))
}

func (metrics *CodegenMetrics) IncrementRuleClassCount(ruleClass string) {
	metrics.ruleClassCount[ruleClass] += 1
}

func (metrics *CodegenMetrics) IncrementUnconvertedCount() {
	metrics.unconvertedModuleCount += 1
}

func (metrics *CodegenMetrics) TotalModuleCount() uint64 {
	return metrics.handCraftedModuleCount +
		metrics.generatedModuleCount +
		metrics.unconvertedModuleCount
}

func (metrics *CodegenMetrics) Dump(filename string) (err error) {
	ser := metrics.Serialize()
	return metrics_pkg.Save(&ser, filename)
}

type ConversionType int

const (
	Generated ConversionType = iota
	Handcrafted
)

func (metrics *CodegenMetrics) AddConvertedModule(moduleName string, conversionType ConversionType) {
	// Undo prebuilt_ module name prefix modifications
	moduleName = android.RemoveOptionalPrebuiltPrefix(moduleName)
	metrics.convertedModules = append(metrics.convertedModules, moduleName)

	if conversionType == Handcrafted {
		metrics.handCraftedModuleCount += 1
	} else if conversionType == Generated {
		metrics.generatedModuleCount += 1
	}
}
