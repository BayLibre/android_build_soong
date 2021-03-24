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

import (
	"fmt"
	"sort"
)

// BazelTargetModuleProperties contain properties and metadata used for
// Blueprint to BUILD file conversion.
type BazelTargetModuleProperties struct {
	// The Bazel rule class for this target.
	Rule_class string `blueprint:"mutated"`

	// The target label for the bzl file containing the definition of the rule class.
	Bzl_load_location string `blueprint:"mutated"`
}

const BazelTargetModuleNamePrefix = "__bp2build__"

// Label is used to represent a Bazel compatible Label. Also stores the original bp text to support
// string replacement.
type Label struct {
	Bp_text string
	Label   string
}

// LabelList is used to represent a list of Bazel labels.
type LabelListAttribute struct {
	Value LabelList

	ArchValues labelListArchValues
}

// LabelList is a basic data structure for label lists.
type LabelList struct {
	Includes []Label
	Excludes []Label
}

// Arch-specific label_list typed Bazel attribute values. This should correspond
// to the types of architectures supported for compilation in arch.go.
type labelListArchValues struct {
	X86     LabelList
	X86_64  LabelList
	Arm     LabelList
	Arm64   LabelList
	Default LabelList
}

// Append appends the fields of other labelList to the corresponding fields of ll.
func (ll *LabelList) Append(other LabelList) {
	if len(ll.Includes) > 0 || len(other.Includes) > 0 {
		ll.Includes = append(ll.Includes, other.Includes...)
	}
	if len(ll.Excludes) > 0 || len(other.Excludes) > 0 {
		ll.Excludes = append(other.Excludes, other.Excludes...)
	}
}

func UniqueBazelLabels(originalLabels []Label) []Label {
	uniqueLabelsSet := make(map[Label]bool)
	for _, l := range originalLabels {
		uniqueLabelsSet[l] = true
	}
	var uniqueLabels []Label
	for l, _ := range uniqueLabelsSet {
		uniqueLabels = append(uniqueLabels, l)
	}
	sort.SliceStable(uniqueLabels, func(i, j int) bool {
		return uniqueLabels[i].Label < uniqueLabels[j].Label
	})
	return uniqueLabels
}

func UniqueBazelLabelList(originalLabelList LabelList) LabelList {
	var uniqueLabelList LabelList
	uniqueLabelList.Includes = UniqueBazelLabels(originalLabelList.Includes)
	uniqueLabelList.Excludes = UniqueBazelLabels(originalLabelList.Excludes)
	return uniqueLabelList
}

// HasArchSpecificValues returns true if the attribute contains
// architecture-specific label_list values.
func (attrs *LabelListAttribute) HasArchSpecificValues() bool {
	for _, arch := range []string{"x86", "x86_64", "arm", "arm64", "default"} {
		if len(attrs.GetValueForArch(arch).Includes) > 0 || len(attrs.GetValueForArch(arch).Excludes) > 0 {
			return true
		}
	}
	return false
}

// GetValueForArch returns the label_list attribute value for an architecture.
func (attrs *LabelListAttribute) GetValueForArch(arch string) LabelList {
	switch arch {
	case "x86":
		return attrs.ArchValues.X86
	case "x86_64":
		return attrs.ArchValues.X86_64
	case "arm":
		return attrs.ArchValues.Arm
	case "arm64":
		return attrs.ArchValues.Arm64
	case "default":
		return attrs.ArchValues.Default
	default:
		panic(fmt.Errorf("Unknown arch: %s", arch))
	}
}

func (attrs *LabelListAttribute) HasTargetSpecificValues() bool {
	for _, os := range []string{"android", "linux_bionic"} {
		if len(attrs.GetValueForTarget(os).Includes) > 0 || len(attrs.GetValueForTarget(os).Excludes) > 0 {
			return true
		}
	}
	return false
}

// SetValueForArch sets the label_list attribute value for an OS target.
func (attrs *LabelListAttribute) GetValueForTarget(target string) LabelList {
	ret := LabelList{}
	switch target {
	case "linux_glibc":
		ret.Append(attrs.ArchValues.X86)
		ret.Append(attrs.ArchValues.X86_64)
	case "darwin":
		ret.Append(attrs.ArchValues.X86)
		ret.Append(attrs.ArchValues.X86_64)
	case "linux_bionic":
		ret.Append(attrs.ArchValues.Arm64)
		ret.Append(attrs.ArchValues.X86_64)
	case "windows":
		ret.Append(attrs.ArchValues.X86)
		ret.Append(attrs.ArchValues.X86_64)
	case "android":
		ret.Append(attrs.ArchValues.X86)
		ret.Append(attrs.ArchValues.X86_64)
		ret.Append(attrs.ArchValues.Arm)
		ret.Append(attrs.ArchValues.Arm64)
	case "fuchsia":
		ret.Append(attrs.ArchValues.Arm64)
		ret.Append(attrs.ArchValues.X86_64)
	case "common_os":
		ret.Append(attrs.ArchValues.Default)
	case "default":
		ret.Append(attrs.ArchValues.Default)
	default:
		panic(fmt.Errorf("Unknown target: %s", target))
	}
	return ret
}

