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
