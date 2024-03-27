// Copyright 2017 Google Inc. All rights reserved.
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

package cc

import (
	"android/soong/android"
	"android/soong/android/team_proto"
	"log"
	"testing"

	"github.com/google/blueprint"
	"google.golang.org/protobuf/proto"
)

func TestTestOnlyProvider(t *testing.T) {
	t.Parallel()
	ctx := android.GroupFixturePreparers(
		prepareForCcTest,
		android.FixtureRegisterWithContext(func(ctx android.RegistrationContext) {
			ctx.RegisterModuleType("cc_test_host", TestHostFactory)
		}),
	).RunTestWithBp(t, `
                // These should be test-only
                cc_fuzz { name: "cc-fuzz" }
                cc_test { name: "cc-test", gtest:false }
                cc_benchmark { name: "cc-benchmark" }
                cc_library { name: "cc-library-forced",
                             test_only: true }
                cc_test_library {name: "cc-test-library", gtest: false}
                cc_test_host {name: "cc-test-host", gtest: false}

                // These should not be.
                cc_genrule { name: "cc_genrule", cmd: "echo foo", out: ["out"] }
                cc_library { name: "cc_library" }
                cc_library_static { name: "cc_static" }
                cc_library_shared { name: "cc_library_shared" }

                cc_object { name: "cc-object" }
	`)

	// Visit all modules and ensure only the ones that should
	// marked as test-only are marked as test-only.

	actualTestOnly := make(map[string]bool)
	ctx.VisitAllModules(func(m blueprint.Module) {
		if provider, ok := android.OtherModuleProvider(ctx.TestContext.OtherModuleProviderAdaptor(), m, android.TestOnlyProviderKey); ok {
			if provider.TestOnly {
				actualTestOnly[m.Name()] = provider.TestOnly
			}
		}
	})

	expectedModules := map[string]bool{
		"cc-test":           true,
		"cc-library-forced": true,
		"cc-fuzz":           true,
		"cc-benchmark":      true,
		"cc-test-library":   true,
		"cc-test-host":      true,
	}

	assertMapEntriesEqualAnyOrder(t, "compare maps", expectedModules, actualTestOnly)
}

func TestTestOnlyInTeamsProto(t *testing.T) {
	t.Parallel()
	ctx := android.GroupFixturePreparers(
		android.PrepareForTestWithTeamBuildComponents,
		prepareForCcTest,
		android.FixtureRegisterWithContext(func(ctx android.RegistrationContext) {
			ctx.RegisterParallelSingletonType("all_teams", android.AllTeamsFactory)
			ctx.RegisterModuleType("cc_test_host", TestHostFactory)

		}),
	).RunTestWithBp(t, `
                package { default_team: "someteam"}

                // These should be test-only
                cc_fuzz { name: "cc-fuzz" }
                cc_test { name: "cc-test", gtest:false }
                cc_benchmark { name: "cc-benchmark" }
                cc_library { name: "cc-library-forced",
                             test_only: true }
                cc_test_library {name: "cc-test-library", gtest: false}
                cc_test_host {name: "cc-test-host", gtest: false}

                // These should not be.
                cc_genrule { name: "cc_genrule", cmd: "echo foo", out: ["out"] }
                cc_library { name: "cc_library" }
                cc_library_static { name: "cc_static" }
                cc_library_shared { name: "cc_library_shared" }

                cc_object { name: "cc-object" }
		team {
			name: "someteam",
			trendy_team_id: "cool_team",
		}
	`)

	var teams *team_proto.AllTeams
	teams = getTeamProtoOutput(t, ctx)

	// map of module name -> trendy team name.
	actualTrueModules := make(map[string]bool)
	for _, teamProto := range teams.Teams {
		if Bool(teamProto.TestOnly) {
			actualTrueModules[teamProto.GetTargetName()] = Bool(teamProto.TestOnly)
		}
	}
	expectedModules := map[string]bool{
		"cc-test":           true,
		"cc-library-forced": true,
		"cc-fuzz":           true,
		"cc-benchmark":      true,
		"cc-test-library":   true,
		"cc-test-host":      true,
	}

	assertMapEntriesEqualAnyOrder(t, "compare maps", expectedModules, actualTrueModules)
}

func getTeamProtoOutput(t *testing.T, ctx *android.TestResult) *team_proto.AllTeams {
	teams := new(team_proto.AllTeams)
	config := ctx.SingletonForTests("all_teams")
	allOutputs := config.AllOutputs()

	protoPath := allOutputs[0]

	out := config.MaybeOutput(protoPath)
	outProto := []byte(android.ContentFromFileRuleForTests(t, ctx.TestContext, out))
	if err := proto.Unmarshal(outProto, teams); err != nil {
		log.Fatalln("Failed to parse teams proto:", err)
	}
	return teams
}

func assertMapEntriesEqualAnyOrder(t *testing.T, message string, expected map[string]bool, actual map[string]bool) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Errorf("%s: Expected len %d does not match Actual len %d: Actual (%v)", message, len(expected), len(actual), actual)
		return
	}
	for key, expectedValue := range expected {
		actualValue, found := actual[key]
		if !found {
			t.Errorf("%s: Expected contains key: %s but Actual does not.",
				message, key)
			return
		}
		if expectedValue != actualValue {
			t.Errorf("%s: expected value (%v), for %s does not match actual value (%v)",
				message, expectedValue, key, actualValue)
			return
		}
	}

}
