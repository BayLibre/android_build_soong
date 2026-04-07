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
	"encoding/json"
	"fmt"
	"html"
	"os"
	"reflect"
	"regexp"
	"strings"

	"android/soong/android"

	"github.com/google/blueprint/bootstrap/bpdoc"
	"github.com/google/blueprint/proptools"
)

// Schema version. Increment when making breaking changes to the JSON format.
const moduleTypeSchemaVersion = 1

// Top-level JSON structure.
type moduleTypeSchema struct {
	Version        int                              `json:"version"`
	Arch           archSchema                       `json:"arch"`
	PropertyGroups map[string]propertyGroupEntry    `json:"property_groups"`
	ModuleTypes    map[string]moduleTypeSchemaEntry `json:"module_types"`
}

// archSchema describes the valid keys for arch/multilib/target blocks.
// Properties with tags.arch_variant == true can appear inside these blocks.
type archSchema struct {
	Arch     map[string]archTypeSchema `json:"arch"`
	Multilib []string                  `json:"multilib"`
	Target   []string                  `json:"target"`
}

// archTypeSchema describes an architecture and its sub-variants.
type archTypeSchema struct {
	ArchVariants []string `json:"arch_variants,omitempty"`
	CpuVariants  []string `json:"cpu_variants,omitempty"`
	Features     []string `json:"features,omitempty"`
}

// A shared property group (corresponds to a bpdoc.PropertyStruct used by multiple module types).
type propertyGroupEntry struct {
	Documentation string                   `json:"documentation,omitempty"`
	Properties    map[string]propertyEntry `json:"properties"`
}

// A module type entry.
type moduleTypeSchemaEntry struct {
	Documentation string                   `json:"documentation,omitempty"`
	Package       string                   `json:"package"`
	Groups        []string                 `json:"groups,omitempty"`
	Properties    map[string]propertyEntry `json:"properties"`
}

// A single property.
type propertyEntry struct {
	Type          string                   `json:"type"`
	Configurable  bool                     `json:"configurable,omitempty"`
	Documentation string                   `json:"documentation,omitempty"`
	Default       string                   `json:"default,omitempty"`
	Required      bool                     `json:"required,omitempty"`
	Tags          *propertyTags            `json:"tags,omitempty"`
	Properties    map[string]propertyEntry `json:"properties,omitempty"`
}

// LSP-relevant struct tags.
type propertyTags struct {
	Path             bool     `json:"path,omitempty"`
	ArchVariant      bool     `json:"arch_variant,omitempty"`
	ProductVariables []string `json:"product_variables,omitempty"`
}

// productVariableMap maps property names to the list of product variable names
// that support setting that property. Built once from the static variableProperties struct.
type productVariableMap map[string][]string

