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

package java

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/dexpreopt"
	"android/soong/genrule"
	"android/soong/python"
)

var buildDir string

func setUp() {
	var err error
	buildDir, err = ioutil.TempDir("", "soong_java_test")
	if err != nil {
		panic(err)
	}
}

func tearDown() {
	os.RemoveAll(buildDir)
}

// Factory to use to create fixtures for tests in this package.
var javaFixtureFactory = android.NewFixtureFactory(
	&buildDir,
	genrule.PrepareForTestWithGenRuleBuildComponents,
	// Get the CC build components but not default modules.
	cc.PrepareForTestWithCcBuildComponents,
	// Include all the default java modules.
	PrepareForTestWithJavaDefaultModules,
	android.FixtureRegisterWithContext(func(ctx android.RegistrationContext) {
		ctx.RegisterModuleType("java_plugin", PluginFactory)
		ctx.RegisterModuleType("python_binary_host", python.PythonBinaryHostFactory)

		ctx.PreDepsMutators(python.RegisterPythonPreDepsMutators)
		ctx.RegisterPreSingletonType("overlay", OverlaySingletonFactory)
		ctx.RegisterPreSingletonType("sdk_versions", sdkPreSingletonFactory)
	}),
	javaMockFS().AddToFixture(),
	PrepareForTestWithJavaSdkLibraryFiles,
	dexpreopt.PrepareForTestWithDexpreopt,
)

func TestMain(m *testing.M) {
	run := func() int {
		setUp()
		defer tearDown()

		return m.Run()
	}

	os.Exit(run())
}

// testConfig is a legacy way of creating a test Config for testing java modules.
//
// See testJava for an explanation as to how to stop using this deprecated method.
//
// deprecated
func testConfig(env map[string]string, bp string, fs map[string][]byte) android.Config {
	return TestConfig(buildDir, env, bp, fs)
}

// testContext is a legacy way of creating a TestContext for testing java modules.
//
// See testJava for an explanation as to how to stop using this deprecated method.
//
// deprecated
func testContext(config android.Config) *android.TestContext {

	ctx := android.NewTestArchContext(config)
	RegisterRequiredBuildComponentsForTest(ctx)
	ctx.RegisterModuleType("java_plugin", PluginFactory)
	ctx.RegisterModuleType("filegroup", android.FileGroupFactory)
	ctx.RegisterModuleType("python_binary_host", python.PythonBinaryHostFactory)
	ctx.PreArchMutators(android.RegisterDefaultsPreArchMutators)
	ctx.PreArchMutators(android.RegisterComponentsMutator)

	ctx.PreDepsMutators(python.RegisterPythonPreDepsMutators)
	ctx.PostDepsMutators(android.RegisterOverridePostDepsMutators)
	ctx.RegisterPreSingletonType("overlay", OverlaySingletonFactory)
	ctx.RegisterPreSingletonType("sdk_versions", sdkPreSingletonFactory)

	android.RegisterPrebuiltMutators(ctx)

	genrule.RegisterGenruleBuildComponents(ctx)

	// Register module types and mutators from cc needed for JNI testing
	cc.RegisterRequiredBuildComponentsForTest(ctx)

	ctx.PostDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.TopDown("propagate_rro_enforcement", propagateRROEnforcementMutator).Parallel()
	})

	return ctx
}

// run is a legacy way of running tests of java modules.
//
// See testJava for an explanation as to how to stop using this deprecated method.
//
// deprecated
func run(t *testing.T, ctx *android.TestContext, config android.Config) {
	t.Helper()

	pathCtx := android.PathContextForTesting(config)
	dexpreopt.SetTestGlobalConfig(config, dexpreopt.GlobalConfigForTests(pathCtx))

	ctx.Register()
	_, errs := ctx.ParseBlueprintsFiles("Android.bp")
	android.FailIfErrored(t, errs)
	_, errs = ctx.PrepareBuildActions(config)
	android.FailIfErrored(t, errs)
}

