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

package python

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"android/soong/android"
	"github.com/google/blueprint"
)

type pyModule struct {
	name          string
	actualVersion string
	destToPySrcs  map[string]string
	destToPyData  map[string]string
}

var (
	buildNamePrefix          = `soong_python_test`
	moduleVariantErrTemplate = "%s: module %q variant %q: %s"
	pkgPathErrTemplate       = "%s: module %q variant %q: pkg_path: %q is not a valid format"
	badIdentifierErrTemplate = "%s: module %q variant %q: srcs: the path %q contains invalid token %q"
	dupRunfileErrTemplate    = "%s: module %q variant %q: found duplicated (runfiles dir) map key: %q from module: %q!"
	noSrcFileErr             = "doesn't have any source files!"
	badSrcFileExtErr         = "srcs: must not have any files except .py file!"
	badDataFileExtErr        = "data: must not have any .py file!"
	bpFile                   = "Blueprints"

	data = []struct {
		desc      string
		mockFiles map[string][]byte

		errors   []string
		expected []pyModule
	}{
		{
			desc: "module without any src files",
			mockFiles: map[string][]byte{
				bpFile: []byte(`subdirs = ["dir"]`),
				filepath.Join("dir", bpFile): []byte(
					`python_library_host {
						name: "lib1",
					}`,
				),
			},
			errors: []string{
				fmt.Sprintf(moduleVariantErrTemplate,
					"dir/Blueprints:1:1", "lib1", "PY3", noSrcFileErr),
			},
		},
		{
			desc: "module with bad src file ext",
			mockFiles: map[string][]byte{
				bpFile: []byte(`subdirs = ["dir"]`),
				filepath.Join("dir", bpFile): []byte(
					`python_library_host {
						name: "lib1",
						srcs: [
							"file1.exe",
						],
					}`,
				),
				"dir/file1.exe": nil,
			},
			errors: []string{
				fmt.Sprintf(moduleVariantErrTemplate,
					"dir/Blueprints:3:11", "lib1", "PY3", badSrcFileExtErr),
			},
		},
		{
			desc: "module with bad data file ext",
			mockFiles: map[string][]byte{
				bpFile: []byte(`subdirs = ["dir"]`),
				filepath.Join("dir", bpFile): []byte(
					`python_library_host {
						name: "lib1",
						srcs: [
							"file1.py",
						],
						data: [
							"file2.py",
						],
					}`,
				),
				"dir/file1.py": nil,
				"dir/file2.py": nil,
			},
			errors: []string{
				fmt.Sprintf(moduleVariantErrTemplate,
					"dir/Blueprints:6:11", "lib1", "PY3", badDataFileExtErr),
			},
		},
		{
			desc: "module with bad pkg_path format",
			mockFiles: map[string][]byte{
				bpFile: []byte(`subdirs = ["dir"]`),
				filepath.Join("dir", bpFile): []byte(
					`python_library_host {
						name: "lib1",
						pkg_path: "a/c/../../",
						srcs: [
							"file1.py",
						],
					}

					python_library_host {
						name: "lib2",
						pkg_path: "a/c/../../../",
						srcs: [
							"file1.py",
						],
					}

					python_library_host {
						name: "lib3",
						pkg_path: "/a/c/../../",
						srcs: [
							"file1.py",
						],
					}`,
				),
				"dir/file1.py": nil,
			},
			errors: []string{
				fmt.Sprintf(pkgPathErrTemplate,
					"dir/Blueprints:11:15", "lib2", "PY3", "a/c/../../../"),
				fmt.Sprintf(pkgPathErrTemplate,
					"dir/Blueprints:19:15", "lib3", "PY3", "/a/c/../../"),
			},
		},
		{
			desc: "module with bad runfile src path format",
			mockFiles: map[string][]byte{
				bpFile: []byte(`subdirs = ["dir"]`),
				filepath.Join("dir", bpFile): []byte(
					`python_library_host {
						name: "lib1",
						pkg_path: "a/b/c/",
						srcs: [
							".file1.py",
							"123/file1.py",
							"-e/f/file1.py",
						],
					}`,
				),
				"dir/.file1.py":     nil,
				"dir/123/file1.py":  nil,
				"dir/-e/f/file1.py": nil,
			},
			errors: []string{
				fmt.Sprintf(badIdentifierErrTemplate, "dir/Blueprints:4:11",
					"lib1", "PY3", "a/b/c/-e/f/file1.py", "-e"),
				fmt.Sprintf(badIdentifierErrTemplate, "dir/Blueprints:4:11",
					"lib1", "PY3", "a/b/c/.file1.py", ".file1"),
				fmt.Sprintf(badIdentifierErrTemplate, "dir/Blueprints:4:11",
					"lib1", "PY3", "a/b/c/123/file1.py", "123"),
			},
		},
		{
			desc: "module with duplicate runfile path",
			mockFiles: map[string][]byte{
				bpFile: []byte(`subdirs = ["dir"]`),
				filepath.Join("dir", bpFile): []byte(
					`python_library_host {
						name: "lib1",
						pkg_path: "a/b/",
						srcs: [
							"c/file1.py",
						],
					}

					python_library_host {
						name: "lib2",
						pkg_path: "a/b/c/",
						srcs: [
							"file1.py",
						],
						py_libs: [
							"lib1",
						],
					}
					`,
				),
				"dir/c/file1.py": nil,
				"dir/file1.py":   nil,
			},
			errors: []string{
				fmt.Sprintf(dupRunfileErrTemplate, "dir/Blueprints:9:6",
					"lib2", "PY3", "a/b/c/file1.py", "lib1"),
			},
		},
		{
			desc: "module for testing dependencies",
			mockFiles: map[string][]byte{
				bpFile: []byte(`subdirs = ["dir"]`),
				filepath.Join("dir", bpFile): []byte(
					`python_library_host {
						name: "lib5",
						pkg_path: "a/b/",
						srcs: [
							"file1.py",
						],
						version_py2: {
							enabled: true,
						},
						version_py3: {
							enabled: true,
						},
					}

					python_library_host {
						name: "lib6",
						pkg_path: "c/d/",
						srcs: [
							"file2.py",
						],
						py_libs: [
							"lib5",
						],
					}

					python_library_host {
						name: "lib7",
						pkg_path: "e/",
						srcs: [
							"file3.py",
						],
						version_py2: {
							enabled: true,
							srcs: [
								"file4.py",
							],
							py_libs: [
								"lib5",
							],
						},
						version_py3: {
							enabled: true,
							srcs: [
								"file5.py",
							],
							py_libs: [
								"lib6",
							],
						},
					}`,
				),
				filepath.Join("dir", "file1.py"): nil,
				filepath.Join("dir", "file2.py"): nil,
				filepath.Join("dir", "file3.py"): nil,
				filepath.Join("dir", "file4.py"): nil,
				filepath.Join("dir", "file5.py"): nil,
			},
			expected: []pyModule{
				{
					name:          "lib5",
					actualVersion: "PY2",
					destToPySrcs: map[string]string{
						"a/b/file1.py": "dir/file1.py",
					},
				},
				{
					name:          "lib6",
					actualVersion: "PY3",
					destToPySrcs: map[string]string{
						"a/b/file1.py": "dir/file1.py",
						"c/d/file2.py": "dir/file2.py",
					},
				},
				{
					name:          "lib7",
					actualVersion: "PY2",
					destToPySrcs: map[string]string{
						"a/b/file1.py": "dir/file1.py",
						"e/file3.py":   "dir/file3.py",
						"e/file4.py":   "dir/file4.py",
					},
				},
				{
					name:          "lib7",
					actualVersion: "PY3",
					destToPySrcs: map[string]string{
						"a/b/file1.py": "dir/file1.py",
						"c/d/file2.py": "dir/file2.py",
						"e/file3.py":   "dir/file3.py",
						"e/file5.py":   "dir/file5.py",
					},
				},
			},
		},
	}
)

