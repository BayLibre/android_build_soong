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

type suiteName = string
type moduleName = string

// Path relative to testcases install path where file should be placed in zip.
// i.e. testcases/Module1/ARCH/foo.apk
// i.e. testcases/Module1/helper.apk
// i.e. testcases/Module1/data/file.xml
type relativeFilePath = string

type allTestSuitesSingleton struct {
	// Path where to write the manifest per suite.
	// Add new one for zip vs manifest.
	outputFiles     map[suiteName]android.OutputPath
	modulesPerSuite map[suiteName][]moduleName

	// Map of suite tag -> []Module
	// suiteTags map[suiteName][]android.Module
	// Map of all testSuite modules we visit during GenerateBuildActions
	// testSuiteModules []android.Module
}

// Visit all modules and collect all suites and use WriteFileRuleVerbatim
// to write it out.
func (ts *allTestSuitesSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// Map of test suite name -> modules listed for that suite directly or by tag.
	files := make(map[suiteName]map[moduleName]android.Paths)
	ts.modulesPerSuite = make(map[suiteName][]moduleName)

	// Walk all modules.
	// 1) If that module provides test suites, add the InstallFiles for that module to that test suite.
	//    (robolectric provides TestSuite interface, foo_test does BaseTestProvider)
	// 2) If module is a test_suite, add the direct modules listed.
	ctx.VisitAllModules(func(module android.Module) {
		if !module.Enabled(ctx) {
			return
		}
		name := ctx.ModuleName(module)
		// TODO(rbraunstein): Need  to implement BaseTestProviderKey for cc_test, sh_test
		if provider, ok := android.SingletonModuleProvider(ctx, module, tradefed.BaseTestProviderKey); ok {
			// fmt.Printf("RON WAS ERE:%v \n", module)

			for _, testSuite := range provider.TestSuites {
				if files[testSuite] == nil {
					files[testSuite] = make(map[string]android.Paths)
				}

				mps := ts.modulesPerSuite[testSuite]
				if mps == nil {
					ts.modulesPerSuite[testSuite] = []moduleName{name}
				} else {
					ts.modulesPerSuite[testSuite] = append(mps, name)
				}

				// TODO(ron): same question as tmc,
				// Just zip the the dir or name all the files under the dir and zip those?
				// We are missing helper apps.
				//   If it is just the dir, how do we ensure the complete package got built? (-host, -target, etc)
				// was tsm.FilesToInstall
				/*
					   Pacakging spec might be better, but still is missing the HelperApps.
					   So would need to add more to the provider.
					for _, p := range provider.InstalledFiles.PackagingSpecs() {
						fmt.Printf("\t%s, %s\n", name, p.RelPathInPackage())
					}
					for _, p := range provider.InstalledFiles.TransitivePackagingSpecs() {
						fmt.Printf("\t* %s, %s\n", name, p.RelPathInPackage())
					}
				*/

				// TODO(ron): use both TestConfig provider and TestSuiteModule to get list and installDir
				// fmt.Printf(" *\t%s, %v\n", name, provider.InstalledFiles.RelativeToTop())
				// fmt.Printf(" *\t%+v\n", provider)
				// fmt.Printf(" *\t%+v\n", android.pathForInstall(ctx, ctx.Config().BuildOS, ctx.Config().BuildArch, "MEME"))
				// relInstalledFiles := []string{}
				installedFiles := provider.InstalledFiles

				// NOTE: The provider is giving us the .intermediate path for these files, not the installed path in testcases.
				// We use the .intermediate path as our dependency for phony and the as the src for the manifest,
				// but we compute the what the installed path would be.
				// Ideally, there would be a function that we could use from the singleton context or the provider would give it
				// to us; something similar to android.Paths.RelativeToTop().
				// The install into testcases during build and current suite creation happens in .mk files so path logic is there.
				///for _, f := range provider.InstalledFiles {
				fmt.Printf("\t * %s, %s %v \n", name, provider.TestcaseDir.Rel(), provider.TestcaseDir)
				// relInstalledFiles = append(relInstalledFiles, f.Rel())
				///}
				// TODO(ron): fix output file to have arch?
				if provider.OutputFile != nil {
					installedFiles = append(installedFiles, provider.OutputFile)
				}
				if provider.TestConfig != nil {
					installedFiles = append(installedFiles, provider.TestConfig)
				}
				files[testSuite][name] = installedFiles

				// files[testSuite][name] = append(files[testSuite][name], provider.InstalledFiles.RelativeToTop()...)
			}
		}
		// For robolectric, ravenwood, just make those providers for BaseTestProviderKey too
		/*
			if tsm, ok := module.(android.TestSuiteModule); ok {
				fmt.Printf("\tPS %s, %v %v\n", name, tsm.FilesToInstall().android.Paths().RelativeToTop(), tsm.FilesToInstall().android.Paths()[0].Rel())

			}
		*/
		// If the module is our new test-suite construct, then add all of its static deps as well.

	})

	// for each suite ....
	ts.outputFiles = make(map[suiteName]android.OutputPath)
	ts.manifestOrZip(ctx, files)
}

