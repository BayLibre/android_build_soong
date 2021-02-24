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
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Provides support for creating test fixtures on which tests can be run. Reduces duplication
// of test setup by allow tests to easily reuse setup code.
//
// Fixture
// =======
// These determine the environment within which a test can be run. Fixtures are mutable and are
// created by FixtureFactory instances and mutated by FixturePreparer instances. They are created by
// first creating a base Fixture (which is essentially empty) and then applying FixturePreparer
// instances to it to modify the environment.
//
// FixtureFactory
// ==============
// These are responsible for creating fixtures. Factories are immutable and are intended to be
// initialized once and reused to create multiple fixtures. Each factory has a list of fixture
// preparers that prepare a fixture for running a test. Factories can also be used to create other
// factories by extending them with additional fixture preparers.
//
// FixturePreparer
// ===============
// These are responsible for modifying a Fixture in preparation for it to run a test. Preparers are
// intended to be immutable and able to prepare multiple Fixture objects simultaneously without
// them sharing any data.
//
// TestResult
// ==========
// These are created by running tests in a Fixture and provide access to the Config and TestContext
// in which the tests were run.
//
// Example
// =======
//
// An exported preparer for use by other packages that need to use java modules.
//
// package java
// var PrepareForJava = FixturePreparers(
//    android.PrepareForArchMutator,
//	  android.PrepareForPrebuilts,
//    FixtureRegisterWithContext(RegisterRequiredBuildComponentsForTest),
//    ...
// )
//
// Some files to use in tests in the java package.
//
// var javaMockFS = android.MockFS{
//		"api/current.txt":        nil,
//		"api/removed.txt":        nil,
//    ...
// }
//
// A package private factory for use for testing java within the java package.
//
// var javaFixtureFactory = NewFixtureFactory(
//    PrepareForJava,
//    FixtureRegisterWithContext(func(ctx android.RegistrationContext) {
//      ctx.RegisterModuleType("test_module", testModule)
//    }),
//    javaMockFS.AddToFixture(),
//    ...
// }
//
// func TestJavaStuff(t *testing.T) {
//   result := javaFixtureFactory.RunTest(t,
//       android.FixtureWithRootAndroidBp(`java_library {....}`),
//       android.MockFS{...}.AddToFixture(),
//   )
//   ... test result ...
// }
//
// package cc
// var PrepareForCC = FixturePreparers(
//    android.PrepareForArchMutator,
//	  android.PrepareForPrebuilts,
//    FixtureRegisterWithContext(RegisterRequiredBuildComponentsForTest),
//    ...
// )
//
// package apex
//
// var PrepareForApex = FixturePreparers(
//    ...
// )
//
// Use modules and mutators from java, cc and apex. Any duplicate preparers (like
// android.PrepareForArchMutator) will be automatically deduped.
//
// var apexFixtureFactory = android.NewFixtureFactory(
//    PrepareForJava,
//    PrepareForCC,
//    PrepareForApex,
// )

// Factory for Fixture objects.
//
// This is configured with a set of FixturePreparer objects that are used to
// initialize each Fixture instance this creates.
type FixtureFactory interface {

	// Creates a copy of this instance and adds some additional preparers.
	Extend(preparers ...FixturePreparer) FixtureFactory

	// Create a Fixture.
	Fixture(t *testing.T, preparers ...FixturePreparer) Fixture

	// Run the test, expecting no errors, returning a TestResult instance.
	//
	// Shorthand for Fixture(t, preparers...).RunTest()
	RunTest(t *testing.T, preparers ...FixturePreparer) *TestResult

	// Run the test with the supplied Android.bp file.
	//
	// Shorthand for RunTest(t, android.FixtureWithRootAndroidBp(bp))
	RunTestWithBp(t *testing.T, bp string) *TestResult
}

// Create a new FixtureFactory that will apply the supplied preparers.
//
// The buildDirSupplier is a pointer to the package level buildDir variable that is initialized by
// the package level setUp method. It has to be a pointer to the variable as the variable will not
// have been initialized at the time the factory is created.
func NewFixtureFactory(buildDirSupplier *string, preparers ...FixturePreparer) FixtureFactory {
	return &fixtureFactory{
		buildDirSupplier: buildDirSupplier,
		preparers:        dedupAndFlattenPreparers(nil, preparers),
	}
}

// A set of mock files to add to the mock file system.
type MockFS map[string][]byte

func (fs MockFS) Merge(extra map[string][]byte) {
	for p, c := range extra {
		fs[p] = c
	}
}

func (fs MockFS) AddToFixture() FixturePreparer {
	return FixtureMergeMockFs(fs)
}

// Modify the config
func FixtureModifyConfig(mutator func(config Config)) FixturePreparer {
	return newSimpleFixturePreparer(func(f *fixture) {
		mutator(f.config)
	})
}

// Modify the config and context
func FixtureModifyConfigAndContext(mutator func(config Config, ctx *TestContext)) FixturePreparer {
	return newSimpleFixturePreparer(func(f *fixture) {
		mutator(f.config, f.ctx)
	})
}

