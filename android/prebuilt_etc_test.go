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

package android

import (
	"io/ioutil"
	"os"
	"testing"
)

func testApex(t *testing.T, bp string) *TestContext {
	config, buildDir := setup(t)
	defer teardown(buildDir)

	ctx := NewTestArchContext()
	ctx.RegisterModuleType("prebuilt_etc", ModuleFactoryAdaptor(PrebuiltEtcFactory))

	ctx.Register()

	ctx.MockFileSystem(map[string][]byte{
		"Android.bp": []byte(bp),
		"myprebuilt": nil,
	})
	_, errs := ctx.ParseFileList(".", []string{"Android.bp"})
	FailIfErrored(t, errs)
	_, errs = ctx.PrepareBuildActions(config)
	FailIfErrored(t, errs)

	return ctx
}

func setup(t *testing.T) (config Config, buildDir string) {
	buildDir, err := ioutil.TempDir("", "soong_apex_test")
	if err != nil {
		t.Fatal(err)
	}

	config = TestArchConfig(buildDir, nil)
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
