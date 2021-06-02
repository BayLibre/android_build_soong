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
	"path/filepath"
	"regexp"
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

var productVariableSubstitutionPattern = regexp.MustCompile("%(d|s)")

// Label is used to represent a Bazel compatible Label. Also stores the original
// bp text to support string replacement.
type Label struct {
	// The string representation of a Bazel target label. This can be a relative
	// or fully qualified label. These labels are used for generating BUILD
	// files with bp2build.
	Label string

	// The original Soong/Blueprint module name that the label was derived from.
	// This is used for replacing references to the original name with the new
	// label, for example in genrule cmds.
	//
	// While there is a reversible 1:1 mapping from the module name to Bazel
	// label with bp2build that could make computing the original module name
	// from the label automatic, it is not the case for handcrafted targets,
	// where modules can have a custom label mapping through the { bazel_module:
	// { label: <label> } } property.
	//
	// With handcrafted labels, those modules don't go through bp2build
	// conversion, but relies on handcrafted targets in the source tree.
	OriginalModuleName string
}

// LabelList is used to represent a list of Bazel labels.
type LabelList struct {
	Includes []Label
	Excludes []Label
}

func (ll *LabelList) deepCopy() LabelList {
	return LabelList{
		Includes: ll.Includes[:],
		Excludes: ll.Excludes[:],
	}
}

// uniqueParentDirectories returns a list of the unique parent directories for
// all files in ll.Includes.
func (ll *LabelList) uniqueParentDirectories() []string {
	dirMap := map[string]bool{}
	for _, label := range ll.Includes {
		dirMap[filepath.Dir(label.Label)] = true
	}
	dirs := []string{}
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}
	return dirs
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

// UniqueSortedBazelLabels takes a []Label and deduplicates the labels, and returns
// the slice in a sorted order.
func UniqueSortedBazelLabels(originalLabels []Label) []Label {
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
	uniqueLabelList.Includes = UniqueSortedBazelLabels(originalLabelList.Includes)
	uniqueLabelList.Excludes = UniqueSortedBazelLabels(originalLabelList.Excludes)
	return uniqueLabelList
}

// Subtract needle from haystack
func SubtractStrings(haystack []string, needle []string) []string {
	// This is really a set
	remainder := make(map[string]bool)

	for _, s := range haystack {
		remainder[s] = true
	}
	for _, s := range needle {
		delete(remainder, s)
	}

	var strings []string
	for s, _ := range remainder {
		strings = append(strings, s)
	}

	sort.SliceStable(strings, func(i, j int) bool {
		return strings[i] < strings[j]
	})

	return strings
}

// Return all needles in a given haystack, where needleFn is true for needles.
func FilterLabelList(haystack LabelList, needleFn func(string) bool) LabelList {
	var includes []Label
	for _, inc := range haystack.Includes {
		if needleFn(inc.Label) {
			includes = append(includes, inc)
		}
	}
	return LabelList{Includes: includes, Excludes: haystack.Excludes}
}

// Return all needles in a given haystack, where needleFn is true for needles.
func FilterLabelListAttribute(haystack LabelListAttribute, needleFn func(string) bool) LabelListAttribute {
	result := MakeLabelListAttribute(FilterLabelList(haystack.Value, needleFn))

	for config, selects := range haystack.ConfigurableValues {
		newSelects := make(labelListSelectValues, len(selects))
		for k, v := range selects {
			newSelects[k] = FilterLabelList(v, needleFn)
		}
		result.ConfigurableValues[config] = newSelects
	}

	return result
}

// Subtract needle from haystack
func SubtractBazelLabelListAttribute(haystack LabelListAttribute, needle LabelListAttribute) LabelListAttribute {
	result := MakeLabelListAttribute(SubtractBazelLabelList(haystack.Value, needle.Value))

	for config, selects := range haystack.ConfigurableValues {
		newSelects := make(labelListSelectValues, len(selects))
		needleSelects := needle.ConfigurableValues[config]

		for k, v := range selects {
			newSelects[k] = SubtractBazelLabelList(v, needleSelects[k])
		}
		result.ConfigurableValues[config] = newSelects
	}

	return result
}

