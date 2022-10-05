// Copyright 2022 Google Inc. All rights reserved.
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
	"fmt"
	"strings"
	"testing"

	"android/soong/android"
	"android/soong/cc/config"
)

func TestTidyFlagsWarningsAsErrors(t *testing.T) {
	// The "tidy_flags" property should not contain -warnings-as-errors.
	type testCase struct {
		libName, bp string
		errorMsg    string   // a negative test; must have error message
		flags       []string // must have substrings in tidyFlags
		noFlags     []string // must not have substrings in tidyFlags
	}

	testCases := []testCase{
		{
			"libfoo1",
			`cc_library_shared { // no warnings-as-errors, good tidy_flags
			  name: "libfoo1",
			  srcs: ["foo.c"],
              tidy_flags: ["-header-filter=dir1/"],
		    }`,
			"",
			[]string{"-header-filter=dir1/"},
			[]string{"-warnings-as-errors"},
		},
		{
			"libfoo2",
			`cc_library_shared { // good use of tidy_checks_as_errors
			  name: "libfoo2",
			  srcs: ["foo.c"],
			  tidy_checks_as_errors: ["xyz-*", "abc"],
		    }`,
			"",
			[]string{
				"-header-filter=^", // there is a default header filter
				"-warnings-as-errors='xyz-*',abc,${config.TidyGlobalNoErrorChecks}",
			},
			[]string{},
		},
	}
	if NoWarningsAsErrorsInTidyFlags {
		testCases = append(testCases, testCase{
			"libfoo3",
			`cc_library_shared { // bad use of -warnings-as-errors in tidy_flags
					  name: "libfoo3",
					  srcs: ["foo.c"],
		              tidy_flags: [
		                "-header-filters=.*",
					    "-warnings-as-errors=xyz-*",
		              ],
				    }`,
			`module "libfoo3" .*: tidy_flags: should not contain .*;` +
				` use tidy_checks_as_errors instead`,
			[]string{},
			[]string{},
		})
	}
	for _, test := range testCases {
		if test.errorMsg != "" {
			testCcError(t, test.errorMsg, test.bp)
			continue
		}
		variant := "android_arm64_armv8-a_shared"
		ctx := testCc(t, test.bp)
		t.Run("caseTidyFlags", func(t *testing.T) {
			flags := ctx.ModuleForTests(test.libName, variant).Rule("clangTidy").Args["tidyFlags"]
			for _, flag := range test.flags {
				if !strings.Contains(flags, flag) {
					t.Errorf("tidyFlags %v for %s does not contain %s.", flags, test.libName, flag)
				}
			}
			for _, flag := range test.noFlags {
				if strings.Contains(flags, flag) {
					t.Errorf("tidyFlags %v for %s should not contain %s.", flags, test.libName, flag)
				}
			}
		})
	}
}

