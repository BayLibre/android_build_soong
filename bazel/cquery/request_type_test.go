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
	const expectedSplits = 3
	noResult := strings.Repeat("|", expectedSplits-1)
	testCases := []struct {
		description          string
		input                string
		expectedOutput       PyBinInfo
		expectedErrorMessage string
	}{
		{
			description:          "no splits",
			input:                noResult,
			expectedOutput:       PyBinInfo{},
			expectedErrorMessage: "Expected 1 zip file; got: ",
		},
		{
			description:    "only bin",
			input:          "test" + noResult,
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
			description: "only zip",
			input:       noResult + "test",
			expectedOutput: PyBinInfo{
				Binary:     "",
				SharedLibs: []string{""},
				SrcZip:     "test",
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
		},
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
			expectedErrorMessage: fmt.Sprintf("Expected %d items for PyBinInfo; got %d: %v", expectedSplits, 2, []string{"", ""}),
		},
		{
			description:          "too many splits",
			input:                strings.Repeat("|", expectedSplits), // 1 too many
			expectedOutput:       PyBinInfo{},
			expectedErrorMessage: fmt.Sprintf("Expected %d items for PyBinInfo; got %d: %v", expectedSplits, expectedSplits+1, make([]string, expectedSplits+1)),
		},
	}
	for _, tc := range testCases {
		actualOutput, err := GetPyBinInfo.ParseResult(tc.input)
		if (err == nil && tc.expectedErrorMessage != "") ||
			(err != nil && err.Error() != tc.expectedErrorMessage) {
			t.Errorf("%q: expected Error `%s`\n, got `%s`", tc.description, tc.expectedErrorMessage, err)
		} else if err == nil && !reflect.DeepEqual(tc.expectedOutput, actualOutput) {
			t.Errorf("%q: expected %#v != actual %#v", tc.description, tc.expectedOutput, actualOutput)
		}
	}
}

func TestGetCcInfoParseResults(t *testing.T) {
	const expectedSplits = 10
	noResult := strings.Repeat("|", expectedSplits-1)
	testCases := []struct {
		description          string
		input                string
		expectedOutput       CcInfo
		expectedErrorMessage string
	}{
		{
			description: "no result",
			input:       noResult,
			expectedOutput: CcInfo{
				OutputFiles:          []string{},
				CcObjectFiles:        []string{},
				CcSharedLibraryFiles: []string{},
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
			input:       "test" + noResult,
			expectedOutput: CcInfo{
				OutputFiles:          []string{"test"},
				CcObjectFiles:        []string{},
				CcSharedLibraryFiles: []string{},
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
			description: "only ToC",
			input:       noResult + "test",
			expectedOutput: CcInfo{
				OutputFiles:          []string{},
				CcObjectFiles:        []string{},
				CcSharedLibraryFiles: []string{},
				CcStaticLibraryFiles: []string{},
				Includes:             []string{},
				SystemIncludes:       []string{},
				Headers:              []string{},
				RootStaticArchives:   []string{},
				RootDynamicLibraries: []string{},
				TocFile:              "test",
			},
		},
		{
			description: "all items set",
			input: "out1, out2" +
				"|object1, object2" +
				"|shared_lib1, shared_lib2" +
				"|static_lib1, static_lib2" +
				"|., dir/subdir" +
				"|system/dir, system/other/dir" +
				"|dir/subdir/hdr.h" +
				"|rootstaticarchive1" +
				"|rootdynamiclibrary1" +
				"|lib.so.toc",
			expectedOutput: CcInfo{
				OutputFiles:          []string{"out1", "out2"},
				CcObjectFiles:        []string{"object1", "object2"},
				CcSharedLibraryFiles: []string{"shared_lib1", "shared_lib2"},
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
			expectedErrorMessage: fmt.Sprintf("Expected %d items, got %q", expectedSplits, []string{"", ""}),
		},
		{
			description:          "too many result splits",
			input:                strings.Repeat("|", expectedSplits+1), // 2 too many
			expectedOutput:       CcInfo{},
			expectedErrorMessage: fmt.Sprintf("Expected %d items, got %q", expectedSplits, make([]string, expectedSplits+2)),
		},
	}
	for _, tc := range testCases {
		actualOutput, err := GetCcInfo.ParseResult(tc.input)
		if (err == nil && tc.expectedErrorMessage != "") ||
			(err != nil && err.Error() != tc.expectedErrorMessage) {
			t.Errorf("%q:\nexpected Error %s\n, got %s", tc.description, tc.expectedErrorMessage, err)
		} else if err == nil && !reflect.DeepEqual(tc.expectedOutput, actualOutput) {
			t.Errorf("%q:\n expected %#v\n!= actual %#v", tc.description, tc.expectedOutput, actualOutput)
		}
	}
}
