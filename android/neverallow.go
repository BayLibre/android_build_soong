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

package android

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// "neverallow" rules for the build system.
//
// This allows things which aren't related to the build system and are enforced
// for sanity, in progress code refactors, or policy to be expressed in a
// straightforward away disjoint from implementations and tests which should
// work regardless of these restrictions.
//
// A module is disallowed if all of the following are true:
// - it is in one of the "in" paths
// - it is not in one of the "notIn" paths
// - it has all "with" properties matched
// - - values are matched as regexes
// - - nested properties are separated with a '.'
// - - if the property is a list, any of the values in the list being matches
//     counts as a match
// - it has none of the "without" properties matched (same rules as above)

func registerNeverallowMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("neverallow", registerMutator).Parallel()
}

var neverallows = []rule{
	*neverallow().in("vendor/", "device/").with("Vndk.Enabled", "true").
		because("the generic system image/VNDK can never contain a library that is device dependent."),
	*neverallow().with("Vndk.Enabled", "true").without("Owner", "nil").
		because("the generic system image/VNDK can never contain a library that is device dependent."),
	*neverallow().notIn("libcore/").with("No_standard_libs", "true"),

	// Example:
	// *neverallow().with("Srcs", "main.cpp"),
	// *neverallow().with("Cflags", "-Werror"),
}

func registerMutator(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

	dir := ctx.ModuleDir() + "/"
	properties := m.GetProperties()

	for _, n := range neverallows {
		if !n.appliesToPath(dir) {
			continue
		}

		if !n.appliesToProperties(properties) {
			continue
		}

		ctx.ModuleErrorf("violates " + n.String())
	}
}

type ruleProperty struct {
	fields        []string       // e.x.: Vndk.Enabled
	value         *regexp.Regexp // e.x.: ^true|false$
	originalValue string         // e.x.: true|false
}

type rule struct {
	// User string for why this is a thing.
	reason string

	paths       []string
	unlessPaths []string

	props       []ruleProperty
	unlessProps []ruleProperty
}

func neverallow() *rule {
	return &rule{}
}
func (r *rule) in(path ...string) *rule {
	r.paths = append(r.paths, path...)
	return r
}
func (r *rule) notIn(path ...string) *rule {
	r.unlessPaths = append(r.unlessPaths, path...)
	return r
}
func (r *rule) with(field, value string) *rule {
	r.props = append(r.props, ruleProperty{
		fields:        strings.Split(field, "."),
		value:         compileValueRegex(field, value),
		originalValue: value,
	})
	return r
}
func (r *rule) without(field, value string) *rule {
	r.unlessProps = append(r.unlessProps, ruleProperty{
		fields:        strings.Split(field, "."),
		value:         compileValueRegex(field, value),
		originalValue: value,
	})
	return r
}
func (r *rule) because(reason string) *rule {
	r.reason = reason
	return r
}

func (r *rule) String() string {
	s := "neverallow"
	for _, v := range r.paths {
		s += " dir:" + v + "*"
	}
	for _, v := range r.unlessPaths {
		s += " -dir:" + v + "*"
	}
	for _, v := range r.props {
		s += " " + strings.Join(v.fields, ".") + "=" + v.originalValue
	}
	for _, v := range r.unlessProps {
		s += " -" + strings.Join(v.fields, ".") + "=" + v.originalValue
	}
	if len(r.reason) != 0 {
		s += " which is restricted because " + r.reason
	}
	return s
}

func (r *rule) appliesToPath(dir string) bool {
	includePath := len(r.paths) == 0 || hasAnyPrefix(dir, r.paths)
	excludePath := hasAnyPrefix(dir, r.unlessPaths)
	return includePath && !excludePath
}

func (r *rule) appliesToProperties(properties []interface{}) bool {
	includeProps := hasAllProperties(properties, r.props)
	excludeProps := hasAnyProperty(properties, r.unlessProps)
	return includeProps && !excludeProps
}

// assorted utils

func compileValueRegex(field string, value string) *regexp.Regexp {
	regex, err := regexp.Compile("^" + value + "$")

	if err != nil {
		panic(fmt.Sprintf("Bad regex %s for field %s.", field, value))
	}

	return regex
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func hasAnyProperty(properties []interface{}, props []ruleProperty) bool {
	for _, v := range props {
		if hasProperty(properties, v) {
			return true
		}
	}
	return false
}

func hasAllProperties(properties []interface{}, props []ruleProperty) bool {
	for _, v := range props {
		if !hasProperty(properties, v) {
			return false
		}
	}
	return true
}

func hasProperty(properties []interface{}, prop ruleProperty) bool {
	for _, propertyStruct := range properties {
		propertiesValue := reflect.ValueOf(propertyStruct).Elem()
		for _, v := range prop.fields {
			if !propertiesValue.IsValid() {
				break
			}
			propertiesValue = propertiesValue.FieldByName(v)
		}
		if !propertiesValue.IsValid() {
			continue
		}

		check := func(v string) bool {
			return prop.value.MatchString(v)
		}

		if matchValue(propertiesValue, check) {
			return true
		}
	}
	return false
}

func matchValue(value reflect.Value, check func(string) bool) bool {
	if !value.IsValid() {
		return false
	}

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return check("nil")
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.String:
		return check(value.String())
	case reflect.Bool:
		return check(strconv.FormatBool(value.Bool()))
	case reflect.Int:
		return check(strconv.FormatInt(value.Int(), 10))
	case reflect.Slice:
		slice, ok := value.Interface().([]string)
		if !ok {
			panic("Can only handle slice of string")
		}
		for _, v := range slice {
			if check(v) {
				return true
			}
		}
		return false
	}

	panic("Can't handle type: " + value.Kind().String())
}