func writeModuleTypeSchema(ctx *android.Context, filename string) error {
	packages, err := getPackages(ctx)
	if err != nil {
		return err
	}

	// Build the reverse map: property name → product variable names.
	pvMap := buildProductVariableMap()

	schema := moduleTypeSchema{
		Version:        moduleTypeSchemaVersion,
		Arch:           buildArchSchema(),
		PropertyGroups: make(map[string]propertyGroupEntry),
		ModuleTypes:    make(map[string]moduleTypeSchemaEntry),
	}

	// First pass: count how many module types use each PropertyStruct,
	// keyed by (PkgPath, Name) to distinguish same-named structs from
	// different packages (e.g. cc.BaseCompilerProperties vs
	// rust.BaseCompilerProperties).
	type qualifiedName struct{ pkgPath, name string }
	psUsageCount := map[qualifiedName]int{}
	namePkgPaths := map[string]map[string]bool{} // name → set of PkgPaths
	for _, pkg := range packages {
		for _, mt := range pkg.ModuleTypes {
			seen := map[qualifiedName]bool{}
			for _, ps := range mt.PropertyStructs {
				qn := qualifiedName{ps.PkgPath, ps.Name}
				if ps.Name == "" || seen[qn] {
					continue
				}
				seen[qn] = true
				psUsageCount[qn]++
				if namePkgPaths[ps.Name] == nil {
					namePkgPaths[ps.Name] = map[string]bool{}
				}
				namePkgPaths[ps.Name][ps.PkgPath] = true
			}
		}
	}
	// groupKey returns the JSON property group key for a PropertyStruct.
	// If the struct name is unique across packages, returns the short name.
	// Otherwise qualifies with the last package path segment to avoid
	// collisions (e.g. "cc.BaseCompilerProperties" vs "rust.BaseCompilerProperties").
	groupKey := func(ps *bpdoc.PropertyStruct) string {
		if paths := namePkgPaths[ps.Name]; len(paths) > 1 && ps.PkgPath != "" {
			parts := strings.Split(ps.PkgPath, "/")
			return parts[len(parts)-1] + "." + ps.Name
		}
		return ps.Name
	}

	// Second pass: extract hand-written product_variables fields from named
	// PropertyStructs (e.g. selinuxContextsProperties) and merge them into
	// pvMap so those properties get tagged. This handles module-specific
	// product_variables that aren't in the global variableProperties struct.
	for _, pkg := range packages {
		for _, mt := range pkg.ModuleTypes {
			for _, ps := range mt.PropertyStructs {
				extractInlineProductVariables(ps.Properties, pvMap)
			}
		}
	}

	// Third pass: build shared property groups and module type entries.
	for _, pkg := range packages {
		for _, mt := range pkg.ModuleTypes {
			entry := moduleTypeSchemaEntry{
				Documentation: stripHTML(string(mt.Text)),
				Package:       pkg.Path,
				Properties:    make(map[string]propertyEntry),
			}

			seen := map[qualifiedName]bool{}
			for _, ps := range mt.PropertyStructs {
				// Skip archPropRoot — its arch/multilib/target fields are
				// opaque interface{} types. The valid keys are in the
				// top-level "arch" section, and arch_variant tags on
				// properties indicate which can appear inside those blocks.
				if ps.Name == "archPropRoot" {
					continue
				}
				qn := qualifiedName{ps.PkgPath, ps.Name}
				if ps.Name != "" && seen[qn] {
					continue
				}
				seen[qn] = true

				gKey := groupKey(ps)

				if ps.Name != "" && psUsageCount[qn] >= 2 {
					// Shared group: emit to property_groups if not already there.
					if _, ok := schema.PropertyGroups[gKey]; !ok {
						groupCtx := fmt.Sprintf("property_group=%s", gKey)
						props := convertProperties(ps.Properties, pvMap, groupCtx)
						// Strip product_variables — represented via tags instead.
						delete(props, "product_variables")
						schema.PropertyGroups[gKey] = propertyGroupEntry{
							Documentation: ps.Text,
							Properties:    props,
						}
					}
					entry.Groups = append(entry.Groups, gKey)
				} else {
					// Unique to this module type: inline the properties.
					mtCtx := fmt.Sprintf("module_type=%s, property_struct=%s", mt.Name, ps.Name)
					for k, v := range convertProperties(ps.Properties, pvMap, mtCtx) {
						entry.Properties[k] = v
					}
				}
			}

			// Remove product_variables from inline properties — product variable
			// support is represented via tags on individual properties instead.
			delete(entry.Properties, "product_variables")

			// Inject "name" if not present in any group or inline properties.
			if !hasNameProperty(entry, schema.PropertyGroups) {
				entry.Properties["name"] = propertyEntry{
					Type:          "string",
					Documentation: "The name of the module.",
					Required:      true,
				}
			}

			schema.ModuleTypes[mt.Name] = entry
		}
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0666)
}

// hasNameProperty checks whether "name" is defined in the module type's inline
// properties or in any of its referenced groups.
func hasNameProperty(entry moduleTypeSchemaEntry, groups map[string]propertyGroupEntry) bool {
	if _, ok := entry.Properties["name"]; ok {
		return true
	}
	for _, gName := range entry.Groups {
		if g, ok := groups[gName]; ok {
			if _, ok := g.Properties["name"]; ok {
				return true
			}
		}
	}
	return false
}

// convertProperties converts a slice of bpdoc.Property into a map keyed by property name.
func convertProperties(props []bpdoc.Property, pvMap productVariableMap, ctx string) map[string]propertyEntry {
	result := make(map[string]propertyEntry, len(props))
	for _, p := range props {
		// Skip product_variables properties — both plain and dotted forms.
		// Product variable support is represented via tags on individual properties.
		if p.Name == "product_variables" || strings.HasPrefix(p.Name, "product_variables.") {
			continue
		}
		// Skip empty-name properties. These are artifacts of bpdoc's reflection
		// fallback for dynamically-created structs with unnamed Go types
		// (e.g. []string has reflect.Type.Name() == ""). The useful type info
		// from single-empty-child wrappers is extracted in convertProperty.
		if p.Name == "" {
			continue
		}
		// Handle dotted names from collapseNestedPropertyStructs.
		if strings.Contains(p.Name, ".") {
			insertNested(result, p, pvMap, ctx)
			continue
		}
		result[p.Name] = convertProperty(p, pvMap, ctx)
	}
	return result
}

