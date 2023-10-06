package testing

import (
	"testing"

	"android/soong/android"
	"android/soong/java"
)

var PrepareForTestWithTestRunBuildComponents = android.FixtureRegisterWithContext(RegisterBuildComponents)

func runTest(t *testing.T, errorHandler android.FixtureErrorHandler, bp string) *android.TestResult {
	return android.GroupFixturePreparers(
		PrepareForTestWithTestRunBuildComponents, java.PrepareForIntegrationTestWithJava).
		ExtendWithErrorHandler(errorHandler).
		RunTestWithBp(t, bp)
}