// Subtract needle from haystack
func SubtractBazelLabels(haystack []Label, needle []Label) []Label {
	// This is really a set
	remainder := make(map[Label]bool)

	for _, label := range haystack {
		remainder[label] = true
	}
	for _, label := range needle {
		delete(remainder, label)
	}

	var labels []Label
	for label, _ := range remainder {
		labels = append(labels, label)
	}

	sort.SliceStable(labels, func(i, j int) bool {
		return labels[i].Label < labels[j].Label
	})

	return labels
}

// Appends two LabelLists, returning the combined list.
func AppendBazelLabelLists(a LabelList, b LabelList) LabelList {
	var result LabelList
	result.Includes = append(a.Includes, b.Includes...)
	result.Excludes = append(a.Excludes, b.Excludes...)
	return result
}

// Subtract needle from haystack
func SubtractBazelLabelList(haystack LabelList, needle LabelList) LabelList {
	var result LabelList
	result.Includes = SubtractBazelLabels(haystack.Includes, needle.Includes)
	// NOTE: Excludes are intentionally not subtracted
	result.Excludes = haystack.Excludes
	return result
}

const (
	// ArchType names in arch.go
	ARCH_ARM    = "arm"
	ARCH_ARM64  = "arm64"
	ARCH_X86    = "x86"
	ARCH_X86_64 = "x86_64"

	// OsType names in arch.go
	OS_ANDROID      = "android"
	OS_DARWIN       = "darwin"
	OS_FUCHSIA      = "fuchsia"
	OS_LINUX        = "linux_glibc"
	OS_LINUX_BIONIC = "linux_bionic"
	OS_WINDOWS      = "windows"

	// Targets in arch.go
	TARGET_ANDROID_ARM         = "android_arm"
	TARGET_ANDROID_ARM64       = "android_arm64"
	TARGET_ANDROID_X86         = "android_x86"
	TARGET_ANDROID_X86_64      = "android_x86_64"
	TARGET_DARWIN_X86_64       = "darwin_x86_64"
	TARGET_FUCHSIA_ARM64       = "fuchsia_arm64"
	TARGET_FUCHSIA_X86_64      = "fuchsia_x86_64"
	TARGET_LINUX_X86           = "linux_glibc_x86"
	TARGET_LINUX_x86_64        = "linux_glibc_x86_64"
	TARGET_LINUX_BIONIC_ARM64  = "linux_bionic_arm64"
	TARGET_LINUX_BIONIC_X86_64 = "linux_bionic_x86_64"
	TARGET_WINDOWS_X86         = "windows_x86"
	TARGET_WINDOWS_X86_64      = "windows_x86_64"

	// This is the string representation of the default condition wherever a
	// configurable attribute is used in a select statement, i.e.
	// //conditions:default for Bazel.
	//
	// This is consistently named "conditions_default" to mirror the Soong
	// config variable default key in an Android.bp file, although there's no
	// integration with Soong config variables (yet).
	CONDITIONS_DEFAULT = "conditions_default"

	ConditionsDefaultSelectKey = "//conditions:default"

	productVariableBazelPackage = "//build/bazel/product_variables"
)

