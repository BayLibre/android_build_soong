// Copyright 2024 Google Inc. All rights reserved.
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
	"testing"

	"github.com/google/blueprint/bootstrap/bpdoc"
)

func TestConvertBpdocType(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantType     string
		wantConfig   bool
	}{
		// Basic scalar types.
		{name: "string", input: "string", wantType: "string"},
		{name: "bool", input: "bool", wantType: "bool"},
		{name: "int", input: "int", wantType: "int"},
		{name: "int64 normalizes to int", input: "int64", wantType: "int"},
		{name: "interface normalizes to any", input: "interface", wantType: "any"},
		{name: "interface {} normalizes to any", input: "interface {}", wantType: "any"},

		// Pointer types — reflection artifacts.
		{name: "pointer string", input: "*string", wantType: "string"},
		{name: "pointer bool", input: "*bool", wantType: "bool"},
		{name: "pointer int64", input: "*int64", wantType: "int"},

		// bpdoc "list of X" patterns.
		{name: "list of string", input: "list of string", wantType: "string[]"},
		{name: "list of bool", input: "list of bool", wantType: "bool[]"},
		{name: "list of int", input: "list of int", wantType: "int[]"},
		{name: "list of named struct", input: "list of Dist", wantType: "Dist[]"},
		{name: "list of empty element", input: "list of ", wantType: "struct[]"},

		// Raw Go slice syntax — reflection artifacts.
		{name: "go slice string", input: "[]string", wantType: "string[]"},
		{name: "go slice bool", input: "[]bool", wantType: "bool[]"},
		{name: "go slice int64", input: "[]int64", wantType: "int[]"},

		// Configurable variants.
		{name: "configurable string", input: "configurable string", wantType: "string", wantConfig: true},
		{name: "configurable bool", input: "configurable bool", wantType: "bool", wantConfig: true},
		{name: "configurable int", input: "configurable int", wantType: "int", wantConfig: true},
		{name: "configurable int64", input: "configurable int64", wantType: "int", wantConfig: true},
		{name: "configurable list of string", input: "configurable list of string", wantType: "string[]", wantConfig: true},

		// Empty type — anonymous struct.
		{name: "empty type", input: "", wantType: ""},

		// Unrecognized named types — passed through for convertProperty to handle.
		{name: "named struct type", input: "ApexNativeDependencies", wantType: "ApexNativeDependencies"},
		{name: "package-qualified type", input: "java.SomeType", wantType: "java.SomeType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotConfig := convertBpdocType(tt.input, "test")
			if gotType != tt.wantType {
				t.Errorf("convertBpdocType(%q) type = %q, want %q", tt.input, gotType, tt.wantType)
			}
			if gotConfig != tt.wantConfig {
				t.Errorf("convertBpdocType(%q) configurable = %v, want %v", tt.input, gotConfig, tt.wantConfig)
			}
		})
	}
}

func TestConvertProperty_ScalarWithLeakedProperties(t *testing.T) {
	// Bug 2: configurable wrapper leaks inner structure.
	// Input: type "bool" with sub-property {"bool": {"type": "bool"}}.
	p := bpdoc.Property{
		Name: "some_bool",
		Type: "configurable bool",
		Properties: []bpdoc.Property{
			{Name: "bool", Type: "bool"},
		},
	}
	entry := convertProperty(p, nil, "test")

	if entry.Type != "bool" {
		t.Errorf("Type = %q, want %q", entry.Type, "bool")
	}
	if !entry.Configurable {
		t.Error("Configurable = false, want true")
	}
	if entry.Properties != nil {
		t.Errorf("Properties should be nil for scalar type, got %v", entry.Properties)
	}
}

func TestConvertProperty_EmptyKeyUnwrap(t *testing.T) {
	// Bug 3: empty-key sub-property wrapping the real type.
	// Input: type "" with sub-property {"": {"type": "[]string"}}.
	p := bpdoc.Property{
		Name: "conditions_default",
		Type: "",
		Properties: []bpdoc.Property{
			{Name: "", Type: "[]string"},
		},
	}
	entry := convertProperty(p, nil, "test")

	if entry.Type != "string[]" {
		t.Errorf("Type = %q, want %q", entry.Type, "string[]")
	}
	if entry.Properties != nil {
		t.Errorf("Properties should be nil after unwrapping, got %v", entry.Properties)
	}
}

func TestConvertProperty_NamedStructNormalization(t *testing.T) {
	// Bug 1: named struct type should normalize to "struct".
	p := bpdoc.Property{
		Name: "deps",
		Type: "ApexNativeDependencies",
		Properties: []bpdoc.Property{
			{Name: "native_shared_libs", Type: "list of string"},
			{Name: "binaries", Type: "list of string"},
		},
	}
	entry := convertProperty(p, nil, "test")

	if entry.Type != "struct" {
		t.Errorf("Type = %q, want %q", entry.Type, "struct")
	}
	if entry.Properties == nil {
		t.Error("Properties should be preserved for struct type")
	}
	if _, ok := entry.Properties["native_shared_libs"]; !ok {
		t.Error("Properties missing 'native_shared_libs'")
	}
}

