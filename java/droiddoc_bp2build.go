// Copyright 2023 Google Inc. All rights reserved.
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
	"android/soong/android"
	"android/soong/bazel"
	"android/soong/ui/metrics/bp2build_metrics_proto"

	"github.com/google/blueprint/proptools"
)

type javadocAttributes struct {
	Srcs            bazel.LabelListAttribute
	Filter_packages []string

	Deps           bazel.LabelListAttribute
	System_modules *bazel.Label

	Sdk_version  *string
	Java_version *string

	Flags             []string
	Additional_inputs bazel.LabelList
	Out               []string
}

func (j *Javadoc) propsToAttrs(ctx android.Bp2buildMutatorContext) (javadocAttributes, bool) {
	var attrs javadocAttributes
	if proptools.String(j.properties.Sdk_version) == "" {
		ctx.MarkBp2buildUnconvertible(bp2build_metrics_proto.UnconvertedReasonType_PROPERTY_UNSUPPORTED, "sdk_version unset")
		return attrs, false
	} else if proptools.String(j.properties.Sdk_version) == "" {
		ctx.MarkBp2buildUnconvertible(bp2build_metrics_proto.UnconvertedReasonType_PROPERTY_UNSUPPORTED, "sdk_version core_platform")
		return attrs, false
	}

	attrs.Sdk_version = j.properties.Sdk_version
	attrs.Java_version = j.properties.Java_version

	var srcs bazel.LabelListAttribute
	var deps bazel.LabelListAttribute
	archVariantProps := j.GetArchVariantProperties(ctx, &JavadocProperties{})
	for axis, configToProps := range archVariantProps {
		for config, p := range configToProps {
			if props, ok := p.(*JavadocProperties); ok {
				s := android.BazelLabelForModuleSrcExcludes(ctx, props.Srcs, props.Exclude_srcs)
				srcs.SetSelectValue(axis, config, s)
				deps.SetSelectValue(axis, config, android.BazelLabelForModuleDeps(ctx, props.Libs))
			}
		}
	}

	return attrs, true
}
