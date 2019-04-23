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

package build

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"android/soong/ui/logger"
	"android/soong/ui/terminal"
)

func testContext() Context {
	return Context{&ContextImpl{
		Context: context.Background(),
		Logger:  logger.New(&bytes.Buffer{}),
		Writer:  terminal.NewWriter(terminal.NewCustomStdio(&bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{})),
	}}
}

func TestConfigParseArgsJK(t *testing.T) {
	ctx := testContext()

	testCases := []struct {
		args []string

		parallel  int
		keepGoing int
		remaining []string
	}{
		{nil, -1, -1, nil},

		{[]string{"-j"}, -1, -1, nil},
		{[]string{"-j1"}, 1, -1, nil},
		{[]string{"-j1234"}, 1234, -1, nil},

		{[]string{"-j", "1"}, 1, -1, nil},
		{[]string{"-j", "1234"}, 1234, -1, nil},
		{[]string{"-j", "1234", "abc"}, 1234, -1, []string{"abc"}},
		{[]string{"-j", "abc"}, -1, -1, []string{"abc"}},
		{[]string{"-j", "1abc"}, -1, -1, []string{"1abc"}},

		{[]string{"-k"}, -1, 0, nil},
		{[]string{"-k0"}, -1, 0, nil},
		{[]string{"-k1"}, -1, 1, nil},
		{[]string{"-k1234"}, -1, 1234, nil},

		{[]string{"-k", "0"}, -1, 0, nil},
		{[]string{"-k", "1"}, -1, 1, nil},
		{[]string{"-k", "1234"}, -1, 1234, nil},
		{[]string{"-k", "1234", "abc"}, -1, 1234, []string{"abc"}},
		{[]string{"-k", "abc"}, -1, 0, []string{"abc"}},
		{[]string{"-k", "1abc"}, -1, 0, []string{"1abc"}},

		// TODO: These are supported in Make, should we support them?
		//{[]string{"-kj"}, -1, 0},
		//{[]string{"-kj8"}, 8, 0},

		// -jk is not valid in Make
	}

	for _, tc := range testCases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			defer logger.Recover(func(err error) {
				t.Fatal(err)
			})

			c := &configImpl{
				parallel:  -1,
				keepGoing: -1,
			}
			c.parseArgs(ctx, tc.args)

			if c.parallel != tc.parallel {
				t.Errorf("for %q, parallel:\nwant: %d\n got: %d\n",
					strings.Join(tc.args, " "),
					tc.parallel, c.parallel)
			}
			if c.keepGoing != tc.keepGoing {
				t.Errorf("for %q, keep going:\nwant: %d\n got: %d\n",
					strings.Join(tc.args, " "),
					tc.keepGoing, c.keepGoing)
			}
			if !reflect.DeepEqual(c.arguments, tc.remaining) {
				t.Errorf("for %q, remaining arguments:\nwant: %q\n got: %q\n",
					strings.Join(tc.args, " "),
					tc.remaining, c.arguments)
			}
		})
	}
}

func TestConfigParseArgsVars(t *testing.T) {
	ctx := testContext()

	testCases := []struct {
		env  []string
		args []string

		expectedEnv []string
		remaining   []string
	}{
		{},
		{
			env: []string{"A=bc"},

			expectedEnv: []string{"A=bc"},
		},
		{
			args: []string{"abc"},

			remaining: []string{"abc"},
		},

		{
			args: []string{"A=bc"},

			expectedEnv: []string{"A=bc"},
		},
		{
			env:  []string{"A=a"},
			args: []string{"A=bc"},

			expectedEnv: []string{"A=bc"},
		},

		{
			env:  []string{"A=a"},
			args: []string{"A=", "=b"},

			expectedEnv: []string{"A="},
			remaining:   []string{"=b"},
		},
	}

	for _, tc := range testCases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			defer logger.Recover(func(err error) {
				t.Fatal(err)
			})

			e := Environment(tc.env)
			c := &configImpl{
				environ: &e,
			}
			c.parseArgs(ctx, tc.args)

			if !reflect.DeepEqual([]string(*c.environ), tc.expectedEnv) {
				t.Errorf("for env=%q args=%q, environment:\nwant: %q\n got: %q\n",
					tc.env, tc.args,
					tc.expectedEnv, []string(*c.environ))
			}
			if !reflect.DeepEqual(c.arguments, tc.remaining) {
				t.Errorf("for env=%q args=%q, remaining arguments:\nwant: %q\n got: %q\n",
					tc.env, tc.args,
					tc.remaining, c.arguments)
			}
		})
	}
}