func (ts *allTestSuitesSingleton) MakeVars(ctx android.MakeVarsContext) {
	// TODO(ron): one for each suite and a .zip version
	for suite, outputFile := range ts.outputFiles {
		ctx.DistForGoal(suite, outputFile)
	}

	fmt.Printf("MODULE LIST: %v\n", ts.modulesPerSuite)
	// Add a dependency on the -install target for each module.

	// TODO(ron): Do we want `m MySuite` to build all the modules in /testscases/
	// or just .intermediates?
	// If /testcases/, add one of the following.
	// Otherwise, atest will build it locally when running
	// or the build will package it general-tests/device-tests.zip without
	// our forcing the build.
	forceInstallAtBuildTime := false
	if forceInstallAtBuildTime {
		for suite, moduleList := range ts.modulesPerSuite {
			for _, module := range moduleList {
				// ctx.Phony(suite, android.PathForPhony(ctx, module+"-install"))
				ctx.Phony(suite, android.PathForPhony(ctx, module))
			}
		}
	}
}

type packagingInfo struct {
	// The `Path` converted to a string for the input.  Generally an intermediate file.
	Src string `json:"src"`
	// Relative pathname under testcases where the file should go.
	Dest string `json:"dest"`
}

// generate the build rule for the zip.
func (ts *allTestSuitesSingleton) manifestOrZip(ctx android.SingletonContext, files map[suiteName]map[moduleName]android.Paths) {

	for suite, modules := range files {
		fmt.Printf("MAN: %s, %+v\n", suite, modules)
		// TODO(ron): sort the output file, iterate in order.
		outputPath := android.PathForOutput(ctx, suiteDirectory, fmt.Sprintf(manifestFileName, suite))

		// convert the .intermediate paths for the apks, datafiles, etc to the relative path that they
		// should look like in the testcases dir.
		// This should already exist in paths.go, but I can't find one to use for the singleton context.
		packaging := map[moduleName][]packagingInfo{}
		allSuiteDeps := android.Paths{}

		// iterate in sorted order to keep output in stable order.
		for _, moduleName := range android.SortedKeys(modules) {
			allSuiteDeps = append(allSuiteDeps, modules[moduleName]...)
			relModulePaths := make([]packagingInfo, len(modules[moduleName]))
			for i, dep := range modules[moduleName] {
				relModulePaths[i].Src = dep.String()
				relModulePaths[i].Dest = dep.Rel()
			}
			packaging[moduleName] = relModulePaths
		}

		data, err := json.Marshal(packaging)
		if err != nil {
			ctx.Errorf("Unable to marshal suite data. %s", err)
		}

		android.WriteFileRuleVerbatim(ctx, outputPath, string(data))
		// fmt.Printf("DATA: %s -- %s\n", suite, string(data))
		// TODO(RON): Add the dependencies!!
		// The trick is this is coming from .intermediates dir, not installed dir.
		// Need to get a path in the other module installed dir.
		// Outputfile Isn't even there. TestCase is now?
		ctx.Phony(suite, outputPath)
		// Ideally, depend on the dep in the -install directory, not the .intermediates dir..
		// Via:
		//    1) fix the provider to give this to us
		//    2) compute the install tree per module ourselves if we can.
		//    3) Rely on the fact that we depend on -install target for the module and
		//       just zip that dir rather than list all files.
		// testcases directory, not depend on .intermediates.
		ctx.Phony(suite, allSuiteDeps...)
		ts.outputFiles[suite] = outputPath
	}
}
