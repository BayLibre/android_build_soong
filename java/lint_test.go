// Copyright 2020 Google Inc. All rights reserved.
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

package java

import (
	"testing"

	"android/soong/android"
)

func Test_linter_lint(t *testing.T) {
	type fields struct {
		name                 string
		manifest             android.Path
		mergedManifest       android.Path
		srcs                 android.Paths
		srcJars              android.Paths
		resources            android.Paths
		classpath            android.Paths
		classes              android.Path
		extraLintCheckJars   android.Paths
		test                 bool
		library              bool
		minSdkVersion        string
		targetSdkVersion     string
		compileSdkVersion    string
		javaLanguageLevel    string
		kotlinLanguageLevel  string
		properties           LintProperties
		buildModuleReportZip bool
	}
	type args struct {
		ctx android.ModuleContext
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &linter{
				name:                 tt.fields.name,
				manifest:             tt.fields.manifest,
				mergedManifest:       tt.fields.mergedManifest,
				srcs:                 tt.fields.srcs,
				srcJars:              tt.fields.srcJars,
				resources:            tt.fields.resources,
				classpath:            tt.fields.classpath,
				classes:              tt.fields.classes,
				extraLintCheckJars:   tt.fields.extraLintCheckJars,
				test:                 tt.fields.test,
				library:              tt.fields.library,
				minSdkVersion:        tt.fields.minSdkVersion,
				targetSdkVersion:     tt.fields.targetSdkVersion,
				compileSdkVersion:    tt.fields.compileSdkVersion,
				javaLanguageLevel:    tt.fields.javaLanguageLevel,
				kotlinLanguageLevel:  tt.fields.kotlinLanguageLevel,
				properties:           tt.fields.properties,
				buildModuleReportZip: tt.fields.buildModuleReportZip,
			}
			_ = l
		})
	}
}
