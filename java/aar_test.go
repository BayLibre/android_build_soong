package java

import (
	"android/soong/android"
	"testing"
)

func TestAarImportProducesJniPackages(t *testing.T) {
	ctx := android.GroupFixturePreparers(
		PrepareForTestWithJavaDefaultModules,
	).RunTestWithBp(t, `
		android_library_import {
			name: "aar-no-jni",
			aars: ["aary.aar"],
		}
		android_library_import {
			name: "aar-jni",
			aars: ["aary.aar"],
			extract_jni: true,
		}`)

	testCases := []struct {
		name       string
		hasPackage bool
	}{
		{
			name:       "aar-no-jni",
			hasPackage: false,
		},
		{
			name:       "aar-jni",
			hasPackage: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := ctx.ModuleForTests(tc.name, "android_common")
			if !tc.hasPackage {
				return
			}
			outputFile := "arm64-v8a_jni.zip"
			jniOutputLibZip := app.Output(outputFile)
			if jniOutputLibZip.Rule == nil {
				t.Errorf("did not find output file %v", outputFile)
			}
		})
	}
}
