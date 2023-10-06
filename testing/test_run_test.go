package testrun

import (
	"strings"
	"testing"

	"android/soong/android"
)

func TestAconfigDeclarations(t *testing.T) {
	bp := `
		test_run {
    name: "module_name",
    teamId: "12345",
    tests: [
        "CtsHealthFitnessDeviceTestCases",
    ]
}

android_test {
    name: "CtsHealthFitnessDeviceTestCases",
}
	`
	result := runTest(t, android.FixtureExpectsNoErrors, bp)

	module := result.ModuleForTests("module_name", "").Module().(*DeclarationsModule)

	// Check that the provider has the right contents
	depData := result.ModuleProvider(module, declarationsProviderKey).(declarationsProviderData)
	android.AssertStringEquals(t, "package", depData.Package, "com.example.package")
	if !strings.HasSuffix(depData.IntermediatePath.String(), "/intermediate.pb") {
		t.Errorf("Missing intermediates path in provider: %s", depData.IntermediatePath.String())
	}
}
