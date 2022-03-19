/*
 * Copyright (C) 2022 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bcid

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"android/soong/android"
	"android/soong/apex"
	"android/soong/bcid/provenance_metadata_proto"
	"android/soong/java"
	"google.golang.org/protobuf/encoding/prototext"
)

const (
	apkContent  = "app1.apk content"
	apexContent = "apex1.apex content"
)

var mockedFiles = android.GroupFixturePreparers(
	android.FixtureAddFile("prebuilt/arm/app1.apk", []byte(apkContent)),
	android.FixtureAddFile("prebuilt/arm64/app1.apk", []byte(apkContent)),
	android.FixtureAddFile("prebuilt/x86/app1.apk", []byte(apkContent)),
	android.FixtureAddFile("prebuilt/x86_64/app1.apk", []byte(apkContent)),

	android.FixtureAddFile("apex1-arm.apex", []byte(apexContent)),
	android.FixtureAddFile("apex1-arm64.apex", []byte(apexContent)),
	android.FixtureAddFile("apex1-x86.apex", []byte(apexContent)),
	android.FixtureAddFile("apex1-x86_64.apex", []byte(apexContent)),
)

func TestBcidSingleton(t *testing.T) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithBcidSingleton,
		android.PrepareForTestWithAndroidMk,
		java.PrepareForTestWithJavaDefaultModules,
		apex.PrepareForTestWithApexBuildComponents,
		mockedFiles).RunTestWithBp(t, `
		android_app_import {
				name: "app1",
				default_dev_cert: true,
				arch: {
						arm: {
								apk: "prebuilt/arm/app1.apk",
						},
						arm64: {
								apk: "prebuilt/arm64/app1.apk",
						},
						x86: {
								apk: "prebuilt/x86/app1.apk",
						},
						x86_64: {
								apk: "prebuilt/x86_64/app1.apk",
						},
				},
		}
		prebuilt_apex {
				name: "apex1",
				arch: {
						arm: {
								src: "apex1-arm.apex",
						},
						arm64: {
								src: "apex1-arm64.apex",
						},
						x86: {
								src: "apex1-x86.apex",
						},
						x86_64: {
								src: "apex1-x86_64.apex",
						},
				},
				filename: "apex1.apex",
		}
	`)

	bytes, _ := ioutil.ReadFile(filepath.Join(result.Config.OutDir(), provenanceMetadataFile))
	metadataList := provenance_metadata_proto.ProvenanceMetaDataList{}
	err := prototext.Unmarshal(bytes, &metadataList)
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	android.AssertIntEquals(t, "metadata list size", 2, len(metadataList.GetMetadata()))
	for _, metaData := range metadataList.GetMetadata() {
		switch metaData.ModuleName {
		case "prebuilt_app1":
			android.AssertStringDoesContain(t, "artifact path", metaData.ArtifactPath, "/app1.apk")
			android.AssertStringEquals(t, "artifact sha256 hash", genSha256(apkContent), metaData.ArtifactSha256)
			android.AssertStringEquals(t, "artifact install hash", "/system/app/app1/app1.apk", metaData.ArtifactInstallPath)
		case "prebuilt_apex1":
			android.AssertStringDoesContain(t, "artifact path", metaData.ArtifactPath, "apex1-")
			android.AssertStringEquals(t, "artifact sha256 hash", genSha256(apexContent), metaData.ArtifactSha256)
			android.AssertStringEquals(t, "artifact install hash", "/system/apex/apex1.apex", metaData.ArtifactInstallPath)
		default:
			t.Errorf("Unexpected metadata: %s", metaData)
		}
	}
}

func genSha256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