// convertProperty converts a single bpdoc.Property to a propertyEntry.
func convertProperty(p bpdoc.Property, pvMap productVariableMap, ctx string) propertyEntry {
	propCtx := fmt.Sprintf("%s, property=%s", ctx, p.Name)
	jsonType, configurable := convertBpdocType(p.Type, propCtx)

	tags := extractTags(p.Tag, p.Name, pvMap)

	entry := propertyEntry{
		Type:          jsonType,
		Configurable:  configurable,
		Documentation: stripHTML(string(p.Text)),
		Default:       p.Default,
		Tags:          tags,
	}

	if len(p.Properties) > 0 {
		// Extract type information from any empty-name child. bpdoc's
		// reflection fallback creates these for unnamed Go types (slices,
		// pointers, etc.). The empty-name child may be the only child, or
		// it may coexist with real named children when slice element structs
		// have been resolved by nesting.
		for _, child := range p.Properties {
			if child.Name == "" {
				unwrapped, conf := convertBpdocType(child.Type, propCtx)
				if unwrapped != "" {
					entry.Type = unwrapped
				}
				if conf {
					entry.Configurable = true
				}
				break
			}
		}

		if entry.Type == "" {
			entry.Type = "struct"
		}

		// convertProperties skips empty-name entries, so only real
		// named properties end up in the output.
		props := convertProperties(p.Properties, pvMap, propCtx)
		if len(props) > 0 {
			entry.Properties = props
		}
	}

	// Resolve empty type to "struct" (anonymous struct with no children).
	if entry.Type == "" {
		entry.Type = "struct"
	}

	// Normalize unrecognized named types to their schema equivalents.
	// Types with sub-properties are structs (e.g. "ApexNativeDependencies").
	// Types without sub-properties are named aliases of primitives
	// (e.g. "type Vector string") — emit as "string" since Blueprint
	// syntax uses the underlying type.
	if !isKnownType(entry.Type) {
		if strings.HasSuffix(entry.Type, "[]") {
			entry.Type = "struct[]"
		} else if entry.Properties != nil {
			entry.Type = "struct"
		} else {
			entry.Type = "string"
		}
	}

	// Scalar types must not carry sub-properties. This happens when
	// configurable wrapper structs leak their inner structure
	// (e.g. "type": "bool", "properties": {"bool": {"type": "bool"}}).
	if isScalarType(entry.Type) && entry.Properties != nil {
		entry.Properties = nil
	}

	return entry
}

// knownTypes is the set of valid schema type tokens.
var knownTypes = map[string]bool{
	"string":   true,
	"bool":     true,
	"int":      true,
	"any":      true,
	"struct":   true,
	"string[]": true,
	"bool[]":   true,
	"int[]":    true,
	"struct[]": true,
}

// isKnownType reports whether t is a valid schema type token.
func isKnownType(t string) bool {
	return knownTypes[t]
}

// isScalarType reports whether t is a type that must not carry sub-properties.
func isScalarType(t string) bool {
	switch t {
	case "string", "bool", "int", "any", "string[]", "bool[]", "int[]":
		return true
	}
	return false
}

// convertBpdocType normalizes a bpdoc type string into a clean schema type
// token. It handles Go reflection artifacts (pointer prefixes, raw slice
// syntax), the bpdoc "list of X" / "configurable X" conventions, and maps
// Go-specific type names to their schema equivalents.
//
// Unrecognized named types (e.g. "ApexNativeDependencies") are returned as-is;
// convertProperty is responsible for normalizing those based on whether the
// property has sub-properties.
func convertBpdocType(t string, ctx string) (jsonType string, configurable bool) {
	// Strip configurable prefix first — it wraps any other type.
	if strings.HasPrefix(t, "configurable ") {
		inner := t[len("configurable "):]
		innerType, _ := convertBpdocType(inner, ctx)
		return innerType, true
	}

	// Strip pointer prefix — reflection artifact (e.g. *string, *bool).
	if strings.HasPrefix(t, "*") {
		inner := t[1:]
		innerType, conf := convertBpdocType(inner, ctx)
		return innerType, conf
	}

	// Known scalar types.
	switch t {
	case "string":
		return "string", false
	case "bool":
		return "bool", false
	case "int", "int64":
		return "int", false
	case "interface", "interface {}":
		return "any", false
	}

	// bpdoc "list of X" pattern (from AST ArrayType).
	if strings.HasPrefix(t, "list of ") {
		elem := strings.TrimSpace(t[len("list of "):])
		if elem == "" {
			return "struct[]", false
		}
		innerType, _ := convertBpdocType(elem, ctx)
		if innerType == "" {
			innerType = "struct"
		}
		return innerType + "[]", false
	}

	// Raw Go slice syntax — reflection artifact (e.g. []string).
	if strings.HasPrefix(t, "[]") {
		elem := t[2:]
		if elem == "" {
			return "struct[]", false
		}
		innerType, _ := convertBpdocType(elem, ctx)
		if innerType == "" {
			innerType = "struct"
		}
		return innerType + "[]", false
	}

	// Empty type — anonymous struct from AST, resolved in convertProperty.
	if t == "" {
		return "", false
	}

	// Anything else is an unrecognized named type (e.g. "ApexNativeDependencies",
	// "java.SomeType"). Return as-is; convertProperty normalizes based on context.
	return t, false
}

