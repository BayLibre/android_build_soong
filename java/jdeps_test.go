// Copyright 2014 Google Inc. All rights reserved.
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
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

func assertEqualSlices(t *testing.T, a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) == false {
		t.Fatalf("%s != %s", a, b)
	}
}

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}

func TestCollectJavaLibraryPropertiesAddLibsDeps(t *testing.T) {
	expected := []string{"Foo", "Bar"}
	module := LibraryFactory().(*Library)
	module.properties.Libs = append(module.properties.Libs, expected...)
	bpInfo := &blueprintInfo{}

	bpInfo.collectJavaLibrayModuleInfo(module)

	assertEqualSlices(t, bpInfo.deps, expected)
}

func TestCollectJavaLibraryPropertiesAddStaticLibsDeps(t *testing.T) {
	expected := []string{"Foo", "Bar"}
	module := LibraryFactory().(*Library)
	module.properties.Static_libs = append(module.properties.Static_libs, expected...)
	bpInfo := &blueprintInfo{}

	bpInfo.collectJavaLibrayModuleInfo(module)

	assertEqualSlices(t, bpInfo.deps, expected)
}

func TestCollectJavaLibraryPropertiesAddScrs(t *testing.T) {
	expected := []string{"Foo", "Bar"}
	module := LibraryFactory().(*Library)
	module.properties.Srcs = append(module.properties.Srcs, expected...)
	bpInfo := &blueprintInfo{}

	bpInfo.collectJavaLibrayModuleInfo(module)

	assertEqualSlices(t, bpInfo.srcs, expected)
}

func TestCollectJavaLibraryPropertiesAddAidlIncludeDirs(t *testing.T) {
	expected := []string{"Foo", "Bar"}
	module := LibraryFactory().(*Library)
	module.deviceProperties.Aidl.Include_dirs = append(module.deviceProperties.Aidl.Include_dirs, expected...)
	bpInfo := &blueprintInfo{}

	bpInfo.collectJavaLibrayModuleInfo(module)

	assertEqualSlices(t, bpInfo.aidl_include_dirs, expected)
}

func TestCollectJavaLibraryPropertiesAddAidlLocalIncludeDirs(t *testing.T) {
	expected := []string{"Foo", "Bar"}
	module := LibraryFactory().(*Library)
	module.deviceProperties.Aidl.Local_include_dirs = append(module.deviceProperties.Aidl.Local_include_dirs, expected...)
	bpInfo := &blueprintInfo{}

	bpInfo.collectJavaLibrayModuleInfo(module)

	assertEqualSlices(t, bpInfo.aidl_local_include_dirs, expected)
}

func TestCollectJavaLibraryPropertiesAddAidlExportIncludeDirs(t *testing.T) {
	expected := []string{"Foo", "Bar"}
	module := LibraryFactory().(*Library)
	module.deviceProperties.Aidl.Export_include_dirs = append(module.deviceProperties.Aidl.Export_include_dirs, expected...)
	bpInfo := &blueprintInfo{}

	bpInfo.collectJavaLibrayModuleInfo(module)

	assertEqualSlices(t, bpInfo.aidl_export_include_dirs, expected)
}

func TestCollectSpecialGenrulesDependency(t *testing.T) {
	expected := "services.core.unboosted"
	bpInfo := &blueprintInfo{}
	bpInfo.collectGenrulesSrcsProperties([]string{expected})
	bpInfo.collectSpecialGenrulesDependency()

	assertEqual(t, bpInfo.deps[0], expected)
}

func TestCollectJavaLibraryPropertiesAddJarjarRules(t *testing.T) {
	expected := "Jarjar_rules.txt"
	module := LibraryFactory().(*Library)
	module.properties.Jarjar_rules = &expected
	bpInfo := &blueprintInfo{}

	bpInfo.collectJavaLibrayModuleInfo(module)

	assertEqual(t, bpInfo.jarjar_rules[0], expected)
}

func TestCollectPrebuiltJarsPropertiesAddJars(t *testing.T) {
	expected := []string{"Foo.jar", "Bar.jar"}
	bpInfo := &blueprintInfo{}

	bpInfo.collectPrebuiltJarsProperties(expected)

	assertEqualSlices(t, bpInfo.jars, expected)
}

func TestCollectGenrulesSrcsPropertiesAddSrcs(t *testing.T) {
	expected := []string{"Foo", "Bar"}
	bpInfo := &blueprintInfo{}

	bpInfo.collectGenrulesSrcsProperties(expected)

	assertEqualSlices(t, bpInfo.srcs, expected)
}

func TestCollectFilegroupSrcsPropertiesAddSrcs(t *testing.T) {
	expected := "Foo"
	bpInfo := &blueprintInfo{}

	bpInfo.collectFilegroupSrcsProperties(expected)

	assertEqualSlices(t, bpInfo.srcs, []string{expected})
}

func TestWriteJsonHead(t *testing.T) {
	expected := "{\n"
	var buf bytes.Buffer

	writeJsonHead(&buf)

	assertEqual(t, buf.String(), expected)
}

func TestWriteJsonModuleHead(t *testing.T) {
	name := "Foo"
	expected := fmt.Sprintf("  \"%s\": { ", name)
	var buf bytes.Buffer

	writeJsonModuleHead(&buf, name)

	assertEqual(t, buf.String(), expected)
}

func TestWriteJsonModuleContentIsHead(t *testing.T) {
	lebal := "Srcs"
	list := []string{"Foo", "Bar"}
	expected := fmt.Sprintf("\"%s\": [\"%s\", \"%s\"]", lebal, list[0], list[1])
	var buf bytes.Buffer

	writeJsonModuleContent(&buf, lebal, list, true)

	assertEqual(t, buf.String(), expected)
}

func TestWriteJsonModuleContentNotHead(t *testing.T) {
	lebal := "Srcs"
	list := []string{"Foo", "Bar"}
	expected := fmt.Sprintf(", \"%s\": [\"%s\", \"%s\"]", lebal, list[0], list[1])
	var buf bytes.Buffer

	writeJsonModuleContent(&buf, lebal, list, false)

	assertEqual(t, buf.String(), expected)
}

func TestWriteJsonModuleTailNotLast(t *testing.T) {
	expected := " },\n"
	var buf bytes.Buffer

	writeJsonModuleTail(&buf, 2, 1)

	assertEqual(t, buf.String(), expected)
}

func TestWriteJsonModuleTailLast(t *testing.T) {
	expected := " }\n"
	var buf bytes.Buffer

	writeJsonModuleTail(&buf, 1, 1)

	assertEqual(t, buf.String(), expected)
}

func writeJsonModuleTailPanic() {
	var buf bytes.Buffer

	writeJsonModuleTail(&buf, 1, 2)
}

func TestWriteJsonModuleTailPanic(t *testing.T) {
	assertPanic(t, writeJsonModuleTailPanic)
}

func TestWriteJsonTail(t *testing.T) {
	expected := "}"
	var buf bytes.Buffer

	writeJsonTail(&buf)

	assertEqual(t, buf.String(), expected)
}
