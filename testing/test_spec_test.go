// Copyright 2023 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testing

import (
	"strings"
	"testing"

	"android/soong/android"
	"android/soong/testing/test_spec_proto"
	"google.golang.org/protobuf/proto"
)

func TestTestSpec(t *testing.T) {
	bp := `test_spec {
    name: "module-name",
    teamId: "12345",
    tests: [
        "java-test-module-name-one",
				"java-test-module-name-two",
    ]
	}

	java_test {
    name: "java-test-module-name-one",
	}
	java_test {
    name: "java-test-module-name-two",
	}`
	result := runTest(t, android.FixtureExpectsNoErrors, bp)

	module := result.ModuleForTests("module-name", "").Module().(*TestSpecModule)

	// Check that the provider has the right contents
	data := result.ModuleProvider(module, testSpecProviderKey).(testSpecProviderData)
	if !strings.HasSuffix(data.IntermediatePath.String(), "/intermediateTestSpecMetadata.pb") {
		t.Errorf("Missing intermediates path in provider: %s", data.IntermediatePath.String())
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
	targetNames := []string{"java-test-module-name-one", "java-test-module-name-two"}

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
		t.Errorf("Retrieved metadata: %s is not equal to expectedMetadata: %s", metadata, expectedMetadata)
	}
}
