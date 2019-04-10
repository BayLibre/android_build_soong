// Copyright 2019 Google Inc. All rights reserved.
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

package command

import (
	"reflect"
	"testing"
)

func TestIndexList(t *testing.T) {
	tests := []struct {
		description   string
		list          []string
		item          string
		expectedIndex int
	}{{
		description:   "one item, found",
		list:          []string{"test1"},
		item:          "test1",
		expectedIndex: 0,
	}, {
		description:   "no items",
		list:          []string{},
		item:          "test1",
		expectedIndex: -1,
	}, {
		description:   "one item, not found",
		list:          []string{"test1"},
		item:          "test2",
		expectedIndex: -1,
	}, {
		description:   "several items found",
		list:          []string{"test1", "test2", "test3"},
		item:          "test2",
		expectedIndex: 1,
	}, {
		description:   "several items, not found",
		list:          []string{"test1", "test2", "test3"},
		item:          "test4",
		expectedIndex: -1,
	}}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if index := indexList(tt.item, tt.list); index != tt.expectedIndex {
				t.Errorf("expected %d, got %d", tt.expectedIndex, index)
			}
		})
	}
}

func TestInList(t *testing.T) {
	tests := []struct {
		description string
		list        []string
		item        string
		inList      bool
	}{{
		description: "several items, found",
		list:        []string{"test1", "test2", "test3"},
		item:        "test2",
		inList:      true,
	}, {
		description: "several items, not found",
		list:        []string{"test1", "test2", "test3"},
		item:        "test0",
	}, {
		description: "one item, found",
		list:        []string{"test1"},
		item:        "test1",
		inList:      true,
	}, {
		description: "one item, not found",
		list:        []string{"test1"},
		item:        "test0",
	}, {
		description: "no items",
		list:        []string{},
		item:        "test0",
	}}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if found := inList(tt.item, tt.list); found != tt.inList {
				t.Errorf("expected %v, got %v", tt.inList, found)
			}
		})
	}
}

func TestRemoveInList(t *testing.T) {
	tests := []struct {
		description  string
		list         []string
		item         string
		expectedList []string
	}{{
		description:  "entire array is the same element, remove it",
		list:         []string{"test1", "test1", "test1"},
		item:         "test1",
		expectedList: []string{},
	}, {
		description:  "one element in the array to remove it",
		list:         []string{"test1", "test2", "test3"},
		item:         "test1",
		expectedList: []string{"test2", "test3"},
	}, {
		description:  "no element found to remove",
		list:         []string{"test1", "test2", "test3"},
		item:         "test0",
		expectedList: []string{"test1", "test2", "test3"},
	}, {
		description:  "remove last items of the list",
		list:         []string{"test1", "test2", "test2"},
		item:         "test2",
		expectedList: []string{"test1"},
	}, {
		description:  "empty list",
		list:         []string{},
		item:         "test0",
		expectedList: []string{},
	}}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			retVal := removeFromList(tt.item, tt.list)
			if !reflect.DeepEqual(retVal, tt.expectedList) {
				t.Errorf("expected %v, got %v", tt.expectedList, retVal)
			}
		})
	}
}
