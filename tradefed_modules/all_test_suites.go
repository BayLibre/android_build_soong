package tradefed_modules

// Creates manifest files in out/soong/testsuites dir
// From hwj@
//   my-team-suite.testsuite
//      {
//        "test_suite": "my-team-suite",
//        "test_modules": {
//          "device-tests": ["MyTeamDeviceTest"],
//          "general-tests": ["MyTeamUnitTest"]
//        }
//      }
//   my-team-suite.modules
//   RON: not sure who reads this, maybe Tradefed, but tradefed shouldn't need it.
//      [
//        {
//          "test_module": "MyTeamUnitTest",
//          "artifacts": [
//            "host/testcases/MyTeamUnitTest/MyTeamUnitTest.config",
//            "host/testcases/MyTeamUnitTest/MyTeamUnitTest.apk"
//          ]
//        },
//        {
//          "test_module": "MyTeamDeviceTest",
//          "artifacts": [
//            "target/testcases/MyTeamDeviceTest/MyTeamDeviceTest.config",
//            "target/testcases/MyTeamDeviceTest/MyTeamDeviceTest.apk"
//          ]
//        }
//      ]
//
// From ron:
//  my-team-suite.manifest.json
//      {
//        "CtsDevicePolicyManagerTestCases": [
//          {
//            "src": "out/soong/.intermediates/cts/tests/signature/api/cts-current-api-gz/8224669ff1adc232d47c3865f8b0b73b/gen/current.api.gz",
//            "dest": "current.api.gz"
//          },
//          {
//      //      //            "src": "out/soong/.intermediates/cts/hostsidetests/devicepolicy/app/CertInstaller/CtsCertInstallerApp/android_common/8224669ff1adc232d47c3865f8b0b73b/CtsCertInstallerApp.apk",
//            "dest": "CtsCertInstallerApp.apk"
//          },
//
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
const manifestFileName = "suite.%s.manifest.json"
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

// Path relative to testcases install path where file should be placed in zip.
// i.e. testcases/Module1/ARCH/foo.apk
// i.e. testcases/Module1/helper.apk
// i.e. testcases/Module1/data/file.xml
type relativeFilePath = string

type allTestSuitesSingleton struct {
	// Path where to write the manifest per suite.
	// Add new one for zip vs manifest.
	outputFiles        map[suiteName]android.OutputPath
	modulesPerSuite    map[suiteTag][]moduleName
	artifactsPerModule map[moduleName]android.Paths

	// Map of suite tag -> []Module
	// suiteTags map[suiteName][]android.Module
	// Map of all testSuite modules we visit during GenerateBuildActions
	// testSuiteModules []android.Module
}

// Visit all modules and collect all suites and use WriteFileRuleVerbatim
// to write it out.
func (ts *allTestSuitesSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// Map of test suite name -> modules listed for that suite directly or by tag.
	ts.artifactsPerModule = make(map[moduleName]android.Paths)
	ts.modulesPerSuite = make(map[suiteTag][]moduleName)
	suites := make(map[string]testSuiteProperties)

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

			// If we want artifacts
			installedFiles := provider.InstalledFiles

			if provider.OutputFile != nil {
				installedFiles = append(installedFiles, provider.OutputFile)
			}
			if provider.TestConfig != nil {
				installedFiles = append(installedFiles, provider.TestConfig)
			}
			ts.artifactsPerModule[name] = installedFiles
		}
		// If the module is our new test-suite construct, then add all of its static deps as well.
		if tsm, ok := module.(*testSuiteModule); ok {
			suites[name] = tsm.properties
		}

	})

	// for each suite ....
	ts.outputFiles = make(map[suiteName]android.OutputPath)
	for name, props := range suites {
		ts.manifestOrZip(ctx, name, props)
	}
}

func (ts *allTestSuitesSingleton) MakeVars(ctx android.MakeVarsContext) {
	// TODO(ron): one for each suite and a .zip version
	for suite, outputFile := range ts.outputFiles {
		ctx.DistForGoal(suite, outputFile)
	}

	// Add a dependency on the -install target for each module.

	// TODO(ron): Do we want `m MySuite` to build all the modules in /testscases/
	// or just .intermediates?
	// If /testcases/, add one of the following.
	// Otherwise, atest will build it locally when running
	// or the build will package it general-tests/device-tests.zip without
	// our forcing the build.
	forceInstallAtBuildTime := true
	if forceInstallAtBuildTime {
		for suite, moduleList := range ts.modulesPerSuite {
			for _, module := range moduleList {
				// ctx.Phony(suite, android.PathForPhony(ctx, module+"-install"))
				ctx.Phony(suite, android.PathForPhony(ctx, module))
			}
		}
	}
}

// TODO(ron): cleanup once we decide if we need both src and dest in manifest or not.
/*
	type packagingInfo struct {
		// The `Path` converted to a string for the input.  Generally an intermediate file.
		Src string `json:"src"`
		// Relative pathname under testcases where the file should go.
		Dest string `json:"dest"`
	        }
*/
type packagingInfo = string

// generate the build rule for the zip.
func (ts *allTestSuitesSingleton) manifestOrZip(ctx android.SingletonContext, suite suiteName, props testSuiteProperties) {

	// For a given test_suite,
	//   For each suite tag that is part of the suite
	//      Write the module and artifacts
	fmt.Printf("MAN: %s, %+v\n", suite, props.Modules.Suite_tags)
	outputPath := android.PathForOutput(ctx, suiteDirectory, fmt.Sprintf(manifestFileName, suite))
	packaging := map[moduleName][]packagingInfo{}
	allSuiteDeps := android.Paths{}
	suiteModules := props.Modules.Module_names

	for _, tag := range props.Modules.Suite_tags {
		// TODO(ron): create a second output file ...

		// convert the .intermediate paths for the apks, datafiles, etc to the relative path that they
		// should look like in the testcases dir.
		// This should already exist in paths.go, but I can't find one to use for the singleton context.

		suiteModules = append(suiteModules, ts.modulesPerSuite[tag]...)
	}
	// Iterate in sorted order to keep output in stable order.
	sort.Strings(suiteModules)

	// TODO(ron): sort the output file, iterate in order.
	for _, moduleName := range suiteModules {
		artifactPaths := make([]packagingInfo, len(ts.artifactsPerModule[moduleName]))
		allSuiteDeps = append(allSuiteDeps, ts.artifactsPerModule[moduleName]...)
		for i, dep := range ts.artifactsPerModule[moduleName] {
			/*
				relModulePaths[i].Src = dep.String()
				relModulePaths[i].Dest = dep.Rel()
			*/
			// TODO(ron): make canonical, trying OutputPath.RelativeToTop()
			// or create a new method under testing.go to do it.
			artifactPaths[i] = dep.Rel()
		}
		packaging[moduleName] = artifactPaths
	}

	// Write the output to a manifest file based on test_suite module name.
	data, err := json.Marshal(packaging)
	if err != nil {
		ctx.Errorf("Unable to marshal suite data. %s", err)
	}

	android.WriteFileRuleVerbatim(ctx, outputPath, string(data))
	ctx.Phony(suite, outputPath)
	ctx.Phony(suite, allSuiteDeps...)
	ts.outputFiles[suite] = outputPath
}