// testJavaError is a legacy way of running tests of java modules that expect errors.
//
// See testJava for an explanation as to how to stop using this deprecated method.
//
// deprecated
func testJavaError(t *testing.T, pattern string, bp string) (*android.TestContext, android.Config) {
	t.Helper()
	result := javaFixtureFactory.
		Extend(dexpreopt.PrepareForTestWithDexpreopt).
		ExtendWithErrorHandler(android.FixtureExpectsAtLeastOneErrorMatchingPattern(pattern)).
		RunTestWithBp(t, bp)
	return result.TestContext, result.Config
}

// testJavaErrorWithConfig is a legacy way of running tests of java modules that expect errors.
//
// See testJava for an explanation as to how to stop using this deprecated method.
//
// deprecated
func testJavaErrorWithConfig(t *testing.T, pattern string, config android.Config) (*android.TestContext, android.Config) {
	t.Helper()
	// This must be done on the supplied config and not as part of the fixture because any changes to
	// the fixture's config will be ignored when RunTestWithConfig replaces it.
	pathCtx := android.PathContextForTesting(config)
	dexpreopt.SetTestGlobalConfig(config, dexpreopt.GlobalConfigForTests(pathCtx))
	result := javaFixtureFactory.
		ExtendWithErrorHandler(android.FixtureExpectsAtLeastOneErrorMatchingPattern(pattern)).
		RunTestWithConfig(t, config)
	return result.TestContext, result.Config
}

// runWithErrors is a legacy way of running tests of java modules that expect errors.
//
// See testJava for an explanation as to how to stop using this deprecated method.
//
// deprecated
func runWithErrors(t *testing.T, ctx *android.TestContext, config android.Config, pattern string) {
	ctx.Register()
	_, errs := ctx.ParseBlueprintsFiles("Android.bp")
	if len(errs) > 0 {
		android.FailIfNoMatchingErrors(t, pattern, errs)
		return
	}
	_, errs = ctx.PrepareBuildActions(config)
	if len(errs) > 0 {
		android.FailIfNoMatchingErrors(t, pattern, errs)
		return
	}

	t.Fatalf("missing expected error %q (0 errors are returned)", pattern)
	return
}

// testJavaWithFS runs tests using the javaFixtureFactory
//
// See testJava for an explanation as to how to stop using this deprecated method.
//
// deprecated
func testJavaWithFS(t *testing.T, bp string, fs android.MockFS) (*android.TestContext, android.Config) {
	t.Helper()
	result := javaFixtureFactory.Extend(fs.AddToFixture()).RunTestWithBp(t, bp)
	return result.TestContext, result.Config
}

// testJava runs tests using the javaFixtureFactory
//
// Do not add any new usages of this, instead use the javaFixtureFactory directly as it makes it
// much easier to customize the test behavior.
//
// If it is necessary to customize the behavior of an existing test that uses this then please first
// convert the test to using javaFixtureFactory first and then in a following change add the
// appropriate fixture preparers. Keeping the conversion change separate makes it easy to verify
// that it did not change the test behavior unexpectedly.
//
// deprecated
func testJava(t *testing.T, bp string) (*android.TestContext, android.Config) {
	t.Helper()
	result := javaFixtureFactory.RunTestWithBp(t, bp)
	return result.TestContext, result.Config
}

// testJavaWithConfig runs tests using the javaFixtureFactory
//
// See testJava for an explanation as to how to stop using this deprecated method.
//
// deprecated
func testJavaWithConfig(t *testing.T, config android.Config) (*android.TestContext, android.Config) {
	t.Helper()
	result := javaFixtureFactory.RunTestWithConfig(t, config)
	return result.TestContext, result.Config
}

func moduleToPath(name string) string {
	switch {
	case name == `""`:
		return name
	case strings.HasSuffix(name, ".jar"):
		return name
	default:
		return filepath.Join(buildDir, ".intermediates", name, "android_common", "turbine-combined", name+".jar")
	}
}

// defaultModuleToPath constructs a path to the turbine generate jar for a default test module that
// is defined in PrepareForIntegrationTestWithJava
func defaultModuleToPath(name string) string {
	return filepath.Join(buildDir, ".intermediates", defaultJavaDir, name, "android_common", "turbine-combined", name+".jar")
}
