package testing

import (
	"testing"

	"android/soong/android"
	"android/soong/java"
)

var PrepareForTestWithTestSpecBuildComponents = android.FixtureRegisterWithContext(RegisterBuildComponents)

func runTest(t *testing.T, errorHandler android.FixtureErrorHandler, bp string) *android.TestResult {
	return android.GroupFixturePreparers(
		PrepareForTestWithTestSpecBuildComponents, java.PrepareForIntegrationTestWithJava).
		ExtendWithErrorHandler(errorHandler).
		RunTestWithBp(t, bp)
}
