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

type pyBinary struct {
	name           string
	actualVersion  string
	depsPyRunfiles []string
}

var (
	buildNamePrefix          = `soong_python_test`
	moduleVariantErrTemplate = "%s: module %q variant %q: "
	pkgPathErrTemplate       = moduleVariantErrTemplate +
		"pkg_path: %q is not a valid format."
	badIdentifierErrTemplate = moduleVariantErrTemplate +
		"srcs: the path %q contains invalid token %q."
	dupRunfileErrTemplate = moduleVariantErrTemplate +
		"found two files to be placed at the same runfiles location %q." +
		" First file: in module %s at path %q." +
		" Second file: in module %s at path %q."
	noSrcFileErr      = moduleVariantErrTemplate + "doesn't have any source files!"
	badSrcFileExtErr  = moduleVariantErrTemplate + "srcs: found non (.py) file: %q!"
	badDataFileExtErr = moduleVariantErrTemplate + "data: found (.py) file: %q!"
	bpFile            = "Blueprints"

	data = []struct {
		desc      string
		mockFiles map[string][]byte

		errors           []string
		expectedBinaries []pyBinary
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
				fmt.Sprintf(noSrcFileErr,
					"dir/Blueprints:1:1", "lib1", "PY3"),
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
				fmt.Sprintf(badSrcFileExtErr,
					"dir/Blueprints:3:11", "lib1", "PY3", "dir/file1.exe"),
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
				fmt.Sprintf(badDataFileExtErr,
					"dir/Blueprints:6:11", "lib1", "PY3", "dir/file2.py"),
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
					"lib1", "PY3", "runfiles/a/b/c/-e/f/file1.py", "-e"),
				fmt.Sprintf(badIdentifierErrTemplate, "dir/Blueprints:4:11",
					"lib1", "PY3", "runfiles/a/b/c/.file1.py", ".file1"),
				fmt.Sprintf(badIdentifierErrTemplate, "dir/Blueprints:4:11",
					"lib1", "PY3", "runfiles/a/b/c/123/file1.py", "123"),
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
						libs: [
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
					"lib2", "PY3", "runfiles/a/b/c/file1.py", "lib2", "dir/file1.py",
					"lib1", "dir/c/file1.py"),
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
						version: {
							py2: {
								enabled: true,
							},
							py3: {
								enabled: true,
							},
						},
					}

					python_library_host {
						name: "lib6",
						pkg_path: "c/d/",
						srcs: [
							"file2.py",
						],
						libs: [
							"lib5",
						],
					}

					python_binary_host {
						name: "bin",
						pkg_path: "e/",
						srcs: [
							"bin.py",
						],
						libs: [
							"lib5",
						],
						version: {
							py3: {
								enabled: true,
								interpreter: "/usr/bin/python",
								srcs: [
									"file4.py",
								],
								libs: [
									"lib6",
								],
							},
						},
					}`,
				),
				filepath.Join("dir", "file1.py"): nil,
				filepath.Join("dir", "file2.py"): nil,
				filepath.Join("dir", "bin.py"):   nil,
				filepath.Join("dir", "file4.py"): nil,
				"build/soong/python/scripts/stub_template_host.txt": []byte(
					`PYTHON_BINARY = '%interpreter%'
				MAIN_FILE = '%main%'`),
			},
			expectedBinaries: []pyBinary{
				{
					name:          "bin",
					actualVersion: "PY3",
					depsPyRunfiles: []string{
						"runfiles/a/b/file1.py",
						"runfiles/c/d/file2.py",
					},
				},
			},
		},
	}
)

func TestPythonModule(t *testing.T) {
	config := setupBuildEnv(t)
	defer tearDownBuildEnv()
	android.TestPreDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.BottomUp("version_split", versionSplitMutator()).Parallel()
	})
	for _, d := range data {
		t.Run(d.desc, func(t *testing.T) {
			ctx := blueprint.NewContext()
			android.RegisterTestMutators(ctx)
			ctx.RegisterModuleType("python_library_host", PythonLibraryHostFactory)
			ctx.RegisterModuleType("python_binary_host", PythonBinaryHostFactory)
			ctx.MockFileSystem(d.mockFiles)
			_, testErrs := ctx.ParseBlueprintsFiles(bpFile)
			fail(t, testErrs)
			_, actErrs := ctx.PrepareBuildActions(config)
			if len(actErrs) > 0 {
				testErrs = append(testErrs, expectErrors(t, actErrs, d.errors)...)
			} else {
				for _, e := range d.expectedBinaries {
					testErrs = append(testErrs,
						expectModule(t, ctx, e.name, e.actualVersion,
							e.depsPyRunfiles)...)
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
	// TODO(nanzhang): use deepCompare() Once Jeff checked in his pull request.
	if len(actErrStrs) != len(expErrs) {
		panic(actErrStrs[0])
		t.Fatalf("got (%d) errors, expected (%d) errors!", len(actErrStrs), len(expErrs))
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
	depsPyRunfiles []string) (testErrs []error) {
	module := findModule(ctx, name, variant)
	if module == nil {
		t.Fatalf("failed to find module %s!", name)
	}

	base, baseOk := module.(*pythonBaseModule)
	if !baseOk {
		t.Fatalf("%s is not Python module!", name)
	}
	sub, subOk := base.subModule.(*PythonBinary)
	if !subOk {
		t.Fatalf("%s is not Python binary!", name)
	}

	if !reflect.DeepEqual(sub.depsPyRunfiles, depsPyRunfiles) {
		testErrs = append(testErrs, errors.New(fmt.Sprintf(
			`binary "%s" variant "%s" has unexpected depsPyRunfiles: %s!`,
			base.Name(),
			base.properties.ActualVersion,
			sub.depsPyRunfiles)))
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
