// Copyright 2021 Google Inc. All rights reserved.
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

package filesystem

import (
	"os"
	"strings"
	"testing"

	"android/soong/android"
	"android/soong/cc"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

var fixture = android.GroupFixturePreparers(
	android.PrepareForIntegrationTestWithAndroid,
	cc.PrepareForIntegrationTestWithCc,
	PrepareForTestWithFilesystemBuildComponents,
)

// Minimal test
func TestFileSystemDeps(t *testing.T) {
	result := fixture.RunTestWithBp(t, `
		android_filesystem {
			name: "myfilesystem",
			deps: [
				"foo",
			],
		}

		cc_binary {
			name: "foo",
			shared_libs: ["libbar"],
		}

		cc_library {
			name: "libbar",
		}
	`)

	m := result.ModuleForTests("myfilesystem", "android_common")

	android.AssertPathsRelativeToTopEquals(t, "output", []string{
		"out/soong/.intermediates/myfilesystem/android_common/myfilesystem.img",
	}, m.OutputFiles(t, ""))

	android.AssertDeepEquals(t, "deps", []string{
		"bin/foo",
		"lib64/libbar.so",
		"lib64/libc++.so",
		"lib64/libc.so",
		"lib64/libdl.so",
		"lib64/libm.so",
	}, getDeps(t, m).Files())
}

// Testing utilities

type copyCmd struct {
	src, dest string
}

type zipContents struct {
	copies []copyCmd
}

func (z zipContents) Files() []string {
	var ret []string
	for _, cp := range z.copies {
		ret = append(ret, cp.dest)
	}
	return ret
}

func getDeps(t *testing.T, m android.TestingModule) zipContents {
	t.Helper()
	var contents zipContents
	rule := m.Rule("zip_deps")
	zipDir := "/.zip/"
	commands := strings.Split(rule.RuleParams.Command, " && ")
	for _, command := range commands {
		parts := strings.Split(command, " ")
		if len(parts) == 3 && parts[0] == "cp" {
			index := strings.Index(parts[2], zipDir)
			if index == -1 {
				t.Fatal("can't find /.zip/ in", command)
			}
			contents.copies = append(contents.copies, copyCmd{parts[1], parts[2][index+len(zipDir):]})
		}
	}
	return contents
}
