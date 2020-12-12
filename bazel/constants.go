package bazel

type RunName string

// Below is a list bazel execution run names used through out the
// Platform Build systems.
const (
	// Builds the Android build graph.
	GraphBuildRunName = RunName("graph-build")

	// Perform cquery over the Android code base.
	CqueryGraphBuildRunName = RunName("cquery-graph-build")

	// Run bazel as a ninja executer
	NinjaExecRunName = RunName("ninja-exec")
)

// String returns the name of the run.
func (c RunName) String() string {
	return string(c)
}