func TestPythonModule(t *testing.T) {
	config := setupBuildEnv(t)
	defer tearDownBuildEnv()
	for _, d := range data {
		t.Run(d.desc, func(t *testing.T) {
			ctx := android.NewContext()
			ctx.MockFileSystem(d.mockFiles)
			_, testErrs := ctx.ParseBlueprintsFiles(bpFile)
			fail(t, testErrs)
			_, actErrs := ctx.PrepareBuildActions(config)
			if len(actErrs) > 0 {
				testErrs = append(testErrs, expectErrors(t, actErrs, d.errors)...)
			} else {
				for _, e := range d.expected {
					testErrs = append(testErrs,
						expectModule(t, ctx, e.name, e.actualVersion,
							e.destToPySrcs, e.destToPyData)...)
				}
			}
			fail(t, testErrs)
		})
	}
}

func expectErrors(t *testing.T, actErrs []error, expErrs []string) (testErrs []error) {
	actErrStrs := []string{}
	for _, v := range actErrs {
		actErrStrs = append(actErrStrs, v.Error())
	}
	sort.Strings(actErrStrs)
	if len(actErrStrs) != len(expErrs) {
		t.Fatalf("the actual number of errors (%d) is not expected!", len(actErrStrs))
	}
	sort.Strings(expErrs)
	for i, v := range actErrStrs {
		if v != expErrs[i] {
			testErrs = append(testErrs, errors.New(v))
		}
	}

	return
}

