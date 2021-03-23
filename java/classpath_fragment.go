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
	"android/soong/etc"

	"github.com/google/blueprint/proptools"
)

// Build rules and utilities to generate individual packages/modules/SdkExtensions/proto/classpaths.proto
// config files based on build configuration to embed into /system and /apex on a device.
//
// See `derive_classpath` service that reads the configs at runtime and defines *CLASSPATH variables
// on the device.

func init() {
	android.RegisterModuleType("systemserver_classpath_fragment", systemServerClasspathFragmentFactory)
}

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

// TODO(satayev): introduce a way to add test BCP entry
type classpathFragmentProperties struct {
	// Whether to generate the proto config, defaults to true.
	Generate_classpath_proto *bool
	// APEXes to keep in the classpath fragment from the monolith global config.
	Include_apexes []string
	// APEXes to filter out from the monolith global config.
	Exclude_apexes []string
}

type classpathFragment struct {
	android.ModuleBase

	classpathType classpathType

	properties classpathFragmentProperties

	outputFilepath android.OutputPath
	installDirPath android.InstallPath
}

var _ etc.PrebuiltEtcModule = (*classpathFragment)(nil)

func (*classpathFragment) BaseDir() string {
	return "etc"
}

func (*classpathFragment) SubDir() string {
	return "classpaths"
}

func (c *classpathFragment) OutputFile() android.OutputPath {
	return c.outputFilepath
}

func initClasspathFragment(c *classpathFragment, classpathType classpathType) {
	c.classpathType = classpathType
	c.AddProperties(&c.properties)
}

type classpathJar struct {
	path          string
	classpath     classpathType
	minSdkVersion int32
	maxSdkVersion int32
}

func (c *classpathFragment) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	outputFilename := strings.ToLower(c.classpathType.String())
	c.outputFilepath = android.PathForModuleOut(ctx, outputFilename).OutputPath
	c.installDirPath = android.PathForModuleInstall(ctx, c.BaseDir(), c.SubDir())

	if len(c.properties.Include_apexes) > 0 && len(c.properties.Exclude_apexes) > 0 {
		// There is no reason to use both. Either a single APEX only includes itself, or a system
		// config excludes APEXes that have been modularized.
		ctx.ModuleErrorf("cannot specify both include_apexes and exclude_apexes properties simultaneously")
		return
	}

	// TODO(satayev): propagate min/max sdk versions for the jars
	var jars []classpathJar
	includes := c.properties.Include_apexes
	excludes := c.properties.Exclude_apexes
	if proptools.BoolDefault(c.properties.Generate_classpath_proto, true) {
		switch c.classpathType {
		case BOOTCLASSPATH:
			jars = filterClasspathEntries(BOOTCLASSPATH, defaultBootclasspath(ctx), includes, excludes)
			dex2oatbootclasspath := defaultBootImageConfig(ctx).getAnyAndroidVariant().dexLocationsDeps
			jars = append(jars, filterClasspathEntries(DEX2OATBOOTCLASSPATH, dex2oatbootclasspath, includes, excludes)...)
		case SYSTEMSERVERCLASSPATH:
			jars = filterClasspathEntries(SYSTEMSERVERCLASSPATH, systemServerClasspath(ctx), includes, excludes)
		}
	}

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

func filterClasspathEntries(classpath classpathType, entries []string, includes []string, excludes []string) (result []classpathJar) {
	allowed := make(map[string]bool)

	if len(includes) > 0 {
		for _, entry := range entries {
			allow := false
			for _, include := range includes {
				if strings.Contains(entry, include) {
					allow = true
					break
				}
			}
			allowed[entry] = allow
		}
	} else {
		for _, entry := range entries {
			allow := true
			for _, exclude := range excludes {
				if strings.Contains(entry, exclude) {
					allow = false
					break
				}
			}
			allowed[entry] = allow
		}
	}

	for _, entry := range entries {
		if allowed[entry] {
			result = append(result, classpathJar{
				classpath: classpath,
				path:      entry,
			})
		}
	}

	return
}

func systemServerClasspathFragmentFactory() android.Module {
	module := &classpathFragment{}
	initClasspathFragment(module, SYSTEMSERVERCLASSPATH)
	// This module is device-only
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibFirst)
	return module
}

func (c *classpathFragment) AndroidMkEntries() []android.AndroidMkEntries {
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
