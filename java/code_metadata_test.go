package java

import (
	"strings"
	"testing"

	"android/soong/android"
	soongTesting "android/soong/testing"
	"android/soong/testing/code_metadata_proto"
	"google.golang.org/protobuf/proto"
)

func TestCodeMetadata(t *testing.T) {
	bp := `code_metadata {
    name: "module-name",
    teamId: "12345",
    code: [
        "foo",
    ]
	}

	java_sdk_library {
    name: "foo",
		srcs: ["a.java"],
	}`
	result := runCodeMetadataTest(t, android.FixtureExpectsNoErrors, bp)

	module := result.ModuleForTests(
		"module-name", "",
	).Module().(*soongTesting.CodeMetadataModule)

	// Check that the provider has the right contents
	data := result.ModuleProvider(
		module, soongTesting.CodeMetadataProviderKey,
	).(soongTesting.CodeMetadataProviderData)
	if !strings.HasSuffix(
		data.IntermediatePath.String(), "/intermediateCodeMetadata.pb",
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

	metadataList := make([]*code_metadata_proto.CodeMetadata_TargetOwnership, 0, 2)
	teamId := "12345"
	bpFilePath := "Android.bp"
	targetName := "foo"
	srcFile := []string{"a.java"}
	expectedMetadataProto := code_metadata_proto.CodeMetadata_TargetOwnership{
		TrendyTeamId: &teamId,
		TargetName:   &targetName,
		Path:         &bpFilePath,
		SourceFiles:  srcFile,
	}
	metadataList = append(metadataList, &expectedMetadataProto)

	CodeMetadataMetadata := code_metadata_proto.CodeMetadata{TargetOwnershipList: metadataList}
	protoData, _ := proto.Marshal(&CodeMetadataMetadata)
	rawData := string(protoData)
	formattedData := strings.ReplaceAll(rawData, "\n", "\\n")
	expectedMetadata := "'" + formattedData + "\\n'"

	if metadata != expectedMetadata {
		t.Errorf(
			"Retrieved metadata: %s is not equal to expectedMetadata: %s", metadata,
			expectedMetadata,
		)
	}
}
func runCodeMetadataTest(
	t *testing.T, errorHandler android.FixtureErrorHandler, bp string,
) *android.TestResult {
	return android.GroupFixturePreparers(
		soongTesting.PrepareForTestWithTestingBuildComponents, prepareForJavaTest,
		PrepareForTestWithJavaSdkLibraryFiles, FixtureWithLastReleaseApis("foo"),
	).
		ExtendWithErrorHandler(errorHandler).
		RunTestWithBp(t, bp)
}
