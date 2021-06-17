package bp2build

import (
	"android/soong/android"
	"android/soong/bazel"
	"fmt"
)

// Simple metrics struct to collect information about a Blueprint to BUILD
// conversion process.
type CodegenMetrics struct {
	// Total number of Soong/Blueprint modules
	TotalModuleCount int

	// Counts of generated Bazel targets per Bazel rule class
	RuleClassCount map[string]int

	// Total number of handcrafted targets
	handCraftedTargetCount int

	// A map from the original module name to the generated/handcrafted Bazel label.
	NameToLabelMap map[string]string
}

// Log an entry of module name -> Bazel target label.
func (metrics CodegenMetrics) AddNameToLabelEntry(name, label string) {
	// The module name may be prefixed with bazel.BazelTargetModuleNamePrefix if
	// generated from bp2build.
	name = bazel.StripNamePrefix(name)
	if existingLabel, ok := metrics.NameToLabelMap[name]; ok {
		panic(fmt.Errorf(
			"Module '%s' maps to more than one Bazel target label: %s, %s. "+
				"This shouldn't happen. It probably indicates a bug with the bp2build internals.",
			name,
			existingLabel,
			label))
	}
	metrics.NameToLabelMap[name] = label
}

// Print the codegen metrics to stdout.
func (metrics CodegenMetrics) Print() {
	generatedTargetCount := 0
	for _, ruleClass := range android.SortedStringKeys(metrics.RuleClassCount) {
		count := metrics.RuleClassCount[ruleClass]
		fmt.Printf("[bp2build] %s: %d targets\n", ruleClass, count)
		generatedTargetCount += count
	}
	fmt.Printf(
		"[bp2build] Generated %d total BUILD targets and included %d handcrafted BUILD targets from %d Android.bp modules.\n",
		generatedTargetCount,
		metrics.handCraftedTargetCount,
		metrics.TotalModuleCount)
}
