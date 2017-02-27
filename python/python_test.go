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
	"io/ioutil"
	"os"
	"testing"

	"android/soong/android"
	"github.com/google/blueprint"
)

var testData = []struct {
	name     string
	modules  string
	variants int
}{
	{
		name: "lib1",
		modules: `
			python_library_host {
				name: "lib1",
				pkg_path: "a/b",
				srcs: [
					"f1.py",
					"f2.py",
				],
				version_py2: {
					enabled: true,
					srcs:[
						"f3.py",
					],
				},
				version_py3: {
					enabled: true,
					srcs:[
						"f4.py",
					],
				},
			}`,
		variants: 2,
	},
}

func TestModuleVariants(t *testing.T) {
	buildDir, err := ioutil.TempDir("", "soong_test_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(buildDir)

	config := android.TestConfig(buildDir)
	for _, ele := range testData {
		ctx := android.NewContext()
		ctx.MockFileSystem(map[string][]byte{
			"Blueprints":     []byte(`subdirs = ["dir"]`),
			"dir/Blueprints": []byte(ele.modules),
			"dir/f1.py":      nil,
			"dir/f2.py":      nil,
			"dir/f3.py":      nil,
			"dir/f4.py":      nil,
		})

		_, errs := ctx.ParseBlueprintsFiles("Blueprints")
		fail(t, errs)
		_, errs = ctx.PrepareBuildActions(config)
		fail(t, errs)

		var count int
		ctx.VisitAllModules(func(m blueprint.Module) {
			if ctx.ModuleName(m) == ele.name {
				count++
			}
		})
		if count != ele.variants {
			t.Fatalf(ele.name + " has incorrect number of variants.")
		}
	}
}

func fail(t *testing.T, errs []error) {
	if len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.FailNow()
	}
}