func expectModule(t *testing.T, ctx *blueprint.Context, name, variant string,
	destToPySrcs, destToPyData map[string]string) (testErrs []error) {
	module := findModule(ctx, name, variant)
	if module == nil {
		t.Fatalf("failed to find module %s!", name)
	}
	actModule, ok := module.(*pythonBaseModule)
	if !ok {
		t.Fatalf("%s is not Python module!", name)
	}
	if !reflect.DeepEqual(
		actModule.destToPySrcs, destToPySrcs) {
		testErrs = append(testErrs, errors.New(fmt.Sprintf(
			`module "%s" variant "%s" has unexpected destToPySrcs map: %s!`,
			actModule.Name(),
			actModule.properties.ActualVersion,
			actModule.destToPySrcs)))
	}
	if len(actModule.destToPyData) != 0 {
		if !reflect.DeepEqual(
			actModule.destToPyData, destToPyData) {
			testErrs = append(testErrs, errors.New(fmt.Sprintf(
				`module "%s" variant "%s" has unexpected destToPyData map: %s!`,
				actModule.Name(),
				actModule.properties.ActualVersion,
				actModule.destToPyData)))
		}
	} else if len(destToPyData) != 0 {
		testErrs = append(testErrs,
			errors.New(fmt.Sprintf(
				`module "%s" variant "%s" has empty destToPyData map!`,
				actModule.Name(), actModule.properties.ActualVersion)))
	}

	return
}

func setupBuildEnv(t *testing.T) (config android.Config) {
	buildDir, err := ioutil.TempDir("", buildNamePrefix)
	if err != nil {
		t.Fatal(err)
	}
	config = android.TestConfig(buildDir)

	return
}

func tearDownBuildEnv() {
	os.RemoveAll(buildNamePrefix)
}

func findModule(ctx *blueprint.Context, name, variant string) blueprint.Module {
	var ret blueprint.Module
	ctx.VisitAllModules(func(m blueprint.Module) {
		if ctx.ModuleName(m) == name && ctx.ModuleSubDir(m) == variant {
			ret = m
		}
	})
	return ret
}

func fail(t *testing.T, errs []error) {
	if len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.FailNow()
	}
}
