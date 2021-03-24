package bp2build

import (
	"android/soong/android"
	"android/soong/bazel"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Configurability support for bp2build.

var (
	// A map of architectures to the Bazel label of the constraint_value
	// for the @platforms//cpu:cpu constraint_setting
	platformArchMap = map[android.ArchType]string{
		android.Arm:    "@bazel_tools//platforms:arm",
		android.Arm64:  "@bazel_tools//platforms:aarch64",
		android.X86:    "@bazel_tools//platforms:x86_32",
		android.X86_64: "@bazel_tools//platforms:x86_64",
	}

	// A map of target operating systems to the Bazel label of the
	// constraint_value for the @platforms//os:os constraint_setting
	platformOsMap = map[android.OsType]string{
		android.Android:     "@bazel_tools//platforms:android",
		android.CommonOS:    "@bazel_tools//platforms:android",
		android.Darwin:      "@bazel_tools//platforms:osx",
		android.Linux:       "@bazel_tools//platforms:linux",
		android.LinuxBionic: "@bazel_tools//platforms:linux",
		android.Windows:     "@bazel_tools//platforms:windows",
		// Not listed: Fuchsia, NoOsType
	}
)

func prettyPrintStringListAttribute(stringList bazel.StringListAttribute, indent int) (string, error) {
	// A Bazel string_list attribute that may contain a select statement.
	ret, err := prettyPrint(reflect.ValueOf(stringList.Value), indent)
	if err != nil {
		return ret, err
	}

	if !stringList.HasArchSpecificValues() {
		// Select statement not needed.
		return ret, nil
	}

	ret += " + select({\n"
	var selects []string
	for arch, selectKey := range platformArchMap {
		value := stringList.GetValueForArch(arch.Name)
		if len(value) > 0 {
			s := makeIndent(indent + 1)
			list, _ := prettyPrint(reflect.ValueOf(value), indent+1)
			s += fmt.Sprintf("\"%s\": %s", selectKey, list)
			selects = append(selects, s)
		}
	}
	sort.Strings(selects)
	// default condition comes last.
	selects = append(selects,
		fmt.Sprintf("%s\"%s\": [],\n", makeIndent(indent+1), "//conditions:default"))
	ret += strings.Join(selects, ",\n")

	ret += makeIndent(indent)
	ret += "})"
	return ret, nil
}

func prettyPrintLabelListAttribute(labels bazel.LabelListAttribute, indent int) (string, error) {
	// TODO(b/165114590): convert glob syntax
	ret, err := prettyPrint(reflect.ValueOf(labels.Value.Includes), indent)
	if err != nil {
		return ret, err
	}

	if !labels.HasArchSpecificValues() && !labels.HasTargetSpecificValues() {
		// Select statements not needed.
		return ret, nil
	}

	// architecture specific values.
	if labels.HasArchSpecificValues() {
		ret += " + select({\n"
		var selects []string
		for arch, selectKey := range platformArchMap {
			value := labels.GetValueForArch(arch.Name).Includes
			if len(value) > 0 {
				s := makeIndent(indent + 1)
				list, _ := prettyPrint(reflect.ValueOf(value), indent+1)
				s += fmt.Sprintf("\"%s\": %s", selectKey, list)
				selects = append(selects, s)
			}
		}
		sort.Strings(selects)
		// default condition comes last.
		selects = append(selects,
			fmt.Sprintf("%s\"%s\": [],\n", makeIndent(indent+1), "//conditions:default"))
		ret += strings.Join(selects, ",\n")

		ret += makeIndent(indent)
		ret += "})"
	}

	// target os specific values.
	if labels.HasTargetSpecificValues() {
		ret += " + select({\n"
		var selects []string
		for os, selectKey := range platformOsMap {
			value := labels.GetValueForTarget(os.Name).Includes
			if len(value) > 0 {
				s := makeIndent(indent + 1)
				list, _ := prettyPrint(reflect.ValueOf(value), indent+1)
				s += fmt.Sprintf("\"%s\": %s", selectKey, list)
				selects = append(selects, s)
			}
		}
		sort.Strings(selects)
		// default condition comes last.
		selects = append(selects,
			fmt.Sprintf("%s\"%s\": [],\n", makeIndent(indent+1), "//conditions:default"))
		ret += strings.Join(selects, ",\n")

		ret += makeIndent(indent)
		ret += "})"
	}

	return ret, nil
}
