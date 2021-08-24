// Copyright (C) 2021 The Android Open Source Project
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

package sdk

import (
	"fmt"
	"testing"

	"android/soong/android"
)

// Tests for build_release.go

func TestParseBuildReleaseSet(t *testing.T) {
	t.Run("explicit list", func(t *testing.T) {
		set, err := parseBuildReleaseSet("R,Q,S")
		android.AssertDeepEquals(t, "errors", nil, err)
		android.AssertStringEquals(t, "set", "[Q R S]", set.String())
	})
	t.Run("open range", func(t *testing.T) {
		set, err := parseBuildReleaseSet("R+")
		android.AssertDeepEquals(t, "errors", nil, err)
		android.AssertStringEquals(t, "set", "[R S T]", set.String())
	})
	t.Run("closed range", func(t *testing.T) {
		set, err := parseBuildReleaseSet("R-T")
		android.AssertDeepEquals(t, "errors", nil, err)
		android.AssertStringEquals(t, "set", "[R S T]", set.String())
	})
	t.Run("mixture", func(t *testing.T) {
		set, err := parseBuildReleaseSet("Q,S+")
		android.AssertDeepEquals(t, "errors", nil, err)
		android.AssertStringEquals(t, "set", "[Q S T]", set.String())
	})
	t.Run("invalid release in list", func(t *testing.T) {
		set, err := parseBuildReleaseSet("A,Q")
		android.AssertDeepEquals(t, "set", (*buildReleaseSet)(nil), set)
		android.AssertStringDoesContain(t, "errors", fmt.Sprint(err), `unknown release "A", expected one of Q`)
	})
	t.Run("invalid release in open range", func(t *testing.T) {
		set, err := parseBuildReleaseSet("A+")
		android.AssertDeepEquals(t, "set", (*buildReleaseSet)(nil), set)
		android.AssertStringDoesContain(t, "errors", fmt.Sprint(err), `unknown release "A", expected one of Q`)
	})
	t.Run("invalid release in closed range start", func(t *testing.T) {
		set, err := parseBuildReleaseSet("A-Q")
		android.AssertDeepEquals(t, "set", (*buildReleaseSet)(nil), set)
		android.AssertStringDoesContain(t, "errors", fmt.Sprint(err), `unknown release "A", expected one of Q`)
	})
	t.Run("invalid release in closed range end", func(t *testing.T) {
		set, err := parseBuildReleaseSet("Q-A")
		android.AssertDeepEquals(t, "set", (*buildReleaseSet)(nil), set)
		android.AssertStringDoesContain(t, "errors", fmt.Sprint(err), `unknown release "A", expected one of Q`)
	})
	t.Run("invalid closed range reversed", func(t *testing.T) {
		set, err := parseBuildReleaseSet("S-Q")
		android.AssertDeepEquals(t, "set", (*buildReleaseSet)(nil), set)
		android.AssertStringDoesContain(t, "errors", fmt.Sprint(err), `invalid closed range, start release "S" is later than end release "Q"`)
	})
}

func TestBuildReleaseSetContains(t *testing.T) {
	t.Run("contains", func(t *testing.T) {
		set, _ := parseBuildReleaseSet("Q,S")
		android.AssertBoolEquals(t, "set contains Q", true, set.contains(buildReleaseQ))
		android.AssertBoolEquals(t, "set does not contain R", false, set.contains(buildReleaseR))
		android.AssertBoolEquals(t, "set contains S", true, set.contains(buildReleaseS))
		android.AssertBoolEquals(t, "set does not contain T", false, set.contains(buildReleaseT))
	})
}

func TestPropertyPrunerByBuildRelease(t *testing.T) {
	type nested struct {
		S_only string `supported_build_releases:"S"`
	}

	type testBuildReleasePruner struct {
		Default      string
		Q_and_R_only string `supported_build_releases:"Q-R"`
		R_later      string `supported_build_releases:"R+"`
		Nested       nested
	}

	input := testBuildReleasePruner{
		Default:      "Default",
		Q_and_R_only: "Q_and_R_only",
		R_later:      "R_later",
		Nested: nested{
			S_only: "S_only",
		},
	}

	t.Run("target Q", func(t *testing.T) {
		testStruct := input
		pruner := newPropertyPrunerByBuildRelease(&testStruct, buildReleaseQ)
		pruner.pruneProperties(&testStruct)

		expected := input
		expected.R_later = ""
		expected.Nested.S_only = ""
		android.AssertDeepEquals(t, "test struct", expected, testStruct)
	})

	t.Run("target R", func(t *testing.T) {
		testStruct := input
		pruner := newPropertyPrunerByBuildRelease(&testStruct, buildReleaseR)
		pruner.pruneProperties(&testStruct)

		expected := input
		expected.Nested.S_only = ""
		android.AssertDeepEquals(t, "test struct", expected, testStruct)
	})

	t.Run("target S", func(t *testing.T) {
		testStruct := input
		pruner := newPropertyPrunerByBuildRelease(&testStruct, buildReleaseS)
		pruner.pruneProperties(&testStruct)

		expected := input
		expected.Q_and_R_only = ""
		android.AssertDeepEquals(t, "test struct", expected, testStruct)
	})

	t.Run("target T", func(t *testing.T) {
		testStruct := input
		pruner := newPropertyPrunerByBuildRelease(&testStruct, buildReleaseT)
		pruner.pruneProperties(&testStruct)

		expected := input
		expected.Q_and_R_only = ""
		expected.Nested.S_only = ""
		android.AssertDeepEquals(t, "test struct", expected, testStruct)
	})
}
