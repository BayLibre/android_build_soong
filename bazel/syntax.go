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

package bazel

// Syntax abstractions for the Bazel BUILD language.

// A struct representing a Bazel label list.
type LabelList struct {
	Labels []Label
}

// NewLabelList creates a list of Labels from a list of Strings.
func NewLabelList(labelStrings []string) LabelList {
	var labels []Label
	for _, s := range labelStrings {
		labels = append(labels, Label(s))
	}
	return LabelList{Labels: labels}
}

type Glob struct {
	Include             LabelList
	Exclude             LabelList
	Allow_empty         bool
	Exclude_directories bool
}

// NewGlob creates a new Glob with default values applied.
func NewGlob(include LabelList, exclude LabelList) Glob {
	return Glob{
		Include:             include,
		Exclude:             exclude,
		Allow_empty:         true,
		Exclude_directories: true,
	}
}

// Labels are used to identify a Target or a Source/Derived file.
//
// Simple type alias to special case Labels from regular strings.
// In the future, we'll want to model this after Bazel's Label, which
// seperates the package identifier from the target label name itself.
type Label string

func (l Label) String() string {
	return string(l)
}