func TestTidyConfigFile(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name       string         // test lib name
		bp         string         // a root .bp file, or empty
		errorMsg   string         // a negative test; must have error message
		configFile string         // expected in -config-file= flag
		depFiles   []string       // expected implicit input/dependent files
		mockFS     android.MockFS // mocked files
		// To support InheritParentConfig, multiple .clang-tidy files will be added
		// as dependent files.
	}
	noDepFile := []string{}
	noMockFile := android.MockFS{}
	testCases := []testCase{
		{ // no error, no expected config/dependent files
			"libt1",
			`cc_library_shared { name: "libt1", srcs: ["t1.c"] }`,
			"", "", noDepFile, noMockFile,
		},
		{ // config file path is relative to source root
			"libt2",
			`cc_library_shared {
			  name: "libt2",
			  srcs: ["t2.c"],
			  tidy_config_file: "d1/d2/myconfig",
			}`,
			"", "d1/d2/myconfig", []string{"d1/d2/myconfig"},
			android.MockFS{"d1/d2/myconfig": []byte("")},
		},
		{ // --config-file flag should not be used
			"libt3",
			`cc_library_shared { name: "libt3", srcs: ["t3.c"], tidy_flags: ["--config-file=myconfig"] }`,
			"--config-file=myconfig should be replaced with the tidy_config_file property",
			"", noDepFile, noMockFile,
		},
		{ // -config-file flag should not be used
			"libt4",
			`cc_library_shared { name: "libt4", srcs: ["t4.c"], tidy_flags: ["-any-flag", "-config-file=xyz"] }`,
			"-config-file=xyz should be replaced with the tidy_config_file property",
			"", noDepFile, noMockFile,
		},
		{ // tidy_config_file overrides .clang-tidy, but src dir's .clang-tidy can be inherited
			"libt5",
			`cc_library_shared { name: "libt5", srcs: ["t5.c"], tidy_config_file: "d1/myconfig" }`,
			"", "d1/myconfig", []string{"d1/myconfig", ".clang-tidy"},
			android.MockFS{
				"d1/myconfig":    []byte(""),
				"d1/.clang-tidy": []byte(""),
				".clang-tidy":    []byte(""),
			},
		},
		{ // missing tidy_config_file
			"libt6",
			`cc_library_shared { name: "libt6", srcs: ["t6.c"], tidy_config_file: "d3/myconfig3" }`,
			"Can't find config file: d3/myconfig3", "", noDepFile, noMockFile,
		},
		{ // multiple .clang-tidy in parent directories
			"libt7",
			`cc_library_shared { name: "libt7", srcs: ["d1/d2/d3/t7.c"] }`,
			"", "", []string{"d1/d2/d3/.clang-tidy", "d1/.clang-tidy"},
			android.MockFS{
				"d1/.clang-tidy":       []byte(""),
				"d1/d2/other.tidy":     []byte(""), // not .clang-tidy
				"d1/d2/d3/.clang-tidy": []byte(""),
			},
		},
		{ // default .clang-tidy in one parent directory
			"libt8",
			`cc_library_shared { name: "libt8", srcs: ["d1/d2/d3/t8.c"] }`,
			"", "", []string{"d1/.clang-tidy"},
			android.MockFS{"d1/.clang-tidy": []byte("")},
		},
		{ // has .clang-tidy in srcs parent dirs; do not use those in .bp dirs
			"libt9",
			"", // use d1/d2/d3/d4/Androird.bp
			"", // no error
			"", // no tidy_config_file
			[]string{"s/d1/d2/.clang-tidy", "s/d1/.clang-tidy"},
			android.MockFS{
				"s/d1/.clang-tidy":       []byte(""),
				"s/d1/d2/.clang-tidy":    []byte(""),
				"s/Android.bp":           []byte(`filegroup { name: "fg", srcs: ["d1/d2/d3/t9.c"] }`),
				"d1/.clang-tidy":         []byte(""),
				"d1/d2/d3/.clang-tidy":   []byte(""),
				"d1/d2/d3/d4/Android.bp": []byte(`cc_library_shared { name: "libt9", srcs: [":fg"] }`),
			},
		},
		{ // default .clang-tidy in .bp directory, not in srcs directory.
			"libt10",
			"", // use d1/d2/d3/d4/Androird.bp
			"", // no error
			"d1/d2/d3/.clang-tidy",
			[]string{"d1/d2/d3/.clang-tidy", "d1/.clang-tidy"},
			android.MockFS{
				"s/Android.bp":           []byte(`filegroup { name: "fg", srcs: ["d1/d2/d3/t10.c"] }`),
				"d1/.clang-tidy":         []byte(""),
				"d1/d2/d3/.clang-tidy":   []byte(""),
				"d1/d2/d3/d4/Android.bp": []byte(`cc_library_shared { name: "libt10", srcs: [":fg"] }`),
			},
		},
		{ // source root .clang-tidy is also recognized, but normal builds do not have it.
			"libt11",
			"", // use d1/Androird.bp
			"", // no error
			"", // no explicit tidy_config_file
			[]string{"d1/.clang-tidy", ".clang-tidy"},
			android.MockFS{
				".clang-tidy":    []byte(""), // adding a file to root directory can affect other tests?
				"d1/.clang-tidy": []byte(""), // okay
				"d1/Android.bp":  []byte(`cc_library_shared { name: "libt11", srcs: ["t11.c"] }`),
			},
		},
	}
	for _, test := range testCases {
		if test.errorMsg != "" {
			testCcError(t, test.errorMsg, test.bp)
			continue
		}
		variant := "android_arm64_armv8-a_shared"
		t.Run(test.name, func(t *testing.T) {
			// each test is run once with RBE and once without
			for _, rbe := range []string{"true", "false"} {
				testEnv := map[string]string{}
				testEnv["USE_RBE"] = rbe
				testEnv["RBE_CLANG_TIDY"] = "true"
				// prepare mockedClangTidyDirs for the search of .clang-tidy
				mockedClangTidyDirs := make(map[string]bool)
				for file, _ := range test.mockFS {
					if strings.HasSuffix(file, ".clang-tidy") {
						dir := GetClangTidyFileDir(file)
						mockedClangTidyDirs[dir] = true
					}
				}
				myFixClangTidyDirs := func(config android.Config, ctx *android.TestContext) {
					ClangTidyDirs = func(_ android.Config) map[string]bool {
						return mockedClangTidyDirs
					}
				}
				// add mocked files and environment variables
				preparers := android.GroupFixturePreparers(prepareForCcTest,
					test.mockFS.AddToFixture(),
					android.FixtureModifyConfigAndContext(myFixClangTidyDirs),
					android.FixtureMergeEnv(testEnv))
				var ctx *android.TestResult
				if test.bp == "" {
					ctx = preparers.RunTest(t)
				} else {
					ctx = preparers.RunTestWithBp(t, test.bp)
				}

				tidyRule := ctx.ModuleForTests(test.name, variant).Rule("clangTidy")
				flags := tidyRule.Args["tidyFlags"]
				// check ninja rule implicit dependent files
				if len(test.depFiles) > 0 {
					// should not have the default global checks
					android.AssertStringDoesNotContain(t, "unexpected tidy checks",
						tidyRule.Args["tidyFlags"], config.TidyDefaultGlobalChecks())
					for _, depFile := range test.depFiles {
						android.AssertStringListContains(t, "missing dependency in clangTidy rule",
							tidyRule.Implicits.Strings(), depFile)
					}
				} else {
					android.AssertStringDoesContain(t, "missing tidy checks",
						tidyRule.Args["tidyFlags"], config.TidyDefaultGlobalChecks())
					if len(tidyRule.Implicits.Strings()) > 0 {
						t.Errorf("%s has unexpected dependency: %s.", test.name, tidyRule.Implicits.Strings())
					}
				}
				// check RBE implicit inputs
				if rbe == "true" {
					// "implicitInputs" is set only when RBE is enabled
					expected := strings.Join(test.depFiles, ",")
					android.AssertStringEquals(t, test.name+" rbe implicitInputs", expected, tidyRule.Args["implicitInputs"])
				} else {
					android.AssertStringEquals(t, test.name+" no rbe implicitInputs", "", tidyRule.Args["implicitInputs"])
				}
				// check the -config-file= flag
				android.AssertStringEquals(t, test.name+" -config-file", test.configFile, FindTidyConfigFileInFlags(flags))
			}
		})
	}
}

