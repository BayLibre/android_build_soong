/*
 * Copyright (C) 2021 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package java

import (
	"fmt"
	"strings"

	"android/soong/android"

	"github.com/google/blueprint/proptools"
)

// Build rules and utilities to generate individual packages/modules/SdkExtensions/proto/classpaths.proto
// config files based on build configuration to embed into /system and /apex on a device.
//
// See `derive_classpath` service that reads the configs at runtime and defines *CLASSPATH variables
// on the device.

type classpathType int

const (
	// Matches definition in packages/modules/SdkExtensions/proto/classpaths.proto
	BOOTCLASSPATH classpathType = iota
	DEX2OATBOOTCLASSPATH
	SYSTEMSERVERCLASSPATH
)

func (c classpathType) String() string {
	return [...]string{"BOOTCLASSPATH", "DEX2OATBOOTCLASSPATH", "SYSTEMSERVERCLASSPATH"}[c]
}

type classpathFragmentProperties struct {
	// APEX name that includes this classpath fragment. This is used to filter appropriate
	// classpath jars from the monolith global config. Must only be set if the classpath fragment
	// is used in an APEX.
	Include_apex *string
	// APEXes to filter out from the monolith global config. Must only be used by non-apex classpath
	// fragments.
	Exclude_apexes []string
}

// classpathFragment interface is implemented by a module that contributes jars to a *CLASSPATH
// variables at runtime.
type classpathFragment interface {
	android.Module

	classpathFragmentBase() *ClasspathFragmentBase
}

// ClasspathFragmentBase is meant to be embedded in any module types that implement classpathFragment;
// such modules are expected to call initClasspathFragment().
type ClasspathFragmentBase struct {
	properties classpathFragmentProperties

	outputFilepath android.OutputPath
	installDirPath android.InstallPath
}

func (c *ClasspathFragmentBase) classpathFragmentBase() *ClasspathFragmentBase {
	return c
}

// Initializes ClasspathFragmentBase struct. Must be called by all modules that include ClasspathFragmentBase.
func initClasspathFragment(c classpathFragment) {
	base := c.classpathFragmentBase()
	c.AddProperties(&base.properties)
}

// Matches definition of Jar in packages/modules/SdkExtensions/proto/classpaths.proto
type classpathJar struct {
	path      string
	classpath classpathType
	// TODO(satayev): propagate min/max sdk versions for the jars
	minSdkVersion int32
	maxSdkVersion int32
}

func (c *ClasspathFragmentBase) generateAndroidBuildActions(ctx android.ModuleContext) {
	outputFilename := ctx.ModuleName() + ".pb"
	c.outputFilepath = android.PathForModuleOut(ctx, outputFilename).OutputPath
	c.installDirPath = android.PathForModuleInstall(ctx, "etc", "classpaths")

	includeApex := proptools.String(c.properties.Include_apex)
	excludeApexes := c.properties.Exclude_apexes

	if len(includeApex) > 0 && len(excludeApexes) > 0 {
		// There is no reason to use both. Either a single APEX only includes itself, or a system
		// config excludes APEXes that have been modularized.
		ctx.ModuleErrorf("cannot specify both include_apex and exclude_apexes properties simultaneously")
		return
	}

	predicates := []func(string) bool{
		// Keep any jar that's part of the includeApex
		func(path string) bool {
			return hasAnySubstring(path, includeApex)
		},
		// Filter out any jars that are part of APEXes we are excluding
		func(path string) bool {
			return !hasAnySubstring(path, excludeApexes...)
		},
	}

	var jars []classpathJar

	bootclasspath := filter(defaultBootclasspath(ctx), predicates...)
	jars = appendClasspathJar(jars, BOOTCLASSPATH, bootclasspath...)
	dex2oatbootclasspath := filter(defaultBootImageConfig(ctx).getAnyAndroidVariant().dexLocationsDeps, predicates...)
	jars = appendClasspathJar(jars, DEX2OATBOOTCLASSPATH, dex2oatbootclasspath...)
	systemserverclasspath := filter(systemServerClasspath(ctx), predicates...)
	jars = appendClasspathJar(jars, SYSTEMSERVERCLASSPATH, systemserverclasspath...)

	generatedJson := android.PathForModuleOut(ctx, outputFilename+".json")
	writeClasspathsJson(ctx, generatedJson, jars)

	rule := android.NewRuleBuilder(pctx, ctx)
	rule.Command().
		BuiltTool("conv_classpaths_proto").
		Flag("encode").
		Flag("--format=json").
		FlagWithInput("--input=", generatedJson).
		FlagWithOutput("--output=", c.outputFilepath)

	rule.Build("classpath_fragment", "Compiling "+c.outputFilepath.String())
}

func writeClasspathsJson(ctx android.ModuleContext, output android.WritablePath, jars []classpathJar) {
	var content strings.Builder
	fmt.Fprintf(&content, "{\n")
	fmt.Fprintf(&content, "\"jars\": [\n")
	for idx, jar := range jars {
		fmt.Fprintf(&content, "{\n")

		fmt.Fprintf(&content, "\"relativePath\": \"%s\",\n", jar.path)
		fmt.Fprintf(&content, "\"classpath\": \"%s\"\n", jar.classpath)

		if idx < len(jars)-1 {
			fmt.Fprintf(&content, "},\n")
		} else {
			fmt.Fprintf(&content, "}\n")
		}
	}
	fmt.Fprintf(&content, "]\n")
	fmt.Fprintf(&content, "}\n")
	android.WriteFileRule(ctx, output, content.String())
}

func appendClasspathJar(slice []classpathJar, classpathType classpathType, paths ...string) (result []classpathJar) {
	result = append(result, slice...)
	for _, path := range paths {
		result = append(result, classpathJar{
			path:      path,
			classpath: classpathType,
		})
	}
	return
}

// Filters a given list, where each element is tested against all given predicate functions, and
// returns the filtered elements in the same relative order as the input.
func filter(list []string, predicates ...func(string) bool) (result []string) {
	for _, s := range list {
		pass := true
		for _, predicate := range predicates {
			if !predicate(s) {
				pass = false
				break
			}
		}
		if pass {
			result = append(result, s)
		}
	}
	return
}

// Checks if a given string contains at least one of the substrings.
func hasAnySubstring(str string, substrings ...string) bool {
	for _, s := range substrings {
		if strings.Contains(str, s) {
			return true
		}
	}
	return false
}

func (c *ClasspathFragmentBase) getAndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(c.outputFilepath),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", c.installDirPath.ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", c.outputFilepath.Base())
			},
		},
	}}
}
