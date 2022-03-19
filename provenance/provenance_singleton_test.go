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

package provenance

import (
	"strings"
	"testing"

	"android/soong/android"
	"android/soong/apex"
	"android/soong/java"
)

const (
	apkContent  = "app1.apk content"
	apexContent = "apex1.apex content"
)

var (
	mockedFiles = android.GroupFixturePreparers(
		android.FixtureAddFile("prebuilt/arm/app1.apk", []byte(apkContent)),
		android.FixtureAddFile("prebuilt/arm64/app1.apk", []byte(apkContent)),
		android.FixtureAddFile("prebuilt/x86/app1.apk", []byte(apkContent)),
		android.FixtureAddFile("prebuilt/x86_64/app1.apk", []byte(apkContent)),

		android.FixtureAddFile("apex1-arm.apex", []byte(apexContent)),
		android.FixtureAddFile("apex1-arm64.apex", []byte(apexContent)),
		android.FixtureAddFile("apex1-x86.apex", []byte(apexContent)),
		android.FixtureAddFile("apex1-x86_64.apex", []byte(apexContent)),
	)
)

func TestProvenanceSingleton(t *testing.T) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithProvenanceSingleton,
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

	outputs := result.SingletonForTests("provenance_metadata_singleton").AllOutputs()
	for _, output := range outputs {
		testingBuildParam := result.SingletonForTests("provenance_metadata_singleton").Output(output)
		if strings.Contains(output, "app1") {
			android.AssertStringEquals(t, "Invalid build rule", "android/soong/provenance.genProvenanceMetaData", testingBuildParam.Rule.String())
			android.AssertStringDoesContain(t, "Invalid input", testingBuildParam.Inputs[0].String(), "app1")
			android.AssertStringDoesContain(t, "Invalid output path", output, "soong/.intermediates/provenance_metadata/prebuilt_app1/provenance_metadata.textproto")
			android.AssertStringEquals(t, "Invalid arg module_name", testingBuildParam.Args["module_name"], "prebuilt_app1")
			android.AssertStringEquals(t, "Invalid arg install_path", testingBuildParam.Args["install_path"], "/system/app/app1/app1.apk")
		} else if strings.Contains(output, "apex1") {
			android.AssertStringEquals(t, "Invalid build rule", "android/soong/provenance.genProvenanceMetaData", testingBuildParam.Rule.String())
			android.AssertStringDoesContain(t, "Invalid input", testingBuildParam.Inputs[0].String(), "apex1")
			android.AssertStringDoesContain(t, "Invalid output path", output, "soong/.intermediates/provenance_metadata/prebuilt_apex1/provenance_metadata.textproto")
			android.AssertStringEquals(t, "", testingBuildParam.Args["module_name"], "prebuilt_apex1")
			android.AssertStringEquals(t, "", testingBuildParam.Args["install_path"], "/system/apex/apex1.apex")
		} else if strings.Contains(output, "soong/provenance_metadata.textproto") {
			android.AssertStringEquals(t, "Invalid build rule", "android/soong/provenance.mergeProvenanceMetaData", testingBuildParam.Rule.String())
			android.AssertIntEquals(t, "Invalid input", len(testingBuildParam.Inputs), 2)
			android.AssertStringDoesContain(t, "Invalid output path", output, "soong/provenance_metadata.textproto")
			android.AssertIntEquals(t, "Invalid args", len(testingBuildParam.Args), 0)
		} else if strings.HasSuffix(output, "provenance_metadata") {
			android.AssertStringEquals(t, "Invalid build rule", "<builtin>:phony", testingBuildParam.Rule.String())
			android.AssertStringEquals(t, "Invalid input", testingBuildParam.Inputs[0].String(), "out/soong/provenance_metadata.textproto")
			android.AssertStringEquals(t, "Invalid output path", output, "provenance_metadata")
			android.AssertIntEquals(t, "Invalid args", len(testingBuildParam.Args), 0)
		}
	}
}