func TestTidyChecks(t *testing.T) {
	// The "tidy_checks" property defines additional checks appended
	// to global default. But there are some checks disabled after
	// the local tidy_checks.
	bp := `
		cc_library_shared { // has global checks + extraGlobalChecks
			name: "libfoo_1",
			srcs: ["foo.c"],
		}
		cc_library_shared { // has only local checks + extraGlobalChecks
			name: "libfoo_2",
			srcs: ["foo.c"],
			tidy_checks: ["-*", "xyz-*"],
		}
		cc_library_shared { // has global checks + local checks + extraGlobalChecks
			name: "libfoo_3",
			srcs: ["foo.c"],
			tidy_checks: ["-abc*", "xyz-*", "mycheck"],
		}
		cc_library_shared { // has only local checks after "-*" + extraGlobalChecks
			name: "libfoo_4",
			srcs: ["foo.c"],
			tidy_checks: ["-abc*", "xyz-*", "mycheck", "-*", "xyz-*"],
		}`
	ctx := testCc(t, bp)

	globalChecks := "-checks=${config.TidyDefaultGlobalChecks},"
	firstXyzChecks := "-checks='-*','xyz-*',"
	localXyzChecks := "'-*','xyz-*'"
	localAbcChecks := "'-abc*','xyz-*',mycheck"
	extraGlobalChecks := ",${config.TidyGlobalNoChecks}"
	testCases := []struct {
		libNumber int      // 1,2,3,...
		checks    []string // must have substrings in -checks
		noChecks  []string // must not have substrings in -checks
	}{
		{1, []string{globalChecks, extraGlobalChecks}, []string{localXyzChecks, localAbcChecks}},
		{2, []string{firstXyzChecks, extraGlobalChecks}, []string{globalChecks, localAbcChecks}},
		{3, []string{globalChecks, localAbcChecks, extraGlobalChecks}, []string{localXyzChecks}},
		{4, []string{firstXyzChecks, extraGlobalChecks}, []string{globalChecks, localAbcChecks}},
	}
	t.Run("caseTidyChecks", func(t *testing.T) {
		variant := "android_arm64_armv8-a_shared"
		for _, test := range testCases {
			libName := fmt.Sprintf("libfoo_%d", test.libNumber)
			flags := ctx.ModuleForTests(libName, variant).Rule("clangTidy").Args["tidyFlags"]
			splitFlags := strings.Split(flags, " ")
			foundCheckFlag := false
			for _, flag := range splitFlags {
				if strings.HasPrefix(flag, "-checks=") {
					foundCheckFlag = true
					for _, check := range test.checks {
						if !strings.Contains(flag, check) {
							t.Errorf("tidyFlags for %s does not contain %s.", libName, check)
						}
					}
					for _, check := range test.noChecks {
						if strings.Contains(flag, check) {
							t.Errorf("tidyFlags for %s should not contain %s.", libName, check)
						}
					}
					break
				}
			}
			if !foundCheckFlag {
				t.Errorf("tidyFlags for %s does not contain -checks=.", libName)
			}
		}
	})
}