// Modify the context
func FixtureModifyContext(mutator func(ctx *TestContext)) FixturePreparer {
	return newSimpleFixturePreparer(func(f *fixture) {
		mutator(f.ctx)
	})
}

func FixtureRegisterWithContext(registeringFunc func(ctx RegistrationContext)) FixturePreparer {
	return FixtureModifyContext(func(ctx *TestContext) { registeringFunc(ctx) })
}

// Modify the mock filesystem
func FixtureModifyMockFS(mutator func(fs MockFS)) FixturePreparer {
	return newSimpleFixturePreparer(func(f *fixture) {
		mutator(f.mockFS)
	})
}

// Merge the supplied file system into the mock filesystem.
//
// Paths that already exist in the mock file system are overridden.
func FixtureMergeMockFs(mockFS MockFS) FixturePreparer {
	return FixtureModifyMockFS(func(fs MockFS) {
		fs.Merge(mockFS)
	})
}

// Add a file to the mock filesystem
func FixtureAddFile(path string, contents []byte) FixturePreparer {
	return FixtureModifyMockFS(func(fs MockFS) {
		fs[path] = contents
	})
}

// Add a text file to the mock filesystem
func FixtureAddTextFile(path string, contents string) FixturePreparer {
	return FixtureAddFile(path, []byte(contents))
}

// Add the root Android.bp file with the supplied contents.
func FixtureWithRootAndroidBp(contents string) FixturePreparer {
	return FixtureAddTextFile("Android.bp", contents)
}

// Create a composite FixturePreparer that is equivalent to applying each of the supplied
// FixturePreparer instances in order.
func FixturePreparers(preparers ...FixturePreparer) FixturePreparer {
	return &compositeFixturePreparer{dedupAndFlattenPreparers(nil, preparers)}
}

type simpleFixturePreparerVisitor func(preparer *simpleFixturePreparer)

// FixturePreparer is an opaque interface that can change a fixture.
type FixturePreparer interface {
	// visit calls the supplied visitor with each *simpleFixturePreparer instances in this preparer,
	visit(simpleFixturePreparerVisitor)
}

type fixturePreparers []FixturePreparer

func (f fixturePreparers) visit(visitor simpleFixturePreparerVisitor) {
	for _, p := range f {
		p.visit(visitor)
	}
}

// dedupAndFlattenPreparers removes any duplicates and flattens any composite FixturePreparer
// instances.
//
// base      - a list of already flattened and deduped preparers that will be applied first before
//             the list of additional preparers. Any duplicates of these in the additional preparers
//             will be ignored.
//
// preparers - a list of additional unflattened, undeduped preparers that will be applied after the
//             base preparers.
//
// Returns a deduped and flattened list of the preparers minus any that exist in the base preparers.
func dedupAndFlattenPreparers(base []*simpleFixturePreparer, preparers fixturePreparers) []*simpleFixturePreparer {
	var list []*simpleFixturePreparer
	visited := make(map[*simpleFixturePreparer]struct{})

	// Mark the already flattened and deduped preparers, if any, as having been seen so that
	// duplicates of these in the additional preparers will be discarded.
	for _, s := range base {
		visited[s] = struct{}{}
	}

	preparers.visit(func(preparer *simpleFixturePreparer) {
		if _, seen := visited[preparer]; !seen {
			visited[preparer] = struct{}{}
			list = append(list, preparer)
		}
	})
	return list
}

// compositeFixturePreparer is a FixturePreparer created from a list of fixture preparers.
type compositeFixturePreparer struct {
	preparers []*simpleFixturePreparer
}

func (c *compositeFixturePreparer) visit(visitor simpleFixturePreparerVisitor) {
	for _, p := range c.preparers {
		p.visit(visitor)
	}
}

// simpleFixturePreparer is a FixturePreparer that applies a function to a fixture.
type simpleFixturePreparer struct {
	function func(fixture *fixture)
}

func (s *simpleFixturePreparer) visit(visitor simpleFixturePreparerVisitor) {
	visitor(s)
}

func newSimpleFixturePreparer(preparer func(fixture *fixture)) FixturePreparer {
	return &simpleFixturePreparer{function: preparer}
}

// Fixture defines the test environment.
type Fixture interface {
	// Run the test, expecting no errors, returning a TestResult instance.
	RunTest() *TestResult
}

// Provides general test support.
type TestHelper struct {
	T *testing.T
}

func (h *TestHelper) AssertStringEquals(message string, expected string, actual string) {
	h.T.Helper()
	if actual != expected {
		h.T.Errorf("%s: expected %s, actual %s", message, expected, actual)
	}
}

func (h *TestHelper) AssertTrimmedStringEquals(message string, expected string, actual string) {
	h.T.Helper()
	h.AssertStringEquals(message, strings.TrimSpace(expected), strings.TrimSpace(actual))
}

