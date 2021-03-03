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

import (
	"regexp"
	"testing"
)

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

	t.Run("FixtureModifyEnv", func(t *testing.T) {
		preparer := FixtureModifyEnv(func(map[string]string) {})
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

	// The next set of tests check that the name is set from the env variables being added.
	t.Run("FixtureMergeEnv", func(t *testing.T) {
		preparer := FixtureMergeEnv(map[string]string{"VAR": "value", "ALPHA": "two"})
		checkName(t, preparer, "Env{ALPHA,VAR}")
	})

	// Check that the name can be overridden.
	t.Run("Custom Name", func(t *testing.T) {
		preparer := FixtureAddFile("path", []byte("contents")).SetName("FilesForMe")
		checkName(t, preparer, "FilesForMe")
	})
}

func TestFixturePartialOrdering(t *testing.T) {

	mutator := func(name string) FixturePreparer {
		return FixtureRegisterWithContext(func(ctx RegistrationContext) {
			ctx.PreDepsMutators(func(ctx RegisterMutatorsContext) {
				ctx.BottomUp(name, func(ctx BottomUpMutatorContext) {
					if order, ok := ctx.Module().(*orderModule); ok {
						order.properties.List = append(order.properties.List, name)
					}
				})
			})
		}).SetName(name)
	}

	// In the following test descriptions the symbols have the following meanings:
	// a>b - means `a` comes before and requires `b`.
	// a>?b - means `a` comes before but does not require `b`.
	// a?>b - means `b` comes after but does not require `a`.

	t.Run("a?>b", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")

		a.PrecedesAndRequires(b)

		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a", "b"}, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b"}, b)
		})
		t.Run("b,a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a", "b"}, b, a)
		})
		t.Run("a,b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a", "b"}, a, b)
		})
	})

	t.Run("a>?b", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")

		a.PrecedesButDoesNotRequire(b)

		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a"}, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b"}, b)
		})
		t.Run("b,a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a", "b"}, b, a)
		})
		t.Run("a,b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a", "b"}, a, b)
		})
	})

	t.Run("a?<b", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")

		a.FollowsAndRequires(b)

		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b", "a"}, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b"}, b)
		})
		t.Run("b,a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b", "a"}, b, a)
		})
		t.Run("a,b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b", "a"}, a, b)
		})
	})

	t.Run("a<?b", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")

		a.FollowsButDoesNotRequire(b)

		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a"}, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b"}, b)
		})
		t.Run("b,a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b", "a"}, b, a)
		})
		t.Run("a,b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b", "a"}, a, b)
		})
	})

	t.Run("a<c,b<c", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")
		c := mutator("c")

		a.AlwaysFollows(c)
		b.AlwaysFollows(c)

		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"c", "a", "b"}, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"c", "b", "a"}, b)
		})
		t.Run("c", func(t *testing.T) {
			checkTopologicalSort(t, []string{"c", "a", "b"}, c)
		})
		t.Run("a,b,c", func(t *testing.T) {
			checkTopologicalSort(t, []string{"c", "a", "b"}, a, b, c)
		})
	})

	t.Run("a<?c,b<c,c<?d,d<?e", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")
		c := mutator("c")
		d := mutator("d")
		e := mutator("e")

		a.FollowsButDoesNotRequire(c)
		b.FollowsAndRequires(c)
		c.FollowsButDoesNotRequire(d)
		d.FollowsButDoesNotRequire(e)

		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a"}, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"c", "b"}, b)
		})
		t.Run("e,b,a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"e", "c", "b", "a"}, e, b, a)
		})
	})

	t.Run("GroupFixturePreparers(a,b,c)", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")
		c := mutator("c")

		GroupFixturePreparers(a, b, c)

		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a"}, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b"}, b)
		})
		t.Run("c", func(t *testing.T) {
			checkTopologicalSort(t, []string{"c"}, c)
		})
		t.Run("c,b,a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"c", "b", "a"}, c, b, a)
		})
	})

	t.Run("OrderFixturePreparers(a,b,c)", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")
		c := mutator("c")

		OrderFixturePreparers(a, b, c)

		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a"}, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, []string{"b"}, b)
		})
		t.Run("c", func(t *testing.T) {
			checkTopologicalSort(t, []string{"c"}, c)
		})
		t.Run("c,b,a", func(t *testing.T) {
			checkTopologicalSort(t, []string{"a", "b", "c"}, c, b, a)
		})
	})

	t.Run("LinkFixturePreparers(a,b,c)", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")
		c := mutator("c")

		LinkFixturePreparers(a, b, c)

		expectedOrder := []string{"a", "b", "c"}
		t.Run("a", func(t *testing.T) {
			checkTopologicalSort(t, expectedOrder, a)
		})
		t.Run("b", func(t *testing.T) {
			checkTopologicalSort(t, expectedOrder, b)
		})
		t.Run("c", func(t *testing.T) {
			checkTopologicalSort(t, expectedOrder, c)
		})
		t.Run("c,b,a", func(t *testing.T) {
			checkTopologicalSort(t, expectedOrder, c, b, a)
		})
	})

	t.Run("cycle - a>?b?>c?>a?", func(t *testing.T) {
		a := mutator("a")
		b := mutator("b")
		c := mutator("c")

		OrderFixturePreparers(a, b, c)
		c.PrecedesButDoesNotRequire(a)

		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(string); ok {
					if regexp.MustCompile("(?s:Cycle\\(s\\) detected.* a \\(.* b \\(.* c \\(.*)").MatchString(err) {
						return
					}
					t.Errorf("unexpected failure: %s", err)
					return
				}
				t.Errorf("unexpected failure: %s", r)
				return
			}
			t.Errorf("expected error but did not detect one")
		}()

		expectedOrder := []string{"a", "b", "c"}
		checkTopologicalSort(t, expectedOrder, a)
	})
}

func checkTopologicalSort(t *testing.T, expected []string, preparers ...FixturePreparer) {
	result := emptyTestFixtureFactory.Extend(
		prepareForOrderTest,
		FixtureWithRootAndroidBp(`
				order {
					name: "order"
				}
		`),
	).
		RunTest(t, preparers...)

	orderModule := result.Module("order", "").(*orderModule)
	list := orderModule.properties.List

	result.AssertArrayString("order mismatch", expected, list)
}

var prepareForOrderTest = FixtureRegisterWithContext(func(ctx RegistrationContext) {
	ctx.RegisterModuleType("order", func() Module {
		module := &orderModule{}
		module.AddProperties(&module.properties)
		InitAndroidModule(module)
		return module
	})
})

type orderProperties struct {
	List []string `blueprint:"mutated"`
}

type orderModule struct {
	ModuleBase

	properties orderProperties
}

func (o *orderModule) GenerateAndroidBuildActions(_ ModuleContext) {}
