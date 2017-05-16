// Copyright 2017 Google Inc. All rights reserved.
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

// This file implements the logic of bpfix and also provides a programmatic interface

package bpfix

import (
	"testing"

	"github.com/google/blueprint/parser"
	"strings"
)

func TestRemoveKnownUnnecessaryVariables_DuplicateValues(t *testing.T) {
	input := `cc_library_shared {
	    name: "iAmAModule",
	    local_include_dirs: ["include"],
	    export_include_dirs: ["include"],
	}
	`
	tree, errs := parser.Parse("", strings.NewReader(input), parser.NewScope(nil))

	mod := tree.Defs[0].(*parser.Module)
	_, found := mod.GetProperty("local_include_dirs")
	if !found {
		t.Fatalf("failed to include key local_include_dirs in parse tree")
	}
	if len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.Fatalf("%d parse errors", len(errs))
	}
	tree, errs = RemoveKnownUnnecessaryVariables(tree)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.Fatalf("%d errors", len(errs))
	}
	mod = tree.Defs[0].(*parser.Module)
	_, found = mod.GetProperty("local_include_dirs")
	if found {
		t.Fatalf("failed to remove key 'local_include_dirs' despite existence of key 'export_include_dirs' with the same value")
	}
}

func TestRemoveKnownUnnecessaryVariables_DifferentValues(t *testing.T) {
	input := `cc_library_shared {
	    name: "iAmAModule2",
	    local_include_dirs: ["value1"],
	    export_include_dirs: ["value2"],
	}
	`
	tree, errs := parser.Parse("", strings.NewReader(input), parser.NewScope(nil))

	mod := tree.Defs[0].(*parser.Module)
	_, found := mod.GetProperty("local_include_dirs")
	if !found {
		t.Fatalf("failed to include key local_include_dirs in parse tree")
	}
	if len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.Fatalf("%d parse errors", len(errs))
	}
	tree, errs = RemoveKnownUnnecessaryVariables(tree)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.Fatalf("%d errors", len(errs))
	}
	mod = tree.Defs[0].(*parser.Module)
	_, found = mod.GetProperty("local_include_dirs")
	if !found {
		t.Fatalf("removed key 'local_include_dirs' while key 'export_include_dirs' existed with a different value")
	}
}
