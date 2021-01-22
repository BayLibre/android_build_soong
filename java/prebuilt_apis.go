// Copyright 2018 Google Inc. All rights reserved.
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

package java

import (
	"sort"
	"strconv"
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/genrule"
)

func init() {
	RegisterPrebuiltApisBuildComponents(android.InitRegistrationContext)
}

func RegisterPrebuiltApisBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("prebuilt_apis", PrebuiltApisFactory)
}

type prebuiltApisProperties struct {
	// list of api version directories
	Api_dirs []string

	// The sdk_version of java_import modules generated based on jar files.
	// Defaults to "current"
	Imports_sdk_version *string

	// If set to true, compile dex for java_import modules. Defaults to false.
	Imports_compile_dex *bool
}

type prebuiltApis struct {
	android.ModuleBase
	properties prebuiltApisProperties
}

func (module *prebuiltApis) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// no need to implement
}

func parseJarPath(path string) (module string, apiver string, scope string) {
	elements := strings.Split(path, "/")

	apiver = elements[0]
	scope = elements[1]

	module = strings.TrimSuffix(elements[2], ".jar")
	return
}

func parseApiFilePath(ctx android.LoadHookContext, path string) (module string, apiver string, scope string) {
	elements := strings.Split(path, "/")
	apiver = elements[0]

	scope = elements[1]
	if scope != "public" && scope != "system" && scope != "test" && scope != "module-lib" && scope != "system-server" {
		ctx.ModuleErrorf("invalid scope %q found in path: %q", scope, path)
		return
	}

	// elements[2] is string literal "api". skipping.
	module = strings.TrimSuffix(elements[3], ".txt")
	return
}

func prebuiltApiModuleName(mctx android.LoadHookContext, module string, scope string, apiver string) string {
	return mctx.ModuleName() + "_" + scope + "_" + apiver + "_" + module
}

func createImport(mctx android.LoadHookContext, module, scope, apiver, path, sdkVersion string, compileDex bool) {
	props := struct {
		Name        *string
		Jars        []string
		Sdk_version *string
		Installable *bool
		Compile_dex *bool
	}{}
	props.Name = proptools.StringPtr(prebuiltApiModuleName(mctx, module, scope, apiver))
	props.Jars = append(props.Jars, path)
	props.Sdk_version = proptools.StringPtr(sdkVersion)
	props.Installable = proptools.BoolPtr(false)
	props.Compile_dex = proptools.BoolPtr(compileDex)

	mctx.CreateModule(ImportFactory, &props)
}

func createFilegroup(mctx android.LoadHookContext, name string, path string) {
	filegroupProps := struct {
		Name *string
		Srcs []string
	}{}
	filegroupProps.Name = proptools.StringPtr(name)
	filegroupProps.Srcs = []string{path}
	mctx.CreateModule(android.FileGroupFactory, &filegroupProps)
}

func createEmptyFile(mctx android.LoadHookContext, name string) {
	props := struct {
		Name *string
		Cmd  *string
		Out  []string
	}{}
	props.Name = proptools.StringPtr(name)
	props.Out = []string{name}
	props.Cmd = proptools.StringPtr("cp /dev/null $(genDir)/" + name)
	mctx.CreateModule(genrule.GenRuleFactory, &props)
}

func getPrebuiltFiles(mctx android.LoadHookContext, p *prebuiltApis, name string) []string {
	mydir := mctx.ModuleDir() + "/"
	var files []string
	for _, apiver := range p.properties.Api_dirs {
		for _, scope := range []string{"public", "system", "test", "core", "module-lib", "system-server"} {
			vfiles, err := mctx.GlobWithDeps(mydir+apiver+"/"+scope+"/"+name, nil)
			if err != nil {
				mctx.ModuleErrorf("failed to glob %s files under %q: %s", name, mydir+apiver+"/"+scope, err)
			}
			files = append(files, vfiles...)
		}
	}
	return files
}

func prebuiltSdkStubs(mctx android.LoadHookContext, p *prebuiltApis) {
	mydir := mctx.ModuleDir() + "/"
	// <apiver>/<scope>/<module>.jar
	files := getPrebuiltFiles(mctx, p, "*.jar")

	sdkVersion := proptools.StringDefault(p.properties.Imports_sdk_version, "current")
	compileDex := proptools.BoolDefault(p.properties.Imports_compile_dex, false)

	for _, f := range files {
		// create a Import module for each jar file
		localPath := strings.TrimPrefix(f, mydir)
		module, apiver, scope := parseJarPath(localPath)
		createImport(mctx, module, scope, apiver, localPath, sdkVersion, compileDex)
	}
}