// SetValueForArch sets the label_list attribute value for an OS target.
func (attrs *LabelListAttribute) SetValueForTarget(target string, value LabelList) {
	switch target {
	case "linux_glibc":
		attrs.ArchValues.X86 = value
		attrs.ArchValues.X86_64 = value
	case "darwin":
		attrs.ArchValues.X86 = value
		attrs.ArchValues.X86_64 = value
	case "linux_bionic":
		attrs.ArchValues.Arm64 = value
		attrs.ArchValues.X86_64 = value
	case "windows":
		attrs.ArchValues.X86 = value
		attrs.ArchValues.X86_64 = value
	case "android":
		attrs.ArchValues.X86 = value
		attrs.ArchValues.X86_64 = value
		attrs.ArchValues.Arm = value
		attrs.ArchValues.Arm64 = value
	case "fuchsia":
		attrs.ArchValues.Arm64 = value
		attrs.ArchValues.X86_64 = value
	case "common_os":
		attrs.ArchValues.Default = value
	case "default":
		attrs.ArchValues.Default = value
	default:
		panic(fmt.Errorf("Unknown target: %s", target))
	}
}

// SetValueForArch sets the label_list attribute value for an architecture.
func (attrs *LabelListAttribute) SetValueForArch(arch string, value LabelList) {
	switch arch {
	case "x86":
		attrs.ArchValues.X86 = value
	case "x86_64":
		attrs.ArchValues.X86_64 = value
	case "arm":
		attrs.ArchValues.Arm = value
	case "arm64":
		attrs.ArchValues.Arm64 = value
	case "default":
		attrs.ArchValues.Default = value
	default:
		panic(fmt.Errorf("Unknown arch: %s", arch))
	}
}

func MakeStringListAttribute(value []string) StringListAttribute {
	return StringListAttribute{Value: value}
}

// StringListAttribute corresponds to the string_list Bazel attribute type with
// support for additional metadata, like configurations.
type StringListAttribute struct {
	// The base value of the string list attribute.
	Value []string

	// Optional additive set of list values to the base value.
	ArchValues stringListArchValues
}

// Arch-specific string_list typed Bazel attribute values. This should correspond
// to the types of architectures supported for compilation in arch.go.
type stringListArchValues struct {
	X86     []string
	X86_64  []string
	Arm     []string
	Arm64   []string
	Default []string
	// TODO(b/181299724): this is currently missing the "common" arch, which
	// doesn't have an equivalent platform() definition yet.
}

// HasArchSpecificValues returns true if the attribute contains
// architecture-specific string_list values.
func (attrs *StringListAttribute) HasArchSpecificValues() bool {
	for _, arch := range []string{"x86", "x86_64", "arm", "arm64", "default"} {
		if len(attrs.GetValueForArch(arch)) > 0 {
			return true
		}
	}
	return false
}

// GetValueForArch returns the string_list attribute value for an architecture.
func (attrs *StringListAttribute) GetValueForArch(arch string) []string {
	switch arch {
	case "x86":
		return attrs.ArchValues.X86
	case "x86_64":
		return attrs.ArchValues.X86_64
	case "arm":
		return attrs.ArchValues.Arm
	case "arm64":
		return attrs.ArchValues.Arm64
	case "default":
		return attrs.ArchValues.Default
	default:
		panic(fmt.Errorf("Unknown arch: %s", arch))
	}
}

// SetValueForArch sets the string_list attribute value for an architecture.
func (attrs *StringListAttribute) SetValueForArch(arch string, value []string) {
	switch arch {
	case "x86":
		attrs.ArchValues.X86 = value
	case "x86_64":
		attrs.ArchValues.X86_64 = value
	case "arm":
		attrs.ArchValues.Arm = value
	case "arm64":
		attrs.ArchValues.Arm64 = value
	case "default":
		attrs.ArchValues.Default = value
	default:
		panic(fmt.Errorf("Unknown arch: %s", arch))
	}
}
