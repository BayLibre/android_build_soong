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
)

func TestTestRun(t *testing.T) {
	bp := `test_run {
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

	module := result.ModuleForTests("module-name", "").Module().(*TestRunModule)

	// Check that the provider has the right contents
	data := result.ModuleProvider(module, testRunProviderKey).(testRunProviderData)
	if !strings.HasSuffix(data.IntermediatePath.String(), "/intermediateTestRunMetadata.pb") {
		t.Errorf("Missing intermediates path in provider: %s", data.IntermediatePath.String())
	}
}
