package cquery

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestGetOutputFilesParseResults(t *testing.T) {
	testCases := []struct {
		description    string
		input          string
		expectedOutput []string
	}{
		{
			description:    "no result",
			input:          "",
			expectedOutput: []string{},
		},
		{
			description:    "one result",
			input:          "test",
			expectedOutput: []string{"test"},
		},
		{
			description:    "splits on comma with space",
			input:          "foo, bar",
			expectedOutput: []string{"foo", "bar"},
		},
	}
	for _, tc := range testCases {
		actualOutput := GetOutputFiles.ParseResult(tc.input)
		if !reflect.DeepEqual(tc.expectedOutput, actualOutput) {
			t.Errorf("%q: expected %#v != actual %#v", tc.description, tc.expectedOutput, actualOutput)
		}
	}
}

func TestGetPyBinInfoParseResults(t *testing.T) {
	testCases := []struct {
		description          string
		input                string
		expectedOutput       PyBinInfo
		expectedErrorMessage string
	}{
		/*{
			description:    "no splits",
			input:          "||",
			expectedOutput: PyBinInfo{},
			expectedErrorMessage: "Expected 1 zip file; got: ",
		},
		{
			description:    "only bin",
			input:          "test||",
			expectedOutput: PyBinInfo{
				//Binary: "test",
			},
			expectedErrorMessage: "Expected 1 zip file; got: ",
		},
		{
			description:    "only libs",
			input:          "|test|",
			expectedOutput: PyBinInfo{
				//SharedLibs: []string{"test"},
			},
			expectedErrorMessage: "Expected 1 zip file; got: ",
		},
		{
			description:    "only zip",
			input:          "||test",
			expectedOutput: PyBinInfo{
				Binary: "",
				SharedLibs: []string{""},
				SrcZip: "test",
			},
			expectedErrorMessage: "",
		},
		{
			description:    "only 2 libs",
			input:          "|lib1, lib2|",
			expectedOutput: PyBinInfo{
				//SharedLibs: []string{"lib1", "lib2"},
			},
			expectedErrorMessage: "Expected 1 zip file; got: ",
		},*/
		{
			description:          "only 2 zips",
			input:                "||z1, z2",
			expectedOutput:       PyBinInfo{},
			expectedErrorMessage: "Expected 1 zip file; got: z1, z2",
		},
		{
			description: "all fields",
			input:       "bin|lib1, lib2|zip",
			expectedOutput: PyBinInfo{
				Binary:     "bin",
				SharedLibs: []string{"lib1", "lib2"},
				SrcZip:     "zip",
			},
			expectedErrorMessage: "",
		},
		{
			description:          "too few splits",
			input:                "|",
			expectedOutput:       PyBinInfo{},
			expectedErrorMessage: fmt.Sprintf("Expected %d items for PyBinInfo; got %d: %v", 3, 2, []string{"", ""}),
		},
		{
			description:          "too many splits",
			input:                strings.Repeat("|", 13),
			expectedOutput:       PyBinInfo{},
			expectedErrorMessage: fmt.Sprintf("Expected %d items for PyBinInfo; got %d: %v", 3, 14, make([]string, 14)),
		},
	}
	for _, tc := range testCases {
		actualOutput, err := GetPyBinInfo.ParseResult(tc.input)
		if (err == nil && tc.expectedErrorMessage != "") ||
			(err != nil && err.Error() != tc.expectedErrorMessage) {
			t.Errorf("%q: expected Error `%s`, got `%s`", tc.description, tc.expectedErrorMessage, err)
		} else if err == nil && !reflect.DeepEqual(tc.expectedOutput, actualOutput) {
			t.Errorf("%q: expected %#v != actual %#v", tc.description, tc.expectedOutput, actualOutput)
		}
	}
}

func TestGetCcInfoParseResults(t *testing.T) {
	testCases := []struct {
		description          string
		input                string
		expectedOutput       CcInfo
		expectedErrorMessage string
	}{
		{
			description: "no result",
			input:       "||||||||",
			expectedOutput: CcInfo{
				OutputFiles:          []string{},
				CcObjectFiles:        []string{},
				CcStaticLibraryFiles: []string{},
				Includes:             []string{},
				SystemIncludes:       []string{},
				Headers:              []string{},
				RootStaticArchives:   []string{},
				RootDynamicLibraries: []string{},
				TocFile:              "",
			},
		},
		{
			description: "only output",
			input:       "test||||||||",
			expectedOutput: CcInfo{
				OutputFiles:          []string{"test"},
				CcObjectFiles:        []string{},
				CcStaticLibraryFiles: []string{},
				Includes:             []string{},
				SystemIncludes:       []string{},
				Headers:              []string{},
				RootStaticArchives:   []string{},
				RootDynamicLibraries: []string{},
				TocFile:              "",
			},
		},
		{
			description: "all items set",
			input:       "out1, out2|static_lib1, static_lib2|object1, object2|., dir/subdir|system/dir, system/other/dir|dir/subdir/hdr.h|rootstaticarchive1|rootdynamiclibrary1|lib.so.toc",
			expectedOutput: CcInfo{
				OutputFiles:          []string{"out1", "out2"},
				CcObjectFiles:        []string{"object1", "object2"},
				CcStaticLibraryFiles: []string{"static_lib1", "static_lib2"},
				Includes:             []string{".", "dir/subdir"},
				SystemIncludes:       []string{"system/dir", "system/other/dir"},
				Headers:              []string{"dir/subdir/hdr.h"},
				RootStaticArchives:   []string{"rootstaticarchive1"},
				RootDynamicLibraries: []string{"rootdynamiclibrary1"},
				TocFile:              "lib.so.toc",
			},
		},
		{
			description:          "too few result splits",
			input:                "|",
			expectedOutput:       CcInfo{},
			expectedErrorMessage: fmt.Sprintf("Expected %d items, got %q", 9, []string{"", ""}),
		},
		{
			description:          "too many result splits",
			input:                strings.Repeat("|", 50),
			expectedOutput:       CcInfo{},
			expectedErrorMessage: fmt.Sprintf("Expected %d items, got %q", 9, make([]string, 51)),
		},
	}
	for _, tc := range testCases {
		actualOutput, err := GetCcInfo.ParseResult(tc.input)
		if (err == nil && tc.expectedErrorMessage != "") ||
			(err != nil && err.Error() != tc.expectedErrorMessage) {
			t.Errorf("%q: expected Error %s, got %s", tc.description, tc.expectedErrorMessage, err)
		} else if err == nil && !reflect.DeepEqual(tc.expectedOutput, actualOutput) {
			t.Errorf("%q: expected %#v != actual %#v", tc.description, tc.expectedOutput, actualOutput)
		}
	}
}
