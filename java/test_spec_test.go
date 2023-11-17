package java

import (
	"strings"
	"testing"

	"android/soong/android"
	soongTesting "android/soong/testing"
	"android/soong/testing/test_spec_proto"
	"google.golang.org/protobuf/proto"
)

func TestTestSpec(t *testing.T) {
	bp := `test_spec {
  name: "module-name",
  teamId: "12345",
  tests: [
    "java-test-module-name-one",
    "java-test-module-name-two"
  ]
}

java_test {
  name: "java-test-module-name-one",
}

java_test {
  name: "java-test-module-name-two",
}`
	result := runTest(t, android.FixtureExpectsNoErrors, bp)

	module := result.ModuleForTests(
		"module-name", "",
	).Module().(*soongTesting.TestSpecModule)

	// Check that the provider has the right contents
	data := result.ModuleProvider(
		module, soongTesting.TestSpecProviderKey,
	).(soongTesting.TestSpecProviderData)
	if !strings.HasSuffix(
		data.IntermediatePath.String(), "/intermediateTestSpecMetadata.pb",
	) {
		t.Errorf(
			"Missing intermediates path in provider: %s",
			data.IntermediatePath.String(),
		)
	}

	buildParamsSlice := module.BuildParamsForTests()
	var metadata = ""
	for _, params := range buildParamsSlice {
		if params.Rule.String() == "android/soong/android.writeFile" {
			metadata = params.Args["content"]
		}
	}

	metadataList := make([]*test_spec_proto.TestSpec_OwnershipMetadata, 0, 2)
	teamId := "12345"
	bpFilePath := "Android.bp"
	targetNames := []string{
		"java-test-module-name-one", "java-test-module-name-two",
	}

	for _, test := range targetNames {
		targetName := test
		metadata := test_spec_proto.TestSpec_OwnershipMetadata{
			TrendyTeamId: &teamId,
			TargetName:   &targetName,
			Path:         &bpFilePath,
		}
		metadataList = append(metadataList, &metadata)
	}
	testSpecMetadata := test_spec_proto.TestSpec{OwnershipMetadataList: metadataList}
	protoData, _ := proto.Marshal(&testSpecMetadata)
	rawData := string(protoData)
	formattedData := strings.ReplaceAll(rawData, "\n", "\\n")
	expectedMetadata := "'" + formattedData + "\\n'"

	if metadata != expectedMetadata {
		t.Errorf(
			"Retrieved metadata: %s is not equal to expectedMetadata: %s", metadata,
			expectedMetadata,
		)
	}

	// Tests for all_test_spec singleton.
	singleton := result.SingletonForTests("all_test_specs")
	writeFileRuleData := singleton.Rule("android/soong/android.writeFile")
	allTestSpecsData := singleton.Rule("android/soong/testing.all_test_specs")
	expectedWriteFileRuleInput := "out/soong/.intermediates/module-name/intermediateTestSpecMetadata.pb"
	expectedWriteFileRuleOutput := "out/soong/ownership/all_test_spec_paths.txt"
	expectedSingletonInput := "out/soong/ownership/all_test_spec_paths.txt"
	expectedWriteFileOutput := "out/soong/ownership/all_test_specs.pb"

	if !strings.Contains(
		strings.TrimSpace(writeFileRuleData.Inputs.Strings()[0]),
		expectedWriteFileRuleInput,
	) {
		t.Errorf(
			"Retrieved writeFileRuleInput: %s is not equal to expectedWriteFileRuleInput: %s",
			writeFileRuleData.Inputs.Strings()[0], expectedWriteFileRuleInput,
		)
	}

	if !strings.Contains(
		strings.TrimSpace(writeFileRuleData.Output.String()),
		expectedWriteFileRuleOutput,
	) {
		t.Errorf(
			"Retrieved writeFileRuleOutput: %s is not equal to expectedWriteFileRuleOutput: %s",
			writeFileRuleData.Output.String(), expectedWriteFileRuleOutput,
		)
	}

	if !strings.Contains(
		strings.TrimSpace(allTestSpecsData.Input.String()), expectedSingletonInput,
	) {
		t.Errorf(
			"Retrieved singletonOutput: %s is not equal to expectedSingletonOutput: %s",
			allTestSpecsData.Input.String(), expectedSingletonInput,
		)
	}

	if !strings.Contains(
		strings.TrimSpace(allTestSpecsData.Output.String()),
		expectedWriteFileOutput,
	) {
		t.Errorf(
			"Retrieved singletonOutput: %s is not equal to expectedSingletonOutput: %s",
			allTestSpecsData.Output.String(), expectedWriteFileOutput,
		)
	}
}

func runTest(
	t *testing.T, errorHandler android.FixtureErrorHandler, bp string,
) *android.TestResult {
	return android.GroupFixturePreparers(
		soongTesting.PrepareForTestWithTestSpecBuildComponents,
		PrepareForIntegrationTestWithJava,
	).
		ExtendWithErrorHandler(errorHandler).
		RunTestWithBp(t, bp)
}
