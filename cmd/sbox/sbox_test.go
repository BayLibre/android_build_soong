// Copyright 2022 Google Inc. Al
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

package main

import (
	"android/soong/android"
	"android/soong/cmd/sbox/sbox_proto"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/blueprint/proptools"
)

func Test_filesHaveSameContents(t *testing.T) {

	tests := []struct {
		name     string
		a        string
		b        string
		missingA bool
		missingB bool

		equal bool
	}{
		{
			name:  "empty",
			a:     "",
			b:     "",
			equal: true,
		},
		{
			name:  "equal",
			a:     "foo",
			b:     "foo",
			equal: true,
		},
		{
			name:  "unequal",
			a:     "foo",
			b:     "bar",
			equal: false,
		},
		{
			name:  "unequal different sizes",
			a:     "foo",
			b:     "foobar",
			equal: false,
		},
		{
			name:  "equal large",
			a:     strings.Repeat("a", 2*1024*1024),
			b:     strings.Repeat("a", 2*1024*1024),
			equal: true,
		},
		{
			name:  "equal large unaligned",
			a:     strings.Repeat("a", 2*1024*1024+10),
			b:     strings.Repeat("a", 2*1024*1024+10),
			equal: true,
		},
		{
			name:  "unequal large",
			a:     strings.Repeat("a", 2*1024*1024),
			b:     strings.Repeat("a", 2*1024*1024-1) + "b",
			equal: false,
		},
		{
			name:  "unequal large unaligned",
			a:     strings.Repeat("a", 2*1024*1024+10),
			b:     strings.Repeat("a", 2*1024*1024+9) + "b",
			equal: false,
		},
		{
			name:     "missing a",
			missingA: true,
			b:        "foo",
			equal:    false,
		},
		{
			name:     "missing b",
			a:        "foo",
			missingB: true,
			equal:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "testFilesHaveSameContents")
			if err != nil {
				t.Fatalf("failed to create temp dir: %s", err)
			}
			defer os.RemoveAll(tempDir)

			fileA := filepath.Join(tempDir, "a")
			fileB := filepath.Join(tempDir, "b")

			if !tt.missingA {
				err := ioutil.WriteFile(fileA, []byte(tt.a), 0666)
				if err != nil {
					t.Fatalf("failed to write %s: %s", fileA, err)
				}
			}

			if !tt.missingB {
				err := ioutil.WriteFile(fileB, []byte(tt.b), 0666)
				if err != nil {
					t.Fatalf("failed to write %s: %s", fileB, err)
				}
			}

			if got := filesHaveSameContents(fileA, fileB); got != tt.equal {
				t.Errorf("filesHaveSameContents() = %v, want %v", got, tt.equal)
			}
		})
	}
}

func TestClearOutputDirectory(t *testing.T) {
	testcases := []struct {
		name                         string
		writeType                    writeType
		onlyClearRuleOutputs         bool
		ruleOutputs                  []string
		existingFilesInDirectory     []string
		expectedFilesAfterClear      []string
		expectedDirectoryToBeDeleted bool
	}{
		{
			name:                         "writeType==alwaysWrite clears directory if !onlyClearRuleOutputs",
			writeType:                    alwaysWrite,
			onlyClearRuleOutputs:         false,
			expectedFilesAfterClear:      []string{},
			expectedDirectoryToBeDeleted: true,
		},
		{
			name:                 "writeType==onlyWriteIfChanged leaves rule outputs and clears others",
			writeType:            onlyWriteIfChanged,
			onlyClearRuleOutputs: false,
			ruleOutputs: []string{
				"rule_output1.file",
				"rule_output2.file",
				"rule_output3.file",
			},
			existingFilesInDirectory: []string{
				"rule_output1.file",
				"rule_output2.file",
				"rule_output3.file",
				"existing_1.file",
				"existing_2.file",
				"existing_3.file",
			},
			expectedFilesAfterClear: []string{
				"rule_output1.file",
				"rule_output2.file",
				"rule_output3.file",
			},
		},
		{
			name:                 "onlyClearRuleOutputs==true leaves existing files untouched",
			writeType:            alwaysWrite,
			onlyClearRuleOutputs: true,
			ruleOutputs: []string{
				"rule_output1.file",
				"rule_output2.file",
				"rule_output3.file",
			},
			existingFilesInDirectory: []string{
				"rule_output1.file",
				"rule_output2.file",
				"rule_output3.file",
				"existing_1.file",
				"existing_2.file",
				"existing_3.file",
			},
			expectedFilesAfterClear: []string{
				"existing_1.file",
				"existing_2.file",
				"existing_3.file",
			},
		},
		{
			name:                 "onlyClearRuleOutputs==true leaves all files untouched if writeType==onlyWriteIfChanged",
			writeType:            onlyWriteIfChanged,
			onlyClearRuleOutputs: true,
			ruleOutputs: []string{
				"rule_output1.file",
				"rule_output2.file",
				"rule_output3.file",
			},
			existingFilesInDirectory: []string{
				"rule_output1.file",
				"rule_output2.file",
				"rule_output3.file",
				"existing_1.file",
				"existing_2.file",
				"existing_3.file",
			},
			expectedFilesAfterClear: []string{
				"rule_output1.file",
				"rule_output2.file",
				"rule_output3.file",
				"existing_1.file",
				"existing_2.file",
				"existing_3.file",
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "testClearOutputDirectory")
			if err != nil {
				t.Fatalf("failed to create temp dir: %s", err)
			}
			defer os.RemoveAll(tempDir)

			for _, file := range tc.existingFilesInDirectory {
				err := ioutil.WriteFile(
					filepath.Join(tempDir, file),
					[]byte{},
					0666,
				)
				if err != nil {
					t.Fatalf("failed to write %s: %s", file, err)
				}
			}

			copies := []*sbox_proto.Copy{}
			for _, file := range tc.ruleOutputs {
				copies = append(copies, &sbox_proto.Copy{
					To: proptools.StringPtr(
						filepath.Join(tempDir, file),
					),
					// make a copy of the loop counter so that the pointers are unique
					From: proptools.StringPtr(file),
				})
			}

			clearOutputDirectory(copies, tempDir, tc.writeType, tc.onlyClearRuleOutputs)

			_, err = os.Stat(tempDir)
			directoryDeleted := errors.Is(err, os.ErrNotExist)
			if tc.expectedDirectoryToBeDeleted && !directoryDeleted {
				t.Errorf("expected out directory to be deleted, but it was not")
			} else if !tc.expectedDirectoryToBeDeleted && directoryDeleted {
				t.Errorf("expected out directory to not be deleted, but it was")
			}

			actualFilesList := findAllFilesUnder(tempDir)
			_, onlyInActual, onlyInExpected := android.ListSetDifference(actualFilesList, tc.expectedFilesAfterClear)
			for _, file := range onlyInActual {
				t.Errorf("found unexpected file in output directory: %q; all actual files %v", file, actualFilesList)
			}
			for _, file := range onlyInExpected {
				t.Errorf("expected file not found in output directory: %q; all actual files: %v", file, actualFilesList)
			}
		})
	}
}