func (h *TestHelper) AssertStringDoesContain(message string, s string, expectedSubstring string) {
	h.T.Helper()
	if !strings.Contains(s, expectedSubstring) {
		h.T.Errorf("%s: could not find %q within %q", message, expectedSubstring, s)
	}
}

func (h *TestHelper) AssertStringDoesNotContain(message string, s string, unexpectedSubstring string) {
	h.T.Helper()
	if strings.Contains(s, unexpectedSubstring) {
		h.T.Errorf("%s: unexpectedly found %q within %q", message, unexpectedSubstring, s)
	}
}

func (h *TestHelper) AssertDeepEquals(message string, expected interface{}, actual interface{}) {
	h.T.Helper()
	if !reflect.DeepEqual(actual, expected) {
		h.T.Errorf("%s: expected:\n  %#v\n got:\n  %#v", message, expected, actual)
	}
}

type TestResult struct {
	TestHelper

	fixture *fixture
	Config  Config
	Ctx     *TestContext
}

var _ FixtureFactory = (*fixtureFactory)(nil)

type fixtureFactory struct {
	buildDirSupplier *string
	preparers        []*simpleFixturePreparer
}

func (f *fixtureFactory) Extend(preparers ...FixturePreparer) FixtureFactory {
	all := append(f.preparers, dedupAndFlattenPreparers(f.preparers, preparers)...)
	return &fixtureFactory{
		buildDirSupplier: f.buildDirSupplier,
		preparers:        all,
	}
}

func (f *fixtureFactory) Fixture(t *testing.T, preparers ...FixturePreparer) Fixture {
	config := TestConfig(*f.buildDirSupplier, nil, "", nil)
	ctx := NewTestContext(config)
	fixture := &fixture{
		factory: f,
		t:       t,
		config:  config,
		ctx:     ctx,
		mockFS:  make(MockFS),
	}

	for _, preparer := range f.preparers {
		preparer.function(fixture)
	}

	for _, preparer := range dedupAndFlattenPreparers(f.preparers, preparers) {
		preparer.function(fixture)
	}

	return fixture
}

func (f *fixtureFactory) RunTest(t *testing.T, preparers ...FixturePreparer) *TestResult {
	t.Helper()
	fixture := f.Fixture(t, preparers...)
	return fixture.RunTest()
}

func (f *fixtureFactory) RunTestWithBp(t *testing.T, bp string) *TestResult {
	t.Helper()
	return f.RunTest(t, FixtureWithRootAndroidBp(bp))
}

type fixture struct {
	factory *fixtureFactory
	t       *testing.T
	config  Config
	ctx     *TestContext
	mockFS  MockFS
}

func (f *fixture) RunTest() *TestResult {
	f.t.Helper()

	ctx := f.ctx

	// The TestConfig() method assumes that the mock filesystem is available when creating so creates
	// the mock file system immediately. Similarly, the NewTestContext(Config) method assumes that the
	// supplied Config's FileSystem has been properly initialized before it is called and so it takes
	// its own reference to the filesystem. However, fixtures create the Config and TestContext early
	// so they can be modified by preparers at which time the mockFS has not been populated (because
	// it too is modified by preparers). So, this reinitializes the Config and TestContext's
	// FileSystem using the now populated mockFS.
	f.config.mockFileSystem("", f.mockFS)
	ctx.SetFs(ctx.config.fs)
	if ctx.config.mockBpList != "" {
		ctx.SetModuleListFile(ctx.config.mockBpList)
	}

	ctx.Register()
	_, errs := ctx.ParseBlueprintsFiles("ignored")
	FailIfErrored(f.t, errs)
	_, errs = ctx.PrepareBuildActions(f.config)
	FailIfErrored(f.t, errs)

	return &TestResult{
		TestHelper: TestHelper{T: f.t},
		fixture:    f,
		Config:     f.config,
		Ctx:        ctx,
	}
}

func (r *TestResult) PathRelativeToBuildDir(path Path) string {
	buildDir := filepath.Clean(r.Config.BuildDir()) + "/"
	return strings.TrimPrefix(filepath.Clean(path.String()), buildDir)
}

func (r *TestResult) PathsRelativeToBuildDir(paths Paths) []string {
	var result []string
	for _, path := range paths {
		result = append(result, r.PathRelativeToBuildDir(path))
	}
	return result
}

func (r *TestResult) NewFixture(preparers ...FixturePreparer) Fixture {
	return r.fixture.factory.Fixture(r.T, preparers...)
}

func (r *TestResult) RunTest(preparers ...FixturePreparer) *TestResult {
	r.T.Helper()
	return r.fixture.factory.Fixture(r.T, preparers...).RunTest()
}

func (r *TestResult) Module(name string, variant string) Module {
	return r.Ctx.ModuleForTests(name, variant).Module()
}

func (r *TestResult) ModuleForTests(name string, variant string) TestingModule {
	return r.Ctx.ModuleForTests(name, variant)
}

func (r *TestResult) ResultForSubTest(t *testing.T) *TestResult {
	subTestResult := *r
	r.T = t
	return &subTestResult
}
