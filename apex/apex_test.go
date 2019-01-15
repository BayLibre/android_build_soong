// Copyright 2018 Google Inc. All rights reserved.
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

package apex

import (
	"io/ioutil"
	"os"
	"testing"

	"android/soong/android"
)

func testApex(t *testing.T, bp string) *android.TestContext {
	config, buildDir := setup(t)
	defer teardown(buildDir)

	ctx := android.NewTestArchContext()
	ctx.RegisterModuleType("prebuilt_etc", android.ModuleFactoryAdaptor(android.PrebuiltEtcFactory))

	ctx.Register()

	ctx.MockFileSystem(map[string][]byte{
		"Android.bp": []byte(bp),
		"myprebuilt": nil,
	})
	_, errs := ctx.ParseFileList(".", []string{"Android.bp"})
	android.FailIfErrored(t, errs)
	_, errs = ctx.PrepareBuildActions(config)
	android.FailIfErrored(t, errs)

	return ctx
}

func setup(t *testing.T) (config android.Config, buildDir string) {
	buildDir, err := ioutil.TempDir("", "soong_apex_test")
	if err != nil {
		t.Fatal(err)
	}

	config = android.TestArchConfig(buildDir, nil)
	return
}

func teardown(buildDir string) {
	os.RemoveAll(buildDir)
}

func TestFilesInSubDir(t *testing.T) {
	testApex(t, `
		prebuilt_etc {
			name: "myetc",
			src: "myprebuilt",
		}
	`)
}