var (
	// These are the list of OSes and architectures with a Bazel config_setting
	// and constraint value equivalent. These exist in arch.go, but the android
	// package depends on the bazel package, so a cyclic dependency prevents
	// using those variables here.

	// A map of architectures to the Bazel label of the constraint_value
	// for the @platforms//cpu:cpu constraint_setting
	PlatformArchMap = map[string]string{
		ARCH_ARM:           "//build/bazel/platforms/arch:arm",
		ARCH_ARM64:         "//build/bazel/platforms/arch:arm64",
		ARCH_X86:           "//build/bazel/platforms/arch:x86",
		ARCH_X86_64:        "//build/bazel/platforms/arch:x86_64",
		CONDITIONS_DEFAULT: ConditionsDefaultSelectKey, // The default condition of as arch select map.
	}

	// A map of target operating systems to the Bazel label of the
	// constraint_value for the @platforms//os:os constraint_setting
	PlatformOsMap = map[string]string{
		OS_ANDROID:         "//build/bazel/platforms/os:android",
		OS_DARWIN:          "//build/bazel/platforms/os:darwin",
		OS_FUCHSIA:         "//build/bazel/platforms/os:fuchsia",
		OS_LINUX:           "//build/bazel/platforms/os:linux",
		OS_LINUX_BIONIC:    "//build/bazel/platforms/os:linux_bionic",
		OS_WINDOWS:         "//build/bazel/platforms/os:windows",
		CONDITIONS_DEFAULT: ConditionsDefaultSelectKey, // The default condition of an os select map.
	}

	PlatformTargetMap = map[string]string{
		TARGET_ANDROID_ARM:         "//build/bazel/platforms/os_arch:android_arm",
		TARGET_ANDROID_ARM64:       "//build/bazel/platforms/os_arch:android_arm64",
		TARGET_ANDROID_X86:         "//build/bazel/platforms/os_arch:android_x86",
		TARGET_ANDROID_X86_64:      "//build/bazel/platforms/os_arch:android_x86_64",
		TARGET_DARWIN_X86_64:       "//build/bazel/platforms/os_arch:darwin_x86_64",
		TARGET_FUCHSIA_ARM64:       "//build/bazel/platforms/os_arch:fuchsia_arm64",
		TARGET_FUCHSIA_X86_64:      "//build/bazel/platforms/os_arch:fuchsia_x86_64",
		TARGET_LINUX_X86:           "//build/bazel/platforms/os_arch:linux_glibc_x86",
		TARGET_LINUX_x86_64:        "//build/bazel/platforms/os_arch:linux_glibc_x86_64",
		TARGET_LINUX_BIONIC_ARM64:  "//build/bazel/platforms/os_arch:linux_bionic_arm64",
		TARGET_LINUX_BIONIC_X86_64: "//build/bazel/platforms/os_arch:linux_bionic_x86_64",
		TARGET_WINDOWS_X86:         "//build/bazel/platforms/os_arch:windows_x86",
		TARGET_WINDOWS_X86_64:      "//build/bazel/platforms/os_arch:windows_x86_64",
		CONDITIONS_DEFAULT:         ConditionsDefaultSelectKey, // The default condition of an os select map.
	}

	// TODO(b/187530594): Should we add CONDITIONS_DEFAULT here?
	AllArches = []string{ARCH_ARM, ARCH_ARM64, ARCH_X86, ARCH_X86_64}
)

type Attribute interface {
	HasConfigurableValues() bool
}

type labelSelectValues map[string]*Label

type configurableLabels map[ConfigurationAxis]labelSelectValues

func (cl configurableLabels) setValueForAxis(axis ConfigurationAxis, config string, value *Label) {
	if cl[axis] == nil {
		cl[axis] = make(labelSelectValues)
	}
	cl[axis][config] = value
}

// Represents an attribute whose value is a single label
type LabelAttribute struct {
	Value *Label

	ConfigurableValues configurableLabels
}

func (la LabelAttribute) HasConfigurableValues() bool {
	return len(la.ConfigurableValues) > 0
}

func (la *LabelAttribute) SetValue(value Label) {
	la.SetSelectValue(NoConfigAxis, "", value)
}

func (la *LabelAttribute) SetSelectValue(axis ConfigurationAxis, config string, value Label) {
	axis.validateConfig(config)
	switch axis.configurationType {
	case NoConfig:
		la.Value = &value
	case Arch, Os, ArchOs, ProductVariables:
		if la.ConfigurableValues == nil {
			la.ConfigurableValues = make(configurableLabels)
		}
		la.ConfigurableValues.setValueForAxis(axis, config, &value)
	default:
		panic(fmt.Errorf("Unrecognized ConfigurationAxis %s", axis))
	}
}

func (la *LabelAttribute) SelectValue(axis ConfigurationAxis, config string) Label {
	axis.validateConfig(config)
	switch axis.configurationType {
	case NoConfig:
		return *la.Value
	case Arch, Os, ArchOs, ProductVariables:
		return *la.ConfigurableValues[axis][config]
	default:
		panic(fmt.Errorf("Unrecognized ConfigurationAxis %s", axis))
	}
}

func (la *LabelAttribute) SortedConfigurationAxes() []ConfigurationAxis {
	keys := make([]ConfigurationAxis, 0, len(la.ConfigurableValues))
	for k := range la.ConfigurableValues {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i].less(keys[j]) })
	return keys
}

// labelListSelectValues supports config-specific label_list typed Bazel attribute values.
type labelListSelectValues map[string]LabelList

func (ll labelListSelectValues) appendSelects(other labelListSelectValues) {
	for k, v := range other {
		l := ll[k]
		(&l).Append(v)
		ll[k] = l
	}
}

func (ll labelListSelectValues) HasConfigurableValues() bool {
	for _, v := range ll {
		if len(v.Includes) > 0 {
			return true
		}
	}
	return false
}