func TestConfigCheckTopDir(t *testing.T) {
	ctx := testContext()
	buildRootDir := filepath.Dir(srcDirFileCheck)

	tests := []struct {
		description         string
		path                string
		wantErr             bool
		createBuildRootFile bool
	}{{
		description:         "already at the root source tree",
		createBuildRootFile: true,
	}, {
		description:         "one level deep in the source tree",
		path:                "1",
		createBuildRootFile: true,
		wantErr:             true,
	}, {
		description:         "deep in the source tree",
		createBuildRootFile: true,
		path:                "1/2/3/4/5/6/7/8/9/1/2/3/4/5/6/7/8/9/1/2/3/4/5/6/7/8/9/1/2/3/4/5/6/7",
		wantErr:             true,
	}, {
		description: "not in source tree",
		path:        "1/2/3/4/5",
		wantErr:     true,
	}}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			defer logger.Recover(func(err error) {
				if !tt.wantErr {
					t.Fatalf("Got unexpected error: %v", err)
				}
			})

			// Create the root source tree.
			rootDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(rootDir)

			// Create the build root file. This is to test if topDir returns an error if the build root
			// file does not exist.
			if tt.createBuildRootFile {
				dir := filepath.Join(rootDir, buildRootDir)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Errorf("failed to create %s directory: %v", dir, err)
				}
				f := filepath.Join(rootDir, srcDirFileCheck)
				if err := ioutil.WriteFile(f, []byte{}, 0644); err != nil {
					t.Errorf("failed to create file %s: %v", f, err)
				}
			}

			// Next block of code is to set the current directory.
			dir := rootDir
			if tt.path != "" {
				dir = filepath.Join(dir, tt.path)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Errorf("failed to create %s directory: %v", dir, err)
				}
			}
			curDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get the current directory: %v", err)
			}
			defer func() { os.Chdir(curDir) }()

			if err := os.Chdir(dir); err != nil {
				t.Fatalf("failed to change directory to %s: %v", dir, err)
			}

			checkTopDir(ctx)
		})
	}
}

func TestConfigConvertToTarget(t *testing.T) {
	tests := []struct {
		description    string
		dir            string
		prefix         string
		expectedTarget string
	}{{
		description:    "one level directory in source tree",
		dir:            "test1",
		prefix:         "MODULES-IN-",
		expectedTarget: "MODULES-IN-test1",
	}, {
		description:    "multiple level directories in source tree",
		dir:            "test1/test2/test3/test4",
		prefix:         "GET-INSTALL-PATH-IN-",
		expectedTarget: "GET-INSTALL-PATH-IN-test1-test2-test3-test4",
	}}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			target := convertToTarget(tt.dir, tt.prefix)
			if target != tt.expectedTarget {
				t.Errorf("expected %s, got %s for target", tt.expectedTarget, target)
			}
		})
	}
}

func setTop(t *testing.T, dir string) func() {
	curDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change directory to top dir %s: %v", dir, err)
	}
	return func() { os.Chdir(curDir) }
}

func createBuildFiles(t *testing.T, topDir string, buildFiles []string) {
	for _, buildFile := range buildFiles {
		buildFile = filepath.Join(topDir, buildFile)
		if err := ioutil.WriteFile(buildFile, []byte{}, 0644); err != nil {
			t.Errorf("failed to create file %s: %v", buildFile, err)
		}
	}
}

func createDirectories(t *testing.T, topDir string, dirs []string) {
	for _, dir := range dirs {
		dir = filepath.Join(topDir, dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Errorf("failed to create %s directory: %v", dir, err)
		}
	}
}

