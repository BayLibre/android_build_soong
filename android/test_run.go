// Copyright 2020 Google Inc. All rights reserved.
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

package android

import (
	"android/soong/android/test_run_proto"
	"github.com/google/blueprint"
)

func init() {
	RegisterTestRunBuildComponents(InitRegistrationContext)
}

// Registers the test run module type.
func RegisterTestRunBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("test_run", TestRunFactory)
}

func TestRunFactory() Module {
	module := &TestRunModule{}

	InitAndroidModule(module)
	InitDefaultableModule(module)
	module.AddProperties(&module.properties)

	return module
}

type TestRunModule struct {
	ModuleBase
	DefaultableModuleBase
	BazelModuleBase

	// Properties for "test_run"
	properties struct {
		// Specifies the name of the test config.
		Name string
		// Specifies the team ID.
		TeamId string
		// Specifies the list of tests covered under this module.
		Tests []string
	}
}

type testsDepTagType struct {
	blueprint.BaseDependencyTag
}

var testsDepTag = testsDepTagType{}

func (module *TestRunModule) DepsMutator(ctx BottomUpMutatorContext) {
	// Validate Properties
	if len(module.properties.TeamId) == 0 {
		ctx.PropertyErrorf("TeamId", "Team Id not found in the test_run module. Hint: Maybe the TeamId property hasn't been properly specified.")
	}
	if len(module.properties.Tests) == 0 {
		ctx.PropertyErrorf("Tests", "Expected to attribute some test but none found. Hint: Maybe the test property hasn't been properly specified.")
	}
	ctx.AddDependency(ctx.Module(), testsDepTag, module.properties.Tests...)
}

var validTestTypes = map[string]bool{
	"android_test":                true,
	"art_cc_test":                 true,
	"cc_test":                     true,
	"cc_test_host":                true,
	"csuite_test":                 true,
	"bootclasspath_fragment_test": true,
	"java_test":                   true,
	"java_test_host":              true,
	"python_test":                 true,
	"python_test_host":            true,
	"rust_test":                   true,
	"rust_test_host":              true,
	"sh_test":                     true,
	"sh_test_host":                true,
	"android_robolectric_test":    true,
}

// Check if a given test type is a valid test type
func isValidTestType(testType string) bool {
	_, ok := validTestTypes[testType]
	return ok
}

// Provider published by TestRun
type testRunProviderData struct {
	IntermediatePath WritablePath
}

var testRunProviderKey = blueprint.NewProvider(testRunProviderData{})

func (module *TestRunModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	for _, m := range ctx.GetDirectDepsWithTag(testsDepTag) {
		if !isValidTestType(ctx.OtherModuleType(m)) {
			ctx.ModuleErrorf("Module :%s is not a valid test target.", m.Name())
		}
	}
	location := ctx.ModuleDir() + "/Android.bp"
	var metadataList []*test_run_proto.TestRun_OwnershipMetadata
	for _, test := range module.properties.Tests {
		metadata := test_run_proto.TestRun_OwnershipMetadata{
			TrendyTeamId: &module.properties.TeamId,
			TargetName:   &test,
			Path:         &location,
		}
		metadataList = append(metadataList, &metadata)
	}
	intermediatePath := PathForModuleOut(ctx, "intermediateTestRunMetadata.pb")
	testRunMetadata := test_run_proto.TestRun{OwnershipMetadataList: metadataList}
	WriteFileRule(ctx, intermediatePath, testRunMetadata.String())

	ctx.SetProvider(testRunProviderKey, testRunProviderData{
		IntermediatePath: intermediatePath})
}