func TestWithTidy(t *testing.T) {
	// When WITH_TIDY=1 or (ALLOW_LOCAL_TIDY_TRUE=1 and local tidy:true)
	// a C++ library should depend on .tidy files.
	testCases := []struct {
		withTidy, allowLocalTidyTrue string // "_" means undefined
		needTidyFile                 []bool // for {libfoo_0, libfoo_1} and {libbar_0, libbar_1}
	}{
		{"_", "_", []bool{false, false, false}},
		{"_", "0", []bool{false, false, false}},
		{"_", "1", []bool{false, true, false}},
		{"_", "true", []bool{false, true, false}},
		{"0", "_", []bool{false, false, false}},
		{"0", "1", []bool{false, true, false}},
		{"1", "_", []bool{true, true, false}},
		{"1", "false", []bool{true, true, false}},
		{"1", "1", []bool{true, true, false}},
		{"true", "_", []bool{true, true, false}},
	}
	bp := `
		cc_library_shared {
			name: "libfoo_0", // depends on .tidy if WITH_TIDY=1
			srcs: ["foo.c"],
		}
		cc_library_shared { // depends on .tidy if WITH_TIDY=1 or ALLOW_LOCAL_TIDY_TRUE=1
			name: "libfoo_1",
			srcs: ["foo.c"],
			tidy: true,
		}
		cc_library_shared { // no .tidy
			name: "libfoo_2",
			srcs: ["foo.c"],
			tidy: false,
		}
		cc_library_static {
			name: "libbar_0", // depends on .tidy if WITH_TIDY=1
			srcs: ["bar.c"],
		}
		cc_library_static { // depends on .tidy if WITH_TIDY=1 or ALLOW_LOCAL_TIDY_TRUE=1
			name: "libbar_1",
			srcs: ["bar.c"],
			tidy: true,
		}
		cc_library_static { // no .tidy
			name: "libbar_2",
			srcs: ["bar.c"],
			tidy: false,
		}`
	for index, test := range testCases {
		testName := fmt.Sprintf("case%d,%v,%v", index, test.withTidy, test.allowLocalTidyTrue)
		t.Run(testName, func(t *testing.T) {
			testEnv := map[string]string{}
			if test.withTidy != "_" {
				testEnv["WITH_TIDY"] = test.withTidy
			}
			if test.allowLocalTidyTrue != "_" {
				testEnv["ALLOW_LOCAL_TIDY_TRUE"] = test.allowLocalTidyTrue
			}
			ctx := android.GroupFixturePreparers(prepareForCcTest, android.FixtureMergeEnv(testEnv)).RunTestWithBp(t, bp)
			for n := 0; n < 3; n++ {
				checkLibraryRule := func(foo, variant, ruleName string) {
					libName := fmt.Sprintf("lib%s_%d", foo, n)
					tidyFile := "out/soong/.intermediates/" + libName + "/" + variant + "/obj/" + foo + ".tidy"
					depFiles := ctx.ModuleForTests(libName, variant).Rule(ruleName).Validations.Strings()
					if test.needTidyFile[n] {
						android.AssertStringListContains(t, libName+" needs .tidy file", depFiles, tidyFile)
					} else {
						android.AssertStringListDoesNotContain(t, libName+" does not need .tidy file", depFiles, tidyFile)
					}
				}
				checkLibraryRule("foo", "android_arm64_armv8-a_shared", "ld")
				checkLibraryRule("bar", "android_arm64_armv8-a_static", "ar")
			}
		})
	}
}
