// Copyright (C) 2021 The Android Open Source Project
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
	"android/soong/android"
	"path/filepath"
	"testing"
)

func checkFileInSnapshot(t *testing.T, snapshot android.TestingSingleton, file string) {
	if snapshot.MaybeOutput(file).Rule == nil {
		t.Errorf("File ", file, " not found in snapshot")
	}
}

/// Test host snapshot
//   Set the host snapshot module to single binary (bintest)
//    Validatate that required modules of bintest are included:
//             - bin_subtest
//        along with required sub module's shared libraries:
//             - libsub
//    Validate shared libraries of bintest are included :
//             - libmid
//    Validate any shared libraries of shared libraries are included:
//             - libbase
//    Validate static libraries are NOT included
//    Validate JSON file
//    Validate NOTICE files
func TestHostSnapshotGen(t *testing.T) {
	bp := `
        license_kind {
	     name: "test_notice",
             conditions: ["notice"],
	}
	license {
	     name: "host_test_license",
             visibility: ["//visibility:public"],
             license_kinds: [
               "test_notice"
             ],
             license_text: [
               "NOTICE",
             ],
	}
        cc_library_static {
             name: "libstatic",
             host_supported:  true,
        }
        cc_library_shared {
             name: "libbase",
             host_supported: true,
        }
        cc_library_shared {
             name: "libmid",
             host_supported: true,
             shared_libs: ["libbase"],
        }
        cc_library_shared {
             name: "libsub",
             host_supported: true,
        }
        cc_binary {
             name: "bin_subtest",
             host_supported: true,
             shared_libs: ["libsub"],
        }
        cc_binary {
             name: "bintest",
             host_supported: true,
             shared_libs: ["libmid"],
             static_libs: ["libstatic"],
             required: ["bin_subtest"],
        }
`
	fs := make(map[string][]byte)
	fs["NOTICE"] = []byte("Test notice")

	config := TestConfig(t.TempDir(), android.Android, nil, bp, fs)
	config.TestProductVariables.HostSnapshotModules = []string{"bintest"}

	ctx := testCcWithConfig(t, config)

	snapshotSingleton := ctx.SingletonForTests("host-snapshot")

	snapshotDir := "host-snapshot"

	jsonFile := filepath.Join(snapshotDir, "host_tools.json")

	binDir := filepath.Join(snapshotDir, "bin")
	libDir := filepath.Join(snapshotDir, "lib64")
	noticeDir := filepath.Join(snapshotDir, "NOTICE_FILES")

	sharedLibs := []string{"libmid", "libbase", "libsub"}
	staticLibs := []string{"libstatic"}
	expectedBins := []string{"bintest", "bin_subtest"}
	noticeFiles := []string{}
	buildVar := config.BuildOSTarget.String()

	checkFileInSnapshot(t, snapshotSingleton, jsonFile)

	for _, bin := range expectedBins {
		checkFileInSnapshot(t, snapshotSingleton, filepath.Join(binDir, bin))
		noticeFiles = append(noticeFiles, bin+".txt")
	}

	for _, lib := range sharedLibs {
		CheckSnapshot(t, ctx, snapshotSingleton, lib, lib+".so", libDir, buildVar+"_shared")
		noticeFiles = append(noticeFiles, lib+".txt")
	}
	for _, lib := range staticLibs {
		CheckSnapshotExclude(t, ctx, snapshotSingleton, lib, lib+".a", libDir, buildVar+"_static")
		noticeFiles = append(noticeFiles, lib+".txt")
	}
	for _, notice := range noticeFiles {
		checkFileInSnapshot(t, snapshotSingleton, filepath.Join(noticeDir, notice))
	}

}
