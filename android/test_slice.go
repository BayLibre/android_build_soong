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

package android

import "android/soong/ui/metrics/bp2build_metrics_proto"

func init() {
	RegisterTestSliceBuildComponents(InitRegistrationContext)
}

// Register the license_kind module type.
func RegisterTestSliceBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("test_slice", TestSliceFactory)
}

type testSliceProperties struct {
	Orig                string
	Include_filters     []string
	Exclude_filters     []string
	Include_annotations []string
	Exclude_annotations []string
}

var _ Bazelable = &testSliceModule{}

type testSliceModule struct {
	ModuleBase
	DefaultableModuleBase
	BazelModuleBase

	properties testSliceProperties
}

type bazelTestSliceAttributes struct {
	Orig                string
	Include_filters     []string
	Exclude_filters     []string
	Include_annotations []string
	Exclude_annotations []string
}

func (m *testSliceModule) ConvertWithBp2build(ctx Bp2buildMutatorContext) {
	ctx.MarkBp2buildUnconvertible(bp2build_metrics_proto.UnconvertedReasonType_TYPE_UNSUPPORTED, "na")
	return
	/*
		attrs := &bazelLicenseKindAttributes{
			Conditions: m.properties.Conditions,
			Url:        m.properties.Url,
			Visibility: m.properties.Visibility,
		}
		ctx.CreateBazelTargetModule(
			bazel.BazelTargetModuleProperties{
				Rule_class:        "license_kind",
				Bzl_load_location: "@rules_license//rules:license_kind.bzl",
			},
			CommonAttributes{
				Name: m.Name(),
			},
			attrs)
	*/
}

func (m *testSliceModule) DepsMutator(ctx BottomUpMutatorContext) {
	// Nothing to do.
}

func (m *testSliceModule) GenerateAndroidBuildActions(ModuleContext) {
	// Nothing to do.
	// TODO(ron): yet
}

func TestSliceFactory() Module {
	module := &testSliceModule{}

	base := module.base()
	module.AddProperties(&base.nameProperties, &module.properties, &base.commonProperties.BazelConversionStatus)

	InitAndroidModule(module)
	InitDefaultableModule(module)
	InitBazelModule(module)

	return module
}