func TestConvertProperty_ListOfStruct(t *testing.T) {
	// Bug 4: "list of Dist" should become "struct[]" with sub-properties.
	p := bpdoc.Property{
		Name: "dist",
		Type: "list of Dist",
		Properties: []bpdoc.Property{
			{Name: "targets", Type: "list of string"},
			{Name: "tag", Type: "string"},
		},
	}
	entry := convertProperty(p, nil, "test")

	if entry.Type != "struct[]" {
		t.Errorf("Type = %q, want %q", entry.Type, "struct[]")
	}
	if entry.Properties == nil {
		t.Error("Properties should be preserved for struct[] type")
	}
}

func TestConvertProperty_NamedTypeNoProperties(t *testing.T) {
	// Named type with no sub-properties is a primitive alias
	// (e.g. "type Vector string"), not a struct.
	p := bpdoc.Property{
		Name: "vector",
		Type: "Vector",
	}
	entry := convertProperty(p, nil, "test")

	if entry.Type != "string" {
		t.Errorf("Type = %q, want %q", entry.Type, "string")
	}
}

func TestConvertProperty_InterfaceType(t *testing.T) {
	p := bpdoc.Property{
		Name: "extra",
		Type: "interface",
	}
	entry := convertProperty(p, nil, "test")

	if entry.Type != "any" {
		t.Errorf("Type = %q, want %q", entry.Type, "any")
	}
}

func TestConvertProperties_SkipsEmptyNames(t *testing.T) {
	props := []bpdoc.Property{
		{Name: "real_prop", Type: "string"},
		{Name: "", Type: "[]string"},
		{Name: "another_prop", Type: "bool"},
	}
	result := convertProperties(props, nil, "test")

	if _, ok := result[""]; ok {
		t.Error("empty-name property should have been skipped")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 properties, got %d", len(result))
	}
}

func TestInsertNested_NoDuplicateWrapping(t *testing.T) {
	// "backend.cpp" should produce backend → cpp, not backend → cpp → cpp.
	p := bpdoc.Property{
		Name: "backend.cpp",
		Type: "",
		Properties: []bpdoc.Property{
			{Name: "enabled", Type: "bool"},
			{Name: "cflags", Type: "list of string"},
		},
	}
	m := make(map[string]propertyEntry)
	insertNested(m, p, nil, "test")

	backend, ok := m["backend"]
	if !ok {
		t.Fatal("expected 'backend' key in result")
	}
	if backend.Type != "struct" {
		t.Errorf("backend.Type = %q, want %q", backend.Type, "struct")
	}
	cpp, ok := backend.Properties["cpp"]
	if !ok {
		t.Fatal("expected 'cpp' key under backend")
	}
	if cpp.Type != "struct" {
		t.Errorf("cpp.Type = %q, want %q", cpp.Type, "struct")
	}
	// The key check: cpp should NOT have a nested "cpp" child.
	if _, doubled := cpp.Properties["cpp"]; doubled {
		t.Error("insertNested created double-nesting: backend.cpp.cpp exists")
	}
	if _, ok := cpp.Properties["enabled"]; !ok {
		t.Error("expected 'enabled' under backend.cpp")
	}
	if _, ok := cpp.Properties["cflags"]; !ok {
		t.Error("expected 'cflags' under backend.cpp")
	}
}

func TestInsertNested_MultipleSiblings(t *testing.T) {
	// Multiple dotted siblings should share the same parent.
	m := make(map[string]propertyEntry)
	insertNested(m, bpdoc.Property{
		Name:       "target.android",
		Type:       "",
		Properties: []bpdoc.Property{{Name: "srcs", Type: "list of string"}},
	}, nil, "test")
	insertNested(m, bpdoc.Property{
		Name:       "target.host",
		Type:       "",
		Properties: []bpdoc.Property{{Name: "srcs", Type: "list of string"}},
	}, nil, "test")

	target, ok := m["target"]
	if !ok {
		t.Fatal("expected 'target' key")
	}
	if len(target.Properties) != 2 {
		t.Errorf("expected 2 children under target, got %d", len(target.Properties))
	}
	for _, child := range []string{"android", "host"} {
		entry, ok := target.Properties[child]
		if !ok {
			t.Errorf("expected %q under target", child)
			continue
		}
		if _, doubled := entry.Properties[child]; doubled {
			t.Errorf("double-nesting: target.%s.%s exists", child, child)
		}
	}
}

func TestIsKnownType(t *testing.T) {
	known := []string{"string", "bool", "int", "any", "struct", "string[]", "bool[]", "int[]", "struct[]"}
	for _, typ := range known {
		if !isKnownType(typ) {
			t.Errorf("isKnownType(%q) = false, want true", typ)
		}
	}

	unknown := []string{"int64", "interface", "string_list", "ApexNativeDependencies", "*string", "[]string", "any[]"}
	for _, typ := range unknown {
		if isKnownType(typ) {
			t.Errorf("isKnownType(%q) = true, want false", typ)
		}
	}
}

func TestIsScalarType(t *testing.T) {
	scalars := []string{"string", "bool", "int", "any", "string[]", "bool[]", "int[]"}
	for _, typ := range scalars {
		if !isScalarType(typ) {
			t.Errorf("isScalarType(%q) = false, want true", typ)
		}
	}

	nonScalars := []string{"struct", "struct[]"}
	for _, typ := range nonScalars {
		if isScalarType(typ) {
			t.Errorf("isScalarType(%q) = true, want false", typ)
		}
	}
}