// extractTags pulls LSP-relevant tag values from a reflect.StructTag and
// product variable associations.
func extractTags(tag reflect.StructTag, propName string, pvMap productVariableMap) *propertyTags {
	path := hasTagValue(tag, "android", "path")
	archVariant := hasTagValue(tag, "android", "arch_variant")
	pvars := pvMap[propName]

	if !path && !archVariant && len(pvars) == 0 {
		return nil
	}
	return &propertyTags{
		Path:             path,
		ArchVariant:      archVariant,
		ProductVariables: pvars,
	}
}

// hasTagValue checks if a struct tag contains a specific value for a given key.
func hasTagValue(tag reflect.StructTag, key, value string) bool {
	for _, entry := range strings.Split(tag.Get(key), ",") {
		if entry == value {
			return true
		}
	}
	return false
}

// insertNested handles dotted property names (from collapseNestedPropertyStructs)
// by re-nesting them into the property map hierarchy.
func insertNested(m map[string]propertyEntry, p bpdoc.Property, pvMap productVariableMap, ctx string) {
	parts := strings.SplitN(p.Name, ".", 2)
	if len(parts) == 2 {
		// Intermediate segment: ensure a parent struct exists and recurse.
		parent, ok := m[parts[0]]
		if !ok {
			parent = propertyEntry{
				Type:       "struct",
				Properties: make(map[string]propertyEntry),
			}
		}
		if parent.Properties == nil {
			parent.Properties = make(map[string]propertyEntry)
		}
		p.Name = parts[1]
		insertNested(parent.Properties, p, pvMap, ctx)
		m[parts[0]] = parent
	} else {
		// Leaf segment: convert and assign directly.
		m[parts[0]] = convertProperty(p, pvMap, ctx)
	}
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags and unescapes HTML entities, returning plain text.
func stripHTML(s string) string {
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

// extractInlineProductVariables finds hand-written product_variables fields
// in a PropertyStruct's properties and merges their variable→property mappings
// into pvMap. This handles module-specific product variables like
// selinuxContextsProperties.Product_variables.Address_sanitize.
//
// bpdoc produces a garbled structure for these hand-written fields because
// the anonymous Go structs have no package path. The structure is:
//
//	product_variables -> var_name -> var_name -> prop_name -> "" (type="[]string")
//
// We unwrap the double-nesting and extract the property names.
func extractInlineProductVariables(props []bpdoc.Property, pvMap productVariableMap) {
	for _, p := range props {
		// Handle non-collapsed form: product_variables -> var_name -> props
		if p.Name == "product_variables" {
			for _, varProp := range p.Properties {
				collectInlinePVProps(varProp.Properties, varProp.Name, pvMap)
			}
			continue
		}
		// Handle collapsed dotted name form: "product_variables.var_name" -> props
		// This comes from bpdoc's collapseNestedPropertyStructs.
		if strings.HasPrefix(p.Name, "product_variables.") {
			varName := p.Name[len("product_variables."):]
			collectInlinePVLeaves(p.Properties, varName, pvMap)
		}
	}
}

// collectInlinePVLeaves extracts leaf property names from a collapsed
// product variable's children, skipping empty-name type wrappers.
func collectInlinePVLeaves(props []bpdoc.Property, varName string, pvMap productVariableMap) {
	for _, p := range props {
		if p.Name == "" {
			continue
		}
		found := false
		for _, existing := range pvMap[p.Name] {
			if existing == varName {
				found = true
				break
			}
		}
		if !found {
			pvMap[p.Name] = append(pvMap[p.Name], varName)
		}
	}
}

// collectInlinePVProps recursively walks the garbled bpdoc property tree
// under a product variable, collecting leaf property names. It handles
// double-nesting (same name repeated) and empty-name leaf wrappers.
func collectInlinePVProps(props []bpdoc.Property, varName string, pvMap productVariableMap) {
	for _, p := range props {
		// Skip empty-name leaves — these are the raw type wrappers
		// bpdoc creates for anonymous struct fields.
		if p.Name == "" {
			continue
		}
		// If this child has the same name as the variable, it's the
		// double-nesting artifact — unwrap and recurse.
		if p.Name == varName {
			collectInlinePVProps(p.Properties, varName, pvMap)
			continue
		}
		// If this has children, it might be a nested struct property
		// (e.g., strip.all) or another wrapper — recurse to find leaves.
		if len(p.Properties) > 0 {
			// Check if any child is a real property vs just a type wrapper.
			hasRealChildren := false
			for _, child := range p.Properties {
				if child.Name != "" {
					hasRealChildren = true
					break
				}
			}
			if hasRealChildren {
				collectInlinePVProps(p.Properties, varName, pvMap)
				continue
			}
		}
		// This is a real property name — add to the map.
		found := false
		for _, existing := range pvMap[p.Name] {
			if existing == varName {
				found = true
				break
			}
		}
		if !found {
			pvMap[p.Name] = append(pvMap[p.Name], varName)
		}
	}
}

// buildArchSchema constructs the static arch/multilib/target key listings
// from the build system's registered architectures and OS types.
func buildArchSchema() archSchema {
	schema := archSchema{
		Arch:     make(map[string]archTypeSchema),
		Multilib: []string{"lib32", "lib64", "first", "both", "common", "prefer32", "native_bridge"},
	}

	// Build per-architecture entries with their variants and features.
	for _, arch := range android.ArchTypeList() {
		schema.Arch[arch.Name] = archTypeSchema{
			ArchVariants: android.ArchVariantsFor(arch),
			CpuVariants:  android.CpuVariantsFor(arch),
			Features:     android.ArchFeaturesFor(arch),
		}
	}

	// Build the target key list, mirroring createArchPropTypeDesc in arch.go.
	targets := []string{
		"host",
		"android64",
		"android32",
		"bionic",
		"glibc",
		"musl",
		"linux",
		"host_linux",
		"not_windows",
		"arm_on_x86",
		"arm_on_x86_64",
		"native_bridge",
	}
	for _, os := range android.OsTypeList() {
		targets = append(targets, os.Name)
		for _, arch := range android.ArchTypeList() {
			compound := os.Name + "_" + arch.Name
			targets = appendUnique(targets, compound)

			if os.Linux() {
				targets = appendUnique(targets, "linux_"+arch.Name)
			}
			if os.Linux() && os.Class == android.Host {
				targets = appendUnique(targets, "host_linux_"+arch.Name)
			}
			if os.Bionic() {
				targets = appendUnique(targets, "bionic_"+arch.Name)
			}
			if os.Name == "linux_glibc" {
				targets = appendUnique(targets, "glibc_"+arch.Name)
			}
			if os.Name == "linux_musl" {
				targets = appendUnique(targets, "musl_"+arch.Name)
			}
		}
	}
	schema.Target = targets

	return schema
}

func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}

// buildProductVariableMap builds a reverse map from property name to the list of
// product variable names that support that property. Extracted from the static
// variableProperties struct defined in variable.go.
func buildProductVariableMap() productVariableMap {
	result := make(productVariableMap)

	varPropsType := reflect.TypeOf(android.DefaultProductVariableProperties())
	if varPropsType.Kind() == reflect.Ptr {
		varPropsType = varPropsType.Elem()
	}
	pvField, ok := varPropsType.FieldByName("Product_variables")
	if !ok {
		return result
	}
	pvType := pvField.Type

	for i := 0; i < pvType.NumField(); i++ {
		field := pvType.Field(i)
		if !field.IsExported() {
			continue
		}

		varName := proptools.PropertyNameForField(field.Name)
		propNames := collectPropertyNames(field.Type, "")

		for _, propName := range propNames {
			result[propName] = append(result[propName], varName)
		}
	}

	return result
}

// collectPropertyNames recursively collects property names from a struct type,
// using dot-notation for nested structs.
func collectPropertyNames(t reflect.Type, prefix string) []string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	var names []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := proptools.PropertyNameForField(field.Name)
		if prefix != "" {
			name = prefix + "." + name
		}

		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		if ft.Kind() == reflect.Struct && !proptools.IsConfigurable(field.Type) {
			// Nested struct — recurse.
			names = append(names, collectPropertyNames(field.Type, name)...)
		} else {
			names = append(names, name)
		}
	}
	return names
}