func TestConfigGetTargets(t *testing.T) {
	ctx := testContext()
	tests := []struct {
		// Test description.
		description string

		// Directories passed in to soong_ui.
		dirs []string

		// Directories to be created in order to get the targets.
		createDirs []string

		// Build files to be created to validate if they exist.
		createBuildFiles []string

		// Current directory that the user executed the build action command.
		curDir string

		// The target prefix name.
		targetPrefixName string

		// Expected targets from the function.
		expectedTargets []string

		// Expected build from the build system.
		expectedBuildFiles []string

		// Expecting error from running test case.
		wantErr bool
	}{{
		description:        "one target dir specified",
		dirs:               []string{"1/2/3"},
		createDirs:         []string{"0/1/2/3"},
		createBuildFiles:   []string{"0/1/2/3/Android.bp"},
		targetPrefixName:   "MODULES-IN-",
		curDir:             "0",
		expectedTargets:    []string{"MODULES-IN-0-1-2-3"},
		expectedBuildFiles: []string{"0/1/2/3/Android.mk"},
	}, {
		description:      "one target dir specified, build file does not exist",
		dirs:             []string{"1/2/3"},
		createDirs:       []string{"0/1/2/3"},
		targetPrefixName: "MODULES-IN-",
		curDir:           "test0",
		wantErr:          true,
	}, {
		description:      "one target dir specified, invalid targets specified",
		dirs:             []string{"1/2/3:t1:t2"},
		createDirs:       []string{"0/1/2/3"},
		targetPrefixName: "MODULES-IN-",
		curDir:           "test0",
		wantErr:          true,
	}, {
		description:        "one target dir specified, no target specified but has colon",
		dirs:               []string{"1/2/3:"},
		createDirs:         []string{"0/1/2/3"},
		createBuildFiles:   []string{"0/1/2/3/Android.bp"},
		targetPrefixName:   "MODULES-IN-",
		curDir:             "0",
		expectedTargets:    []string{"MODULES-IN-0-1-2-3"},
		expectedBuildFiles: []string{"0/1/2/3/Android.mk"},
	}, {
		description:        "one target dir specified, single target specified",
		dirs:               []string{"1/2/3:t1,t2"},
		createDirs:         []string{"0/1/2/3"},
		createBuildFiles:   []string{"0/1/2/3/Android.bp"},
		targetPrefixName:   "MODULES-IN-",
		curDir:             "0",
		expectedTargets:    []string{"t1", "t2"},
		expectedBuildFiles: []string{"0/1/2/3/Android.mk"},
	}, {
		description:      "one target dir specified, blank targets",
		dirs:             []string{"1/2/3:,"},
		createDirs:       []string{"0/1/2/3"},
		createBuildFiles: []string{"0/1/2/3/Android.bp"},
		targetPrefixName: "MODULES-IN-",
		curDir:           "0",
		wantErr:          true,
	}, {
		description:      "one target dir specified, blank target",
		dirs:             []string{"1/2/3:,t1"},
		createDirs:       []string{"0/1/2/3"},
		createBuildFiles: []string{"0/1/2/3/Android.bp"},
		targetPrefixName: "MODULES-IN-",
		curDir:           "0",
		wantErr:          true,
	}, {
		description:      "one target dir specified, blank target",
		dirs:             []string{"1/2/3:t1,"},
		createDirs:       []string{"0/1/2/3"},
		createBuildFiles: []string{"0/1/2/3/Android.bp"},
		targetPrefixName: "MODULES-IN-",
		curDir:           "0",
		wantErr:          true,
	}, {
		description:        "one target dir specified, many targets",
		dirs:               []string{"1/2/3:t1,t2,t3,t4,t5,t6,t7,t8,t9,t10"},
		createDirs:         []string{"0/1/2/3"},
		createBuildFiles:   []string{"0/1/2/3/Android.bp"},
		targetPrefixName:   "MODULES-IN-",
		curDir:             "0",
		expectedTargets:    []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9", "t10"},
		expectedBuildFiles: []string{"0/1/2/3/Android.mk"},
	}, {
		description:      "one target dir specified, one target specified, no build file",
		dirs:             []string{"1/2/3:t1"},
		createDirs:       []string{"0/1/2/3"},
		targetPrefixName: "MODULES-IN-",
		curDir:           "0",
		wantErr:          true,
	}, {
		description:      "one target dir specified, one target specified, build file not in target dir",
		dirs:             []string{"1/2/3:t1"},
		createDirs:       []string{"0/1/2/3"},
		createBuildFiles: []string{"0/1/2/Android.mk"},
		targetPrefixName: "MODULES-IN-",
		curDir:           "0",
		wantErr:          true,
	}, {
		description:        "one target dir specified, build file not in target dir",
		dirs:               []string{"1/2/3"},
		createDirs:         []string{"0/1/2/3"},
		createBuildFiles:   []string{"0/1/2/Android.mk"},
		targetPrefixName:   "MODULES-IN-",
		curDir:             "0",
		expectedTargets:    []string{"MODULES-IN-0-1-2"},
		expectedBuildFiles: []string{"0/1/2/Android.mk"},
	}, {
		description:        "multiple targets dir specified, targets specified",
		dirs:               []string{"1/2/3:t1,t2", "3/4:t3,t4,t5"},
		createDirs:         []string{"0/1/2/3", "0/3/4"},
		createBuildFiles:   []string{"0/1/2/3/Android.bp", "0/3/4/Android.mk"},
		targetPrefixName:   "MODULES-IN-",
		curDir:             "0",
		expectedTargets:    []string{"t1", "t2", "t3", "t4", "t5"},
		expectedBuildFiles: []string{"0/1/2/3/Android.mk", "0/3/4/Android.mk"},
	}, {
		description:        "multiple targets dir specified, one directory has targets specified",
		dirs:               []string{"1/2/3:t1,t2", "3/4"},
		createDirs:         []string{"0/1/2/3", "0/3/4"},
		createBuildFiles:   []string{"0/1/2/3/Android.bp", "0/3/4/Android.mk"},
		targetPrefixName:   "GET-INSTALL-PATH-IN-",
		curDir:             "0",
		expectedTargets:    []string{"t1", "t2", "GET-INSTALL-PATH-IN-0-3-4"},
		expectedBuildFiles: []string{"0/1/2/3/Android.mk", "0/3/4/Android.mk"},
	}, {
		description:      "two target dirs specified, one dir exist",
		dirs:             []string{"1/2/3:t1", "3/4"},
		createDirs:       []string{"0/1/2/3"},
		createBuildFiles: []string{"0/1/2/Android.mk"},
		targetPrefixName: "MODULES-IN-",
		curDir:           "0",
		wantErr:          true,
	}, {
		description:        "multiple targets dirs specified at root source tree",
		dirs:               []string{"0/1/2/3:t1,t2", "0/3/4"},
		createDirs:         []string{"0/1/2/3", "0/3/4"},
		createBuildFiles:   []string{"0/1/2/3/Android.bp", "0/3/4/Android.mk"},
		targetPrefixName:   "GET-INSTALL-PATH-IN-",
		curDir:             ".",
		expectedTargets:    []string{"t1", "t2", "GET-INSTALL-PATH-IN-0-3-4"},
		expectedBuildFiles: []string{"0/1/2/3/Android.mk", "0/3/4/Android.mk"},
	}, {
		description:      "no directories specified",
		dirs:             []string{},
		createDirs:       []string{"0/1/2/3", "0/3/4"},
		createBuildFiles: []string{"0/1/2/3/Android.bp", "0/3/4/Android.mk"},
		targetPrefixName: "GET-INSTALL-PATH-IN-",
		curDir:           ".",
	}}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			defer logger.Recover(func(err error) {
				if !tt.wantErr {
					t.Fatalf("Got unexpected error: %v", err)
				}
			})

			// Create the root source tree.
			topDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(topDir)

			createDirectories(t, topDir, tt.createDirs)
			createBuildFiles(t, topDir, tt.createBuildFiles)
			r := setTop(t, topDir)
			defer r()

			targets, buildFiles := getTargetsFromDirs(ctx, tt.curDir, tt.dirs, tt.targetPrefixName)
			if !reflect.DeepEqual(targets, tt.expectedTargets) {
				t.Errorf("expected %v, got %v for targets", tt.expectedTargets, targets)
			}
			if !reflect.DeepEqual(buildFiles, tt.expectedBuildFiles) {
				t.Errorf("expected %v, got %v for build files", tt.expectedBuildFiles, buildFiles)
			}
		})
	}
}

