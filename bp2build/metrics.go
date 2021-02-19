package bp2build

// Simple metrics struct to collect information about a Blueprint to BUILD
// conversion process.
type CodegenMetrics struct {
	// Total number of Soong/Blueprint modules
	TotalModuleCount int

	// Counts of generated Bazel targets per Bazel rule class
	RuleClassCount map[string]int
}
