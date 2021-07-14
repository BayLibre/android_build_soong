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

package sdk

import (
	"fmt"
	"path/filepath"
	"testing"

	"android/soong/android"
	"android/soong/java"
	"github.com/google/blueprint"
)

type testMemberTrait struct {
	android.SdkMemberTraitBase
}

type testMemberType struct {
	android.SdkMemberTypeBase
}

func (t *testMemberType) AddDependencies(ctx android.SdkDependencyContext, dependencyTag blueprint.DependencyTag, names []string) {
	for _, name := range names {
		ctx.AddVariationDependencies(nil, dependencyTag, name)

		if ctx.RequiresTrait(name, testTrait) {
			ctx.AddVariationDependencies(nil, dependencyTag, name+"_test")
		}
	}
}

func (t *testMemberType) IsInstance(module android.Module) bool {
	return true
}

func (t *testMemberType) AddPrebuiltModule(ctx android.SdkMemberContext, member android.SdkMember) android.BpModule {
	moduleType := "java_import"
	if ctx.RequiresTrait(testTrait) {
		moduleType = "java_test_import"
	}
	return ctx.SnapshotBuilder().AddPrebuiltModule(member, moduleType)
}

func (t *testMemberType) CreateVariantPropertiesStruct() android.SdkMemberProperties {
	return &testMemberTypeProperties{}
}

type testMemberTypeProperties struct {
	android.SdkMemberPropertiesBase

	path android.Path
}

func (t *testMemberTypeProperties) PopulateFromVariant(ctx android.SdkMemberContext, variant android.Module) {
	headerJars := variant.(java.ApexDependency).HeaderJars()
	if len(headerJars) != 1 {
		panic(fmt.Errorf("there must be only one header jar from %q", variant.Name()))
	}

	t.path = headerJars[0]
}

func (t *testMemberTypeProperties) AddToPropertySet(ctx android.SdkMemberContext, propertySet android.BpPropertySet) {
	if t.path != nil {
		relative := filepath.Join("javalibs", t.path.Base())
		ctx.SnapshotBuilder().CopyToSnapshot(t.path, relative)
		propertySet.AddProperty("jars", []string{relative})
	}
}

var (
	testTrait = &testMemberTrait{
		SdkMemberTraitBase: android.SdkMemberTraitBase{
			PropertyName: "test",
		},
	}

	testType = &testMemberType{
		SdkMemberTypeBase: android.SdkMemberTypeBase{
			PropertyName: "test_members",
			SupportsSdk:  true,
			Traits:       []android.SdkMemberTrait{testTrait},
		},
	}
)

func init() {
	android.RegisterSdkMemberTrait(testTrait)
	android.RegisterSdkMemberType(testType)
}

func TestBasicTrait(t *testing.T) {
	result := android.GroupFixturePreparers(
		prepareForSdkTestWithJava,
		android.FixtureWithRootAndroidBp(`
			sdk {
				name: "mysdk",
				test_members: ["myjavalib"],
				traits: {
					test: ["myjavalib"],
				},
			}
	
			java_library {
				name: "myjavalib",
				srcs: ["Test.java"],
				system_modules: "none",
				sdk_version: "none",
			}
	
			java_library {
				name: "myjavalib_test",
				srcs: ["Test.java"],
				system_modules: "none",
				sdk_version: "none",
			}
		`),
	).RunTest(t)

	CheckSnapshot(t, result, "mysdk", "",
		checkUnversionedAndroidBpContents(`
// This is auto-generated. DO NOT EDIT.

java_test_import {
    name: "myjavalib",
    prefer: false,
    visibility: ["//visibility:public"],
    apex_available: ["//apex_available:platform"],
    jars: ["javalibs/myjavalib.jar"],
}

java_import {
    name: "myjavalib_test",
    prefer: false,
    visibility: ["//visibility:public"],
    apex_available: ["//apex_available:platform"],
    jars: ["javalibs/myjavalib_test.jar"],
}
`),
		checkVersionedAndroidBpContents(`
// This is auto-generated. DO NOT EDIT.

java_test_import {
    name: "mysdk_myjavalib@current",
    sdk_member_name: "myjavalib",
    visibility: ["//visibility:public"],
    apex_available: ["//apex_available:platform"],
    jars: ["javalibs/myjavalib.jar"],
}

java_import {
    name: "mysdk_myjavalib_test@current",
    sdk_member_name: "myjavalib_test",
    visibility: ["//visibility:public"],
    apex_available: ["//apex_available:platform"],
    jars: ["javalibs/myjavalib_test.jar"],
}

sdk_snapshot {
    name: "mysdk@current",
    visibility: ["//visibility:public"],
    test_members: [
        "mysdk_myjavalib@current",
        "mysdk_myjavalib_test@current",
    ],
}
`),
	)
}

func TestUnsupportedTrait(t *testing.T) {
	android.GroupFixturePreparers(
		prepareForSdkTestWithJava,
		android.FixtureWithRootAndroidBp(`
			sdk {
				name: "mysdk",
				java_header_libs: ["myjavalib"],
				traits: {
					test: ["myjavalib"],
				},
			}
	
			java_library {
				name: "myjavalib",
				srcs: ["Test.java"],
				system_modules: "none",
				sdk_version: "none",
			}
		`),
	).ExtendWithErrorHandler(android.FixtureExpectsAtLeastOneErrorMatchingPattern(`\Qsdk member "myjavalib" has traits [test] that are unsupported by its member type "java_header_libs"\E`)).
		RunTest(t)
}