func TestConfigFindBuildFile(t *testing.T) {
	ctx := testContext()

	tests := []struct {
		// Test description.
		description string

		// Directory to create, also the base directory is where findBuildFile is invoked.
		dir string

		// Array of build files to create in dir.
		buildFiles []string

		// Expected build file path to find.
		expectedBuildFile string
	}{{
		description:       "build file exists at leaf directory",
		dir:               "1/2/3",
		buildFiles:        []string{"1/2/3/Android.bp"},
		expectedBuildFile: "1/2/3/Android.mk",
	}, {
		description:       "build file exists in all directory paths",
		dir:               "1/2/3",
		buildFiles:        []string{"1/Android.mk", "1/2/Android.mk", "1/2/3/Android.mk"},
		expectedBuildFile: "1/2/3/Android.mk",
	}, {
		description: "build file does not exist in all directory paths",
		dir:         "1/2/3",
	}, {
		description: "build file exists only at top directory",
		dir:         "1/2/3",
		buildFiles:  []string{"Android.bp"},
	}, {
		description:       "build file exist in a subdirectory",
		dir:               "1/2/3",
		buildFiles:        []string{"1/2/Android.bp"},
		expectedBuildFile: "1/2/Android.mk",
	}, {
		description:       "build file exists in a subdirectory",
		dir:               "1/2/3",
		buildFiles:        []string{"1/Android.mk"},
		expectedBuildFile: "1/Android.mk",
	}, {
		description: "top directory",
		buildFiles:  []string{"Android.bp"},
	}}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			defer logger.Recover(func(err error) {
				t.Fatalf("Got unexpected error: %v", err)
			})

			topDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(topDir)

			if tt.dir != "" {
				createDirectories(t, topDir, []string{tt.dir})
			}

			createBuildFiles(t, topDir, tt.buildFiles)

			curDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Could not get working directory: %v", err)
			}
			defer func() { os.Chdir(curDir) }()
			if err := os.Chdir(topDir); err != nil {
				t.Fatalf("Could not change top dir to %s: %v", topDir, err)
			}

			buildFile := findBuildFile(ctx, tt.dir)
			if buildFile != tt.expectedBuildFile {
				t.Errorf("expected %q, got %q for build file", tt.expectedBuildFile, buildFile)
			}
		})
	}
}