// LabelListAttribute is used to represent a list of Bazel labels as an
// attribute.
type LabelListAttribute struct {
	// The non-configured attribute label list Value. Required.
	Value LabelList

	// The configured attribute label list Values. Optional
	// a map of independent configurability axes
	ConfigurableValues configurableLabelLists
}

type configurableLabelLists map[ConfigurationAxis]labelListSelectValues

func (cll configurableLabelLists) setValueForAxis(axis ConfigurationAxis, config string, list LabelList) {
	if cll[axis] == nil {
		cll[axis] = make(labelListSelectValues)
	}
	cll[axis][config] = list
}

func (cll configurableLabelLists) Append(other configurableLabelLists) {
	for axis, otherSelects := range other {
		selects := cll[axis]
		if selects == nil {
			selects = make(labelListSelectValues, len(otherSelects))
		}
		selects.appendSelects(otherSelects)
		cll[axis] = selects
	}
}

// MakeLabelListAttribute initializes a LabelListAttribute with the non-arch specific value.
func MakeLabelListAttribute(value LabelList) LabelListAttribute {
	return LabelListAttribute{
		Value:              UniqueBazelLabelList(value),
		ConfigurableValues: make(configurableLabelLists),
	}
}

func (lla *LabelListAttribute) SetValue(list LabelList) {
	lla.SetSelectValue(NoConfigAxis, "", list)
}

func (lla *LabelListAttribute) SetSelectValue(axis ConfigurationAxis, config string, list LabelList) {
	axis.validateConfig(config)
	switch axis.configurationType {
	case NoConfig:
		lla.Value = list
	case Arch, Os, ArchOs, ProductVariables:
		if lla.ConfigurableValues == nil {
			lla.ConfigurableValues = make(configurableLabelLists)
		}
		lla.ConfigurableValues.setValueForAxis(axis, config, list)
	default:
		panic(fmt.Errorf("Unrecognized ConfigurationAxis %s", axis))
	}
}

func (lla *LabelListAttribute) SelectValue(axis ConfigurationAxis, config string) LabelList {
	axis.validateConfig(config)
	switch axis.configurationType {
	case NoConfig:
		return lla.Value
	case Arch, Os, ArchOs, ProductVariables:
		return lla.ConfigurableValues[axis][config]
	default:
		panic(fmt.Errorf("Unrecognized ConfigurationAxis %s", axis))
	}
}

func (lla *LabelListAttribute) SortedConfigurationAxes() []ConfigurationAxis {
	keys := make([]ConfigurationAxis, 0, len(lla.ConfigurableValues))
	for k := range lla.ConfigurableValues {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i].less(keys[j]) })
	return keys
}

// Append all values, including os and arch specific ones, from another
// LabelListAttribute to this LabelListAttribute.
func (lla *LabelListAttribute) Append(other LabelListAttribute) {
	lla.Value.Append(other.Value)
	if lla.ConfigurableValues == nil {
		lla.ConfigurableValues = make(configurableLabelLists)
	}
	lla.ConfigurableValues.Append(other.ConfigurableValues)
}

// HasArchSpecificValues returns true if the attribute contains
// architecture-specific label_list values.
func (lla LabelListAttribute) HasConfigurableValues() bool {
	return len(lla.ConfigurableValues) > 0
}

func (lla *LabelListAttribute) ResolveExcludes() {
	for axis, configToLabels := range lla.ConfigurableValues {
		baseLabels := lla.Value.deepCopy()
		for config, val := range configToLabels {
			allLabels := baseLabels.deepCopy()
			allLabels.Append(val)
			lla.Value = SubtractBazelLabelList(lla.Value, LabelList{Includes: val.Excludes})
			allLabels = SubtractBazelLabelList(allLabels, LabelList{Includes: allLabels.Excludes})
			lla.ConfigurableValues[axis][config] = allLabels
		}

		// After going through all configs, delete the duplicate files for the config
		// values that are already in the base srcs.Value.
		for config, val := range configToLabels {
			lla.ConfigurableValues[axis][config] = SubtractBazelLabelList(val, lla.Value)
		}

		// Now that the Value list is finalized for this axis, compare it with the original
		// list, and put the difference into the default condition for the arch
		// select.
		lla.ConfigurableValues[axis][CONDITIONS_DEFAULT] = SubtractBazelLabelList(baseLabels, lla.Value)
	}
}

