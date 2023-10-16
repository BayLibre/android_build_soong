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
)

type droidstubsAttributes struct {
	javadocAttributes

	Api_filename         *bazel.Label
	Removed_api_filename *bazel.Label

	Previous_api *bazel.Label

	Merge_annotations_dirs bazel.LabelList

	Merge_inclusion_annotations_dirs bazel.LabelList

	Annotations_enabled *bool

	Validate_nullability_from_list *bazel.Label
	Check_nullability_warnings     *bazel.Label // TODO

	Generate_stubs          *bool
	Create_doc_stubs        *bool
	Output_javadoc_comments *bool

	Api_levels_annotations_enabled *bool
	Api_levels_module              *bazel.Label
	Api_levels_annotations_dirs    bazel.LabelList
	Api_levels_sdk_type            *string
	Api_levels_jar_filename        *bazel.Label

	Write_sdk_values *bool

	Extensions_info_file *bazel.Label

	// TODO Check_api
}

func (d *Droidstubs) ConvertWithBp2build(ctx android.Bp2buildMutatorContext) {
}