func TestConfigSplitArgs(t *testing.T) {
	tests := []struct {
		// Test description.
		description string

		// Arguments passed in to soong_ui.
		args []string

		// Expected newArgs list after extracting the directories.
		expectedNewArgs []string

		// Expected directories
		expectedDirs []string
	}{{
		description:     "flags but no directories specified",
		args:            []string{"showcommands", "-j", "-k"},
		expectedNewArgs: []string{"showcommands", "-j", "-k"},
	}, {
		description:     "flags and one directory specified",
		args:            []string{"snod", "-j", "dir:target1,target2"},
		expectedNewArgs: []string{"snod", "-j"},
		expectedDirs:    []string{"dir:target1,target2"},
	}, {
		description:     "flags and directories specified",
		args:            []string{"dist", "-k", "dir1", "dir2:target1,target2"},
		expectedNewArgs: []string{"dist", "-k"},
		expectedDirs:    []string{"dir1", "dir2:target1,target2"},
	}, {
		description:  "only directories specified",
		args:         []string{"dir1", "dir2", "dir3:target1,target2"},
		expectedDirs: []string{"dir1", "dir2", "dir3:target1,target2"},
	}}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			args, dirs := splitArgs(tt.args)
			if !reflect.DeepEqual(tt.expectedNewArgs, args) {
				t.Errorf("expected %v, got %v for arguments", tt.expectedNewArgs, args)
			}
			if !reflect.DeepEqual(tt.expectedDirs, dirs) {
				t.Errorf("expected %v, got %v for directories", tt.expectedDirs, dirs)
			}
		})
	}
}

// newTestConfig is a fake NewConfig function call to test the NewBuildActionConfig function.
func newTestConfig(ctx Context, args ...string) Config {
	config := configImpl{}
	config.arguments = args
	return Config{&config}
}

type envVar struct {
	name  string
	value string
}

type buildActionTestCase struct {
	// Test description.
	description string

	// Arguments passed in to soong_ui.
	args []string

	// Directories to be created.
	createDirs []string

	// Build files to be created.
	createBuildFiles []string

	// Directory where the build action was invoked.
	curDir string

	// WITH_TIDY_ONLY environment variable specified.
	tidyOnly string

	// Expected arguments to be in Config instance.
	expectedArgs []string

	// Expected environment variables to be set.
	expectedEnvVars []envVar

	// Expecting an error?
	wantErr bool
}

