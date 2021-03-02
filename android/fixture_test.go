// Copyright 2021 Google Inc. All rights reserved.
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

import "testing"

// Make sure that FixturePreparer instances are only called once per fixture and in the order in
// which they were added.
func TestFixtureDedup(t *testing.T) {
	list := []string{}

	appendToList := func(s string) FixturePreparer {
		return FixtureModifyConfig(func(_ Config) {
			list = append(list, s)
		})
	}

	preparer1 := appendToList("preparer1")
	preparer2 := appendToList("preparer2")
	preparer3 := appendToList("preparer3")
	preparer4 := appendToList("preparer4")

	preparer1Then2 := GroupFixturePreparers(preparer1, preparer2)

	preparer2Then1 := GroupFixturePreparers(preparer2, preparer1)

	buildDir := "build"
	factory := NewFixtureFactory(&buildDir, preparer1, preparer2, preparer1, preparer1Then2)

	extension := factory.Extend(preparer4, preparer2)

	extension.Fixture(t, preparer1, preparer2, preparer2Then1, preparer3)

	h := TestHelper{t}
	h.AssertDeepEquals("preparers called in wrong order",
		[]string{"preparer1", "preparer2", "preparer4", "preparer3"}, list)
}

func TestFixtureDebug(t *testing.T) {
	checkName := func(t *testing.T, preparer FixturePreparer, expectedName string) {
		h := TestHelper{t}
		h.Helper()

		h.AssertStringEquals("debug name", expectedName, preparer.Name())
	}

	// Check that it correctly records a named function's name, with package but with path.
	t.Run("FixtureRegisterWithContext - named", func(t *testing.T) {
		preparer := FixtureRegisterWithContext(RegisterPackageBuildComponents).(*simpleFixturePreparer)
		checkName(t, preparer, "android.RegisterPackageBuildComponents")
	})

	// The next set of  tests just check to make sure that the name has been set from the function so
	// use an anonymous function as that is simplest.

	// Check that the name of the preparer has been set to local anonymous.
	ensureLocalAnonymous := func(t *testing.T, preparer FixturePreparer) {
		h := TestHelper{t}
		h.Helper()

		h.AssertStringDoesContain("debug name", preparer.Name(), "android.TestFixtureDebug")
	}

	t.Run("FixtureRegisterWithContext", func(t *testing.T) {
		preparer := FixtureRegisterWithContext(func(ctx RegistrationContext) {})
		ensureLocalAnonymous(t, preparer)
	})

	t.Run("FixtureModifyConfig", func(t *testing.T) {
		preparer := FixtureModifyConfig(func(Config) {})
		ensureLocalAnonymous(t, preparer)
	})

	t.Run("FixtureModifyContext", func(t *testing.T) {
		preparer := FixtureModifyContext(func(*TestContext) {})
		ensureLocalAnonymous(t, preparer)
	})

	t.Run("FixtureModifyMockFS", func(t *testing.T) {
		preparer := FixtureModifyMockFS(func(MockFS) {})
		ensureLocalAnonymous(t, preparer)
	})

	// The next set of tests check that the name is set from the files being added to the mock file
	// system.

	t.Run("FixtureMergeMockFs", func(t *testing.T) {
		preparer := FixtureMergeMockFs(MockFS{"path1": nil, "path2": nil, "other": nil})
		checkName(t, preparer, "MockFS{other,path1,path2}")
	})

	t.Run("FixtureAddTextFile", func(t *testing.T) {
		preparer := FixtureAddTextFile("path", "contents")
		checkName(t, preparer, "MockFS{path}")
	})

	t.Run("FixtureAddFile", func(t *testing.T) {
		preparer := FixtureAddFile("path", []byte("contents"))
		checkName(t, preparer, "MockFS{path}")
	})

	t.Run("FixtureWithRootAndroidBp", func(t *testing.T) {
		preparer := FixtureWithRootAndroidBp("contents")
		checkName(t, preparer, "MockFS{Android.bp}")
	})

	// Check that the name can be overridden.
	t.Run("Custom Name", func(t *testing.T) {
		preparer := FixtureAddFile("path", []byte("contents")).SetName("FilesForMe")
		checkName(t, preparer, "FilesForMe")
	})
}