// StringListAttribute corresponds to the string_list Bazel attribute type with
// support for additional metadata, like configurations.
type StringListAttribute struct {
	// The base value of the string list attribute.
	Value []string

	// The configured attribute label list Values. Optional
	// a map of independent configurability axes
	ConfigurableValues configurableStringLists
}

type configurableStringLists map[ConfigurationAxis]stringListSelectValues

func (csl configurableStringLists) Append(other configurableStringLists) {
	for axis, otherSelects := range other {
		selects := csl[axis]
		if selects == nil {
			selects = make(stringListSelectValues, len(otherSelects))
		}
		selects.appendSelects(otherSelects)
		csl[axis] = selects
	}
}

func (csl configurableStringLists) setValueForAxis(axis ConfigurationAxis, config string, list []string) {
	if csl[axis] == nil {
		csl[axis] = make(stringListSelectValues)
	}
	csl[axis][config] = list
}

type stringListSelectValues map[string][]string

func (sl stringListSelectValues) appendSelects(other stringListSelectValues) {
	for k, v := range other {
		sl[k] = append(sl[k], v...)
	}
}

func (sl stringListSelectValues) hasConfigurableValues(other stringListSelectValues) bool {
	for _, val := range sl {
		if len(val) > 0 {
			return true
		}
	}
	return false
}

// MakeStringListAttribute initializes a StringListAttribute with the non-arch specific value.
func MakeStringListAttribute(value []string) StringListAttribute {
	// NOTE: These strings are not necessarily unique or sorted.
	return StringListAttribute{
		Value:              value,
		ConfigurableValues: make(configurableStringLists),
	}
}

// HasConfigurableValues returns true if the attribute contains
// architecture-specific string_list values.
func (sla StringListAttribute) HasConfigurableValues() bool {
	return len(sla.ConfigurableValues) > 0
}

// Append appends all values, including os and arch specific ones, from another
// StringListAttribute to this StringListAttribute
func (sla *StringListAttribute) Append(other StringListAttribute) {
	sla.Value = append(sla.Value, other.Value...)
	if sla.ConfigurableValues == nil {
		sla.ConfigurableValues = make(configurableStringLists)
	}
	sla.ConfigurableValues.Append(other.ConfigurableValues)
}

func (sla *StringListAttribute) SetSelectValue(axis ConfigurationAxis, config string, list []string) {
	axis.validateConfig(config)
	switch axis.configurationType {
	case NoConfig:
		sla.Value = list
	case Arch, Os, ArchOs, ProductVariables:
		if sla.ConfigurableValues == nil {
			sla.ConfigurableValues = make(configurableStringLists)
		}
		sla.ConfigurableValues.setValueForAxis(axis, config, list)
	default:
		panic(fmt.Errorf("Unrecognized ConfigurationAxis %s", axis))
	}
}

func (sla *StringListAttribute) SelectValue(axis ConfigurationAxis, config string) []string {
	axis.validateConfig(config)
	switch axis.configurationType {
	case NoConfig:
		return sla.Value
	case Arch, Os, ArchOs, ProductVariables:
		return sla.ConfigurableValues[axis][config]
	default:
		panic(fmt.Errorf("Unrecognized ConfigurationAxis %s", axis))
	}
}

func (sla *StringListAttribute) SortedConfigurationAxes() []ConfigurationAxis {
	keys := make([]ConfigurationAxis, 0, len(sla.ConfigurableValues))
	for k := range sla.ConfigurableValues {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i].less(keys[j]) })
	return keys
}

// TryVariableSubstitution, replace string substitution formatting within each string in slice with
// Starlark string.format compatible tag for productVariable.
func TryVariableSubstitutions(slice []string, productVariable string) ([]string, bool) {
	ret := make([]string, 0, len(slice))
	changesMade := false
	for _, s := range slice {
		newS, changed := TryVariableSubstitution(s, productVariable)
		ret = append(ret, newS)
		changesMade = changesMade || changed
	}
	return ret, changesMade
}

// TryVariableSubstitution, replace string substitution formatting within s with Starlark
// string.format compatible tag for productVariable.
func TryVariableSubstitution(s string, productVariable string) (string, bool) {
	sub := productVariableSubstitutionPattern.ReplaceAllString(s, "$("+productVariable+")")
	return sub, s != sub
}