func testBuildAction(t *testing.T, tt buildActionTestCase, action BuildAction, buildDependencies bool) {
	ctx := testContext()

	// Environment variables to set it to blank on every test case run.
	resetEnvVars := []string{
		"ONE_SHOT_MAKEFILE",
		"WITH_TIDY_ONLY",
	}

	for _, name := range resetEnvVars {
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("failed to unset environment variable %s: %v", name, err)
		}
	}
	if tt.tidyOnly != "" {
		if err := os.Setenv("WITH_TIDY_ONLY", tt.tidyOnly); err != nil {
			t.Errorf("failed to set WITH_TIDY_ONLY to %s: %v", tt.tidyOnly, err)
		}
	}

	defer logger.Recover(func(err error) {
		if !tt.wantErr {
			t.Fatalf("Got unexpected error: %v", err)
		}
	})

	// Create the root source tree.
	topDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(topDir)

	createDirectories(t, topDir, tt.createDirs)
	createBuildFiles(t, topDir, tt.createBuildFiles)

	r := setTop(t, topDir)
	defer r()

	// The next block is to create the root build file.
	rootBuildFileDir := filepath.Dir(srcDirFileCheck)
	if err := os.MkdirAll(rootBuildFileDir, 0755); err != nil {
		t.Fatalf("Failed to create %s directory: %v", rootBuildFileDir, err)
	}

	if err := ioutil.WriteFile(srcDirFileCheck, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create %s file: %v", srcDirFileCheck, err)
	}

	args := getConfigArgs(action, tt.curDir, buildDependencies, ctx, tt.args)
	if !reflect.DeepEqual(tt.expectedArgs, args) {
		t.Fatalf("expected %v, got %v for config arguments", tt.expectedArgs, args)
	}

	for _, env := range tt.expectedEnvVars {
		if val := os.Getenv(env.name); val != env.value {
			t.Errorf("expecting %s, got %s for environment variable %s", env.value, val, env.name)
		}
	}

	// If it has reached here, it means the error path was not triggered.
	if tt.wantErr {
		t.Errorf("expecting error")
	}
}

func TestConfigNewBuildActionConfigBuildAllModules(t *testing.T) {
	tests := []buildActionTestCase{{
		description:      "normal execution in a directory",
		createDirs:       []string{"0/1/2"},
		createBuildFiles: []string{"0/1/2/Android.mk"},
		curDir:           "0/1/2",
		args:             []string{"-j", "-k", "showcommands"},
		expectedArgs:     []string{"-j", "-k", "showcommands"},
	}, {
		description:  "normal execution in root directory with args",
		args:         []string{"-j", "-k", "fake_module"},
		expectedArgs: []string{"-j", "-k", "fake_module"},
	}, {
		description: "build action executed at root directory, no args",
	}}
	for _, tt := range tests {
		for _, buildDependencies := range []bool{true, false} {
			desc := fmt.Sprintf("build action BUILD_ALL_MODULES, %s, build dependency: %t", tt.description, buildDependencies)
			t.Run(desc, func(t *testing.T) {
				testBuildAction(t, tt, BUILD_ALL_MODULES, buildDependencies)
			})
		}
	}
}

