package tradefed_modules

// Creates manifest files in out/soong/testsuites dir
// Ensure:
//    m my-team-suite -> Builds and installs all tests, or should we let `atest` do the installing.
//                    -> Creates a manifest of all modules and then atest will do the right thing for each.
//
//
import (
	"android/soong/android"
	"android/soong/tradefed"
	"encoding/json"
	"fmt"
	"sort"
)

const suiteDirectory = "testsuites"
const manifestFilePattern = "suite.%s.manifest.json"
const zipFileName = "suite.%s.zip"

func AllTestSuitesFactory() android.Singleton {
	return &allTestSuitesSingleton{}
}

func init() {
	registerAllTestSuiteBuildComponents(android.InitRegistrationContext)
}

func registerAllTestSuiteBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterParallelSingletonType("all_test_suites", AllTestSuitesFactory)
}

// suiteName is the name of test_suite rule.  It implies the name of the outputfile.
type suiteName = string

// suiteTag is name used in test_suites of tests and listed in the test_suite rule.
type suiteTag = string
type moduleName = string

type allTestSuitesSingleton struct {
	// Path where to write the manifest per suite.
	// Add new one for zip vs manifest.
	outputFiles     map[suiteName]android.OutputPath
	modulesPerSuite map[suiteTag][]moduleName
	suites          map[suiteName]testSuiteProperties
}

// Visit all modules and collect all suites and use WriteFileRuleVerbatim
// to write it out.
func (ts *allTestSuitesSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// Map of test suite name -> modules listed for that suite directly or by tag.
	ts.modulesPerSuite = make(map[suiteTag][]moduleName)
	ts.suites = make(map[suiteName]testSuiteProperties)

	// Walk all modules.
	// 1) If that module provides test suites, add the InstallFiles for that module to that test suite.
	//    (robolectric provides TestSuite interface, foo_test does BaseTestProvider)
	// 2) If module is a test_suite, add the direct modules listed.
	ctx.VisitAllModules(func(module android.Module) {
		if !module.Enabled(ctx) {
			return
		}
		name := ctx.ModuleName(module)
		// TODO(rbraunstein): Need  to implement BaseTestProviderKey for cc_test, sh_test, roblectric, ...
		if provider, ok := android.SingletonModuleProvider(ctx, module, tradefed.BaseTestProviderKey); ok {
			for _, testSuite := range provider.TestSuites {
				mps := ts.modulesPerSuite[testSuite]
				if mps == nil {
					ts.modulesPerSuite[testSuite] = []moduleName{name}
				} else {
					ts.modulesPerSuite[testSuite] = append(mps, name)
				}
			}

		}
		// If the module is our new test-suite module type, then add all of its statically declared deps as well.
		if tsm, ok := module.(*testSuiteModule); ok {
			ts.suites[name] = tsm.properties
		}
	})

	ts.outputFiles = make(map[suiteName]android.OutputPath)
	// For each named test_suite, write manifest and zip build rules
	for name, props := range ts.suites {
		ts.manifestOrZip(ctx, name, props)
	}
}

func (ts *allTestSuitesSingleton) MakeVars(ctx android.MakeVarsContext) {
	// TODO(ron): one for each suite and a .zip version
	for suite, outputFile := range ts.outputFiles {
		ctx.DistForGoal(suite+".manifest", outputFile)
	}

	// Add a dependency on the -install target for each module.

	// TODO(ron): Do we want `m MySuite` to build all the modules in /testscases/
	// or just .intermediates?
	// If /testcases/, add one of the following.
	// Otherwise, atest will build it locally when running
	// or the build will package it general-tests/device-tests.zip without
	// our forcing the build.

	for steName, _ := range ts.suites {
		for _, module := range ts.suiteModules(steName) {
			// ctx.Phony(suite, android.PathForPhony(ctx, module+"-install"))
			ctx.Phony(steName+".manifest", android.PathForPhony(ctx, module))
		}
	}
}

// Return all module names referenced by name or suite tag for a given suite.
func (ts *allTestSuitesSingleton) suiteModules(steName suiteName) []moduleName {
	// Add all directly modules
	suite := ts.suites[steName]
	suiteModules := suite.Modules.Module_names

	// And all all modules that have a tag we care about.
	for _, tag := range suite.Modules.Suite_tags {
		suiteModules = append(suiteModules, ts.modulesPerSuite[tag]...)
	}

	return suiteModules
}

// Generate the build rule for the zip.
func (ts *allTestSuitesSingleton) manifestOrZip(ctx android.SingletonContext, suite suiteName, props testSuiteProperties) {

	// For a given test_suite,
	//   For each suite tag that is part of the suite
	//      Write the module and artifacts
	outputPath := android.PathForOutput(ctx, suiteDirectory, fmt.Sprintf(manifestFilePattern, suite))
	allSuiteDeps := android.Paths{}

	// Add all directly modules
	suiteModules := props.Modules.Module_names

	// And all all modules that have a tag we care about.
	for _, tag := range props.Modules.Suite_tags {
		suiteModules = append(suiteModules, ts.modulesPerSuite[tag]...)
	}

	// Iterate in sorted order to keep output in stable order.
	sort.Strings(suiteModules)

	// Write the output to a manifest file based on test_suite module name.
	data, err := json.Marshal(suiteModules)
	if err != nil {
		ctx.Errorf("Unable to marshal suite data. %s", err)
	}

	android.WriteFileRuleVerbatim(ctx, outputPath, string(data))
	ctx.Phony(suite+".manifest", outputPath)
	ctx.Phony(suite+".manifest", allSuiteDeps...)
	ts.outputFiles[suite] = outputPath
}
