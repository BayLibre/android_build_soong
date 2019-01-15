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

func testPrebuiltEtc(t *testing.T, bp string) *TestContext {
	config, buildDir := setUp(t)
	defer tearDown(buildDir)

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

func setUp(t *testing.T) (config Config, buildDir string) {
	buildDir, err := ioutil.TempDir("", "soong_apex_test")
	if err != nil {
		t.Fatal(err)
	}

	config = TestArchConfig(buildDir, nil)
	return
}

func tearDown(buildDir string) {
	os.RemoveAll(buildDir)
}

func TestPrebuiltEtcVariants(t *testing.T) {
	testPrebuiltEtc(t, `
		prebuilt_etc {
			name: "myetc",
			src: "myprebuilt",
		}
	`)
}

func TestPrebuiltEtcOutputPath(t *testing.T) {
	ctx := testPrebuiltEtc(t, `
		prebuilt_etc {
			name: "foo.conf",
			src: "foo.conf",
			filename: "foo.installed.conf",
		}
	`)

	p := ctx.ModuleForTests("foo.conf", "android_common_core").Module().(*PrebuiltEtc)
	if p.outputFilePath.Base() != "foo.installed.conf" {
		t.Errorf("expected foo.installed.conf, got %q", p.outputFilePath.Base())
	}
}

func TestPrebuiltEtcGlob(t *testing.T) {
	ctx := testPrebuiltEtc(t, `
		prebuilt_etc {
			name: "my_foo",
			src: "foo.*",
		}
		prebuilt_etc {
			name: "my_bar",
			src: "bar.*",
			filename_from_src: true,
		}
	`)

	p := ctx.ModuleForTests("my_foo", "android_common_core").Module().(*PrebuiltEtc)
	if p.outputFilePath.Base() != "my_foo" {
		t.Errorf("expected my_foo, got %q", p.outputFilePath.Base())
	}

	p = ctx.ModuleForTests("my_bar", "android_common_core").Module().(*PrebuiltEtc)
	if p.outputFilePath.Base() != "bar.conf" {
		t.Errorf("expected bar.conf, got %q", p.outputFilePath.Base())
	}
}