// TODO: Remove this test case once mm shell build command has been deprecated.
func TestConfigNewBuildActionConfigBuildModulesInDirNoDeps(t *testing.T) {
	tests := []buildActionTestCase{{
		description:      "normal execution in a directory",
		createDirs:       []string{"0/1/2"},
		createBuildFiles: []string{"0/1/2/Android.mk"},
		curDir:           "0/1/2",
		args:             []string{"-j", "-k", "showcommands", "fake-module"},
		expectedArgs:     []string{"-j", "-k", "showcommands", "fake-module", "MODULES-IN-0-1-2"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/2/Android.mk"}},
	}, {
		description:      "makefile in parent directory",
		createDirs:       []string{"0/1/2"},
		createBuildFiles: []string{"0/1/Android.mk"},
		curDir:           "0/1/2",
		expectedArgs:     []string{"MODULES-IN-0-1"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/Android.mk"}},
	}, {
		description:  "build file not found",
		createDirs:   []string{"0/1/2"},
		curDir:       "0/1/2",
		expectedArgs: []string{"MODULES-IN-0-1-2"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/2/Android.mk"}},
	}, {
		description: "build action executed at root directory",
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: ""}},
	}, {
		description:      "GET-INSTALL-PATH specified,",
		args:             []string{"GET-INSTALL-PATH"},
		createDirs:       []string{"0/1/2"},
		createBuildFiles: []string{"0/1/Android.mk"},
		curDir:           "0/1/2",
		expectedArgs:     []string{"GET-INSTALL-PATH-IN-0-1"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/Android.mk"}},
	}, {
		description:      "tidy only environment variable specified,",
		args:             []string{"GET-INSTALL-PATH"},
		createDirs:       []string{"0/1/2"},
		createBuildFiles: []string{"0/1/Android.mk"},
		curDir:           "0/1/2",
		tidyOnly:         "true",
		expectedArgs:     []string{"tidy_only"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/Android.mk"}},
	}}
	for _, tt := range tests {
		t.Run("build action BUILD_MODULES_IN_DIR without their dependencies, "+tt.description, func(t *testing.T) {
			testBuildAction(t, tt, BUILD_MODULES_IN_A_DIRECTORY, false)
		})
	}
}

func TestConfigNewBuildActionConfigBuildModulesInDir(t *testing.T) {
	tests := []buildActionTestCase{{
		description:      "normal execution in a directory",
		createDirs:       []string{"0/1/2"},
		createBuildFiles: []string{"0/1/2/Android.mk"},
		curDir:           "0/1/2",
		args:             []string{"fake-module"},
		expectedArgs:     []string{"fake-module", "MODULES-IN-0-1-2"},
	}, {
		description:      "build file in parent directory",
		createDirs:       []string{"0/1/2"},
		createBuildFiles: []string{"0/1/Android.mk"},
		curDir:           "0/1/2",
		expectedArgs:     []string{"MODULES-IN-0-1"},
	},
		{
			description:      "build file in parent directory, multiple module names passed in",
			createDirs:       []string{"0/1/2"},
			createBuildFiles: []string{"0/1/Android.mk"},
			curDir:           "0/1/2",
			args:             []string{"fake-module1", "fake-module2", "fake-module3"},
			expectedArgs:     []string{"fake-module1", "fake-module2", "fake-module3", "MODULES-IN-0-1"},
		}, {
			description:      "build file in 2nd level parent directory",
			createDirs:       []string{"0/1/2"},
			createBuildFiles: []string{"0/Android.bp"},
			curDir:           "0/1/2",
			expectedArgs:     []string{"MODULES-IN-0"},
		}, {
			description: "build action executed at root directory",
		}, {
			description:  "build file not found - no error is expected to return",
			createDirs:   []string{"0/1/2"},
			curDir:       "0/1/2",
			expectedArgs: []string{"MODULES-IN-0-1-2"},
		}, {
			description:      "GET-INSTALL-PATH specified,",
			args:             []string{"GET-INSTALL-PATH", "-j", "-k", "GET-INSTALL-PATH"},
			createDirs:       []string{"0/1/2"},
			createBuildFiles: []string{"0/1/Android.mk"},
			curDir:           "0/1/2",
			expectedArgs:     []string{"-j", "-k", "GET-INSTALL-PATH-IN-0-1"},
		}, {
			description:      "tidy only environment variable specified,",
			args:             []string{"GET-INSTALL-PATH"},
			createDirs:       []string{"0/1/2"},
			createBuildFiles: []string{"0/1/Android.mk"},
			curDir:           "0/1/2",
			tidyOnly:         "true",
			expectedArgs:     []string{"tidy_only"},
		}}
	for _, tt := range tests {
		t.Run("build action BUILD_MODULES_IN_DIR, "+tt.description, func(t *testing.T) {
			testBuildAction(t, tt, BUILD_MODULES_IN_A_DIRECTORY, true)
		})
	}
}

// TODO: Remove this test case once mmm shell build command has been deprecated.
func TestConfigNewBuildActionConfigBuildModulesInDirsNoDeps(t *testing.T) {
	tests := []buildActionTestCase{{
		description:      "normal execution in a directory",
		createDirs:       []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/2/3.3"},
		createBuildFiles: []string{"0/1/2/3.1/Android.bp", "0/1/2/3.2/Android.bp", "0/1/2/3.3/Android.bp"},
		curDir:           "0/1/2",
		args:             []string{"3.1/:t1,t2", "3.2/:t3,t4", "3.3/:t5,t6"},
		expectedArgs:     []string{"t1", "t2", "t3", "t4", "t5", "t6"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/2/3.1/Android.mk 0/1/2/3.2/Android.mk 0/1/2/3.3/Android.mk"}},
	}, {
		description:      "GET-INSTALL-PATH specified",
		createDirs:       []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/2/3.3"},
		createBuildFiles: []string{"0/1/2/3.1/Android.bp", "0/1/2/3.2/Android.bp", "0/1/2/3.3/Android.bp"},
		curDir:           "0/1/2",
		args:             []string{"GET-INSTALL-PATH", "3.1/", "3.2/", "3.3/:t6"},
		expectedArgs:     []string{"GET-INSTALL-PATH-IN-0-1-2-3.1", "GET-INSTALL-PATH-IN-0-1-2-3.2", "t6"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/2/3.1/Android.mk 0/1/2/3.2/Android.mk 0/1/2/3.3/Android.mk"}},
	}, {
		description:      "tidy only environment variable specified",
		createDirs:       []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/2/3.3"},
		createBuildFiles: []string{"0/1/2/3.1/Android.bp", "0/1/2/3.2/Android.bp", "0/1/2/3.3/Android.bp"},
		curDir:           "0/1/2",
		args:             []string{"GET-INSTALL-PATH", "3.1/", "3.2/", "3.3/:t6"},
		tidyOnly:         "1",
		expectedArgs:     []string{"tidy_only"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/2/3.1/Android.mk 0/1/2/3.2/Android.mk 0/1/2/3.3/Android.mk"}},
	}, {
		description:      "normal execution from top dir directory",
		createDirs:       []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/2/3.3"},
		createBuildFiles: []string{"0/1/2/3.1/Android.bp", "0/1/2/3.2/Android.bp", "0/1/2/3.3/Android.bp"},
		curDir:           ".",
		args:             []string{"0/1/2/3.1", "0/1/2/3.2/:t3,t4", "0/1/2/3.3/:t5,t6"},
		expectedArgs:     []string{"MODULES-IN-0-1-2-3.1", "t3", "t4", "t5", "t6"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: "0/1/2/3.1/Android.mk 0/1/2/3.2/Android.mk 0/1/2/3.3/Android.mk"}},
	}}
	for _, tt := range tests {
		t.Run("build action BUILD_MODULES_IN_DIRS_NO_DEPS, "+tt.description, func(t *testing.T) {
			testBuildAction(t, tt, BUILD_MODULES_IN_DIRECTORIES, false)
		})
	}
}

func TestConfigNewBuildActionConfigBuildModulesInDirs(t *testing.T) {
	tests := []buildActionTestCase{{
		description:      "normal execution in a directory",
		createDirs:       []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/2/3.3"},
		createBuildFiles: []string{"0/1/2/3.1/Android.bp", "0/1/2/3.2/Android.bp", "0/1/2/3.3/Android.bp"},
		curDir:           "0/1/2",
		args:             []string{"3.1/", "3.2/", "3.3/"},
		expectedArgs:     []string{"MODULES-IN-0-1-2-3.1", "MODULES-IN-0-1-2-3.2", "MODULES-IN-0-1-2-3.3"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: ""}},
	}, {
		description:      "GET-INSTALL-PATH specified",
		createDirs:       []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/3"},
		createBuildFiles: []string{"0/1/2/3.1/Android.bp", "0/1/2/3.2/Android.bp", "0/1/Android.bp"},
		curDir:           "0/1",
		args:             []string{"GET-INSTALL-PATH", "2/3.1/", "2/3.2", "3"},
		expectedArgs:     []string{"GET-INSTALL-PATH-IN-0-1-2-3.1", "GET-INSTALL-PATH-IN-0-1-2-3.2", "GET-INSTALL-PATH-IN-0-1"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: ""}},
	}, {
		description:      "tidy only environment variable specified",
		createDirs:       []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/2/3.3"},
		createBuildFiles: []string{"0/1/2/3.1/Android.bp", "0/1/2/3.2/Android.bp", "0/1/2/3.3/Android.bp"},
		curDir:           "0/1/2",
		args:             []string{"GET-INSTALL-PATH", "3.1/", "3.2/", "3.3"},
		tidyOnly:         "1",
		expectedArgs:     []string{"tidy_only"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: ""}},
	}, {
		description:      "normal execution from top dir directory",
		createDirs:       []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/3", "0/2"},
		createBuildFiles: []string{"0/1/2/3.1/Android.bp", "0/1/2/3.2/Android.bp", "0/1/3/Android.bp", "0/2/Android.bp"},
		curDir:           ".",
		args:             []string{"0/1/2/3.1", "0/1/2/3.2", "0/1/3", "0/2"},
		expectedArgs:     []string{"MODULES-IN-0-1-2-3.1", "MODULES-IN-0-1-2-3.2", "MODULES-IN-0-1-3", "MODULES-IN-0-2"},
		expectedEnvVars: []envVar{
			envVar{
				name:  "ONE_SHOT_MAKEFILE",
				value: ""}},
	}}
	for _, tt := range tests {
		t.Run("build action BUILD_MODULES_IN_DIRS, "+tt.description, func(t *testing.T) {
			testBuildAction(t, tt, BUILD_MODULES_IN_DIRECTORIES, true)
		})
	}
}