func createSystemModules(mctx android.LoadHookContext, apiver string) {
	props := struct {
		Name *string
		Libs []string
	}{}
	props.Name = proptools.StringPtr(prebuiltApiModuleName(mctx, "system_modules", "public", apiver))
	props.Libs = append(props.Libs, prebuiltApiModuleName(mctx, "core-for-system-modules", "public", apiver))

	mctx.CreateModule(SystemModulesFactory, &props)
}

func prebuiltSdkSystemModules(mctx android.LoadHookContext, p *prebuiltApis) {
	for _, apiver := range p.properties.Api_dirs {
		jar := android.ExistentPathForSource(mctx,
			mctx.ModuleDir(), apiver, "public", "core-for-system-modules.jar")
		if jar.Valid() {
			createSystemModules(mctx, apiver)
		}
	}
}

func prebuiltApiFiles(mctx android.LoadHookContext, p *prebuiltApis) {
	mydir := mctx.ModuleDir() + "/"
	// <apiver>/<scope>/api/<module>.txt
	files := getPrebuiltFiles(mctx, p, "api/*.txt")

	if len(files) == 0 {
		mctx.ModuleErrorf("no api file found under %q", mydir)
	}

	// construct a map to find out the latest api file path
	// for each (<module>, <scope>) pair.
	type latestApiInfo struct {
		module  string
		scope   string
		version int
		path    string
	}

	// Create filegroups for all (<module>, <scope, <version>) triplets,
	// and a "latest" filegroup variant for each (<module>, <scope>) pair
	moduleName := func(module, scope, version string) string {
		return module + ".api." + scope + "." + version
	}
	m := make(map[string]latestApiInfo)
	max_version := -1
	for _, f := range files {
		localPath := strings.TrimPrefix(f, mydir)
		module, apiver, scope := parseApiFilePath(mctx, localPath)
		createFilegroup(mctx, moduleName(module, scope, apiver), localPath)

		version, err := strconv.Atoi(apiver)
		if err != nil {
			mctx.ModuleErrorf("Found finalized API files in non-numeric dir %v", apiver)
			return
		}

		key := module + "." + scope
		info, ok := m[key]
		if !ok {
			m[key] = latestApiInfo{module, scope, version, localPath}
		} else if version > info.version {
			info.version = version
			info.path = localPath
			m[key] = info
		}

		if version > max_version {
			max_version = version
		}
	}

	// We treat the "latest incompatibilities" specially. These files track incompatibilities
	// between the last frozen version and the work-in-progress version. So figure out if there is
	// a "work in progress" version, with the heuristic that if there is one, it must be the highest
	// version, and must only contain incompatibilities files.
	wip_version := max_version
	for _, v := range m {
		if v.version == wip_version && !strings.HasSuffix(v.module, "incompatibilities") {
			wip_version = -1
			break
		}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Sort the keys in order to make build.ninja stable
	sort.Strings(keys)
	for _, k := range keys {
		info := m[k]
		name := moduleName(info.module, info.scope, "latest")
		// If the latest incompatibilities isn't the WIP version, use empty file.
		if strings.HasSuffix(info.module, "incompatibilities") && info.version != wip_version {
			createEmptyFile(mctx, name)
		} else {
			createFilegroup(mctx, name, info.path)
		}
	}
}

func createPrebuiltApiModules(mctx android.LoadHookContext) {
	if p, ok := mctx.Module().(*prebuiltApis); ok {
		prebuiltApiFiles(mctx, p)
		prebuiltSdkStubs(mctx, p)
		prebuiltSdkSystemModules(mctx, p)
	}
}

// prebuilt_apis is a meta-module that generates filegroup modules for all
// API txt files found under the directory where the Android.bp is located.
// Specifically, an API file located at ./<ver>/<scope>/api/<module>.txt
// generates a filegroup module named <module>-api.<scope>.<ver>.
//
// It also creates <module>-api.<scope>.latest for the latest <ver>.
//
// Similarly, it generates a java_import for all API .jar files found under the
// directory where the Android.bp is located. Specifically, an API file located
// at ./<ver>/<scope>/api/<module>.jar generates a java_import module named
// <prebuilt-api-module>_<scope>_<ver>_<module>, and for SDK versions >= 30
// a java_system_modules module named
// <prebuilt-api-module>_public_<ver>_system_modules
func PrebuiltApisFactory() android.Module {
	module := &prebuiltApis{}
	module.AddProperties(&module.properties)
	android.InitAndroidModule(module)
	android.AddLoadHook(module, createPrebuiltApiModules)
	return module
}
