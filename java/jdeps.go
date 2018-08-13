// Copyright 2018 Go ogle Inc. All rights reserved.
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
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"android/soong/android"
	"android/soong/genrule"
)

// This singleton generates android java dependency into to a json file. It does so for each blueprint Android.bp resulting in a java.Module
// when either make, mm, mma, mmm or mmma is called. Dependency info file is generated in out/target/product/generic_x86_64/module_bp_java_depend.json.

func init() {
	android.RegisterSingletonType("jdeps_generator", jDepsGeneratorSingleton)
}

func jDepsGeneratorSingleton() android.Singleton {
	return &jdepsGeneratorSingleton{}
}

type jdepsGeneratorSingleton struct {
}

const (
	jdepsJsonFilename = "module_bp_java_depend.json"
	jdepsOutDirectory = "out" + string(os.PathSeparator) + "target" + string(os.PathSeparator) + "product" + string(os.PathSeparator) + "generic_x86_64"

	// Environment variables used to modify behavior of this singleton.
	envVariableCollectJavaDeps = "SOONG_COLLECT_JAVA_DEPS"
	envVariableTrue            = "1"
)

type blueprintInfo struct {
	deps                     []string
	srcs                     []string
	aidl_include_dirs        []string
	aidl_local_include_dirs  []string
	aidl_export_include_dirs []string
	jarjar_rules             []string
	jars                     []string
}

func (j *jdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if getEnvVariable(envVariableCollectJavaDeps, ctx) != envVariableTrue {
		fmt.Printf("No need to create file: module_bp_java_depend.json\n")
		return
	}

	moduleInfos := make(map[string]*blueprintInfo)

	ctx.VisitAllModules(func(module android.Module) {
		name := getModuleName(module)
		bpInfo := getBlueprintInfo(name, moduleInfos)
		if ok := collectModuleInfo(module, bpInfo); ok {
			moduleInfos[name] = bpInfo
		}
	})

	bpInfo := getBlueprintInfo("services.core.priorityboosted", moduleInfos)
	bpInfo.collectSpecialGenrulesDependency()

	jfpath := jdepsOutDirectory + string(os.PathSeparator) + jdepsJsonFilename
	createJsonFile(moduleInfos, jfpath)
}

func getEnvVariable(name string, ctx android.SingletonContext) string {
	// Using android.Config.Getenv instead of os.getEnv to guarantee soong will
	// re-run in case this environment variable changes.
	return ctx.Config().Getenv(name)
}

func getBlueprintInfo(name string, moduleInfos map[string]*blueprintInfo) *blueprintInfo {
	if _, ok := moduleInfos[name]; !ok {
		return &blueprintInfo{}
	}
	return moduleInfos[name]
}

func collectModuleInfo(module android.Module, bpInfo *blueprintInfo) bool {
	collected := false
	if ok := bpInfo.collectJavaLibrayModuleInfo(module); ok {
		collected = true
	}
	if ok := bpInfo.collectPrebuiltModuleInfo(module); ok {
		collected = true
	}
	if ok := bpInfo.collectGenrulesModuleInfo(module); ok {
		collected = true
	}
	if ok := bpInfo.collectFilegroupModuleInfo(module); ok {
		collected = true
	}
	return collected
}

func addProperties(list []string, elms []string) (bool, []string) {
	add := false
	if isEmptySlice(list) == true {
		list = append(list, elms...)
		add = true
	} else {
		// TODO:
		// We find VisitAllModules will be re-enter multi-times due to
		// multi-thread. We filter out repeated jar here first until we
		// find a solution.
		for _, elm := range elms {
			if Contains(list, elm) != true {
				list = append(list, elm)
				add = true
			}
		}
	}
	return add, list
}

func (bpInfo *blueprintInfo) isEmpty() bool {
	return isEmptySlice(bpInfo.deps) == true &&
		isEmptySlice(bpInfo.srcs) == true &&
		isEmptySlice(bpInfo.aidl_include_dirs) == true &&
		isEmptySlice(bpInfo.aidl_local_include_dirs) == true &&
		isEmptySlice(bpInfo.aidl_export_include_dirs) == true &&
		isEmptySlice(bpInfo.jarjar_rules) == true &&
		isEmptySlice(bpInfo.jars) == true
}

func (bpInfo *blueprintInfo) collectJavaLibrayModuleInfo(module android.Module) bool {
	collected := false
	if lModule, ok := module.(*Library); ok {
		if deps, ok := lModule.CompilerDeps(); ok {
			collected, bpInfo.deps = addProperties(bpInfo.deps, deps)
		}
		if srcs, ok := lModule.CompilerSrcs(); ok {
			collected, bpInfo.srcs = addProperties(bpInfo.srcs, srcs)
		}
		if dirs, ok := lModule.DeviceAidlIncludeDirs(); ok {
			collected, bpInfo.aidl_include_dirs = addProperties(bpInfo.aidl_include_dirs, dirs)
		}
		if dirs, ok := lModule.DeviceAidlLocalIncludeDirs(); ok {
			collected, bpInfo.aidl_local_include_dirs = addProperties(bpInfo.aidl_local_include_dirs, dirs)
		}
		if dirs, ok := lModule.DeviceAidlExportIncludeDirs(); ok {
			collected, bpInfo.aidl_export_include_dirs = addProperties(bpInfo.aidl_export_include_dirs, dirs)
		}
		if lModule.CompilerJarjarRules() != nil {
			collected, bpInfo.jarjar_rules = addProperties(bpInfo.jarjar_rules, []string{*lModule.CompilerJarjarRules()})
		}
	}
	return collected
}

func getModuleName(module android.Module) string {
	name := module.Name()
	if _, ok := module.(*Import); ok {
		// TODO:
		// We get the Import's name, actually a base module name, but find it's
		// the same as module name with prefix "prebuilt_". We hard codes to trim
		// the prefix first until we find a solution.
		if strings.HasPrefix(name, "prebuilt_") {
			name = strings.Trim(name, "prebuilt_")
		}
	}
	return name
}

func (bpInfo *blueprintInfo) collectPrebuiltModuleInfo(module android.Module) bool {
	collected := false
	if iModule, ok := module.(*Import); ok {
		jars := iModule.PrebuiltSrcs()
		if len(jars) > 0 {
			if ok := bpInfo.collectPrebuiltJarsProperties(jars); ok {
				collected = true
			}
		}
	}
	return collected
}

func (bpInfo *blueprintInfo) collectPrebuiltJarsProperties(jars []string) bool {
	added := false
	added, bpInfo.jars = addProperties(bpInfo.jars, jars)
	return added
}

func (bpInfo *blueprintInfo) collectGenrulesModuleInfo(module android.Module) bool {
	collected := false
	if gModule, ok := module.(*genrule.Module); ok {
		srcs := gModule.Srcs().Strings()
		if len(srcs) > 0 {
			if ok := bpInfo.collectGenrulesSrcsProperties(srcs); ok {
				collected = true
			}
		}
	}
	return collected
}

func (bpInfo *blueprintInfo) collectGenrulesSrcsProperties(srcs []string) bool {
	added := false
	added, bpInfo.srcs = addProperties(bpInfo.srcs, srcs)
	return added
}

func (bpInfo *blueprintInfo) collectFilegroupModuleInfo(module android.Module) bool {
	collected := false
	switch fModule := module.(type) {
	case android.SourceFileProducer:
		if len(fModule.Srcs()) > 0 {
			for _, src := range fModule.Srcs() {
				// TODO:
				// We have to check if the string contained in Filegroup's Srcs is nil,
				// otherwise it'll cause crash. We'll remove it if the issue is fixed.
				if src != nil {
					if ok := bpInfo.collectFilegroupSrcsProperties(src.String()); ok {
						collected = true
					}
				}
			}
		}
	}
	return collected
}

func (bpInfo *blueprintInfo) collectFilegroupSrcsProperties(src string) bool {
	add := false
	srcs := []string{src}
	add, bpInfo.srcs = addProperties(bpInfo.srcs, srcs)
	return add
}

func (bpInfo *blueprintInfo) collectSpecialGenrulesDependency() bool {
	// TODO:
	// Because in frameworks/base/serviecs/core/Android.bp,
	// java_genrule {
	//    name: "services.core.priorityboosted",
	//    srcs: [":services.core.unboosted"],
	//    ...
	// }
	// "srcs" is its dependency, we get it but it's a path to services.core.unboosted.jar.
	// We hard codes here first until we find a solution.
	collected := false
	collected, bpInfo.deps = addProperties(bpInfo.deps, []string{"services.core.unboosted"})
	return collected
}

func (j *Module) CompilerDeps() ([]string, bool) {
	deps := []string{}
	deps = append(deps, j.properties.Libs...)
	deps = append(deps, j.properties.Static_libs...)
	return deps, (len(j.properties.Libs) + len(j.properties.Static_libs)) > 0
}

func (j *Module) CompilerSrcs() ([]string, bool) {
	return j.properties.Srcs, len(j.properties.Srcs) > 0
}

func (j *Module) CompilerJarjarRules() *string {
	return j.properties.Jarjar_rules
}

func (j *Module) DeviceAidlIncludeDirs() ([]string, bool) {
	return j.deviceProperties.Aidl.Include_dirs, len(j.deviceProperties.Aidl.Include_dirs) > 0
}

func (j *Module) DeviceAidlLocalIncludeDirs() ([]string, bool) {
	return j.deviceProperties.Aidl.Local_include_dirs, len(j.deviceProperties.Aidl.Local_include_dirs) > 0
}

func (j *Module) DeviceAidlExportIncludeDirs() ([]string, bool) {
	return j.deviceProperties.Aidl.Export_include_dirs, len(j.deviceProperties.Aidl.Export_include_dirs) > 0
}

func writeJsonHead(file io.Writer) {
	fmt.Fprintf(file, "{\n")
}

func writeJsonModuleHead(file io.Writer, name string) {
	fmt.Fprintf(file, "  \"%s\": { ", name)
}

func writeJsonModuleContent(file io.Writer, lebal string, list []string, isHead bool) bool {
	if len(list) > 0 {
		if isHead == true {
			fmt.Fprintf(file, "\"%s\": [", lebal)
		} else {
			fmt.Fprintf(file, ", \"%s\": [", lebal)
		}
		for i, item := range list {
			if i < len(list)-1 {
				fmt.Fprintf(file, "\"%s\", ", item)
			} else {
				fmt.Fprintf(file, "\"%s\"", item)
			}
		}
		fmt.Fprintf(file, "]")
		return false
	}
	return isHead
}

func writeJsonModuleTail(file io.Writer, number int, count int) {
	if count < number {
		fmt.Fprintf(file, " },\n")
	} else if count == number {
		fmt.Fprintf(file, " }\n")
	} else {
		panic(fmt.Sprintf("Unexpected item number: %d > total items: %d", number, count))
	}
}

func writeJsonTail(file io.Writer) {
	fmt.Fprintf(file, "}")
}

func isEmptySlice(slice []string) bool {
	return strings.TrimSpace(strings.Join(slice, "")) == ""
}

func createJsonFile(moduleInfos map[string]*blueprintInfo, jfpath string) (err error) {
	if file, err := os.Create(jfpath); err == nil {
		f := bufio.NewWriter(file)
		writeJsonHead(f)
		count := 0
		for name, info := range moduleInfos {
			if info.isEmpty() == true {
				continue
			}
			writeJsonModuleHead(f, name)
			isHead := true
			isHead = writeJsonModuleContent(f, "dependencies", info.deps, isHead)
			isHead = writeJsonModuleContent(f, "srcs", info.srcs, isHead)
			isHead = writeJsonModuleContent(f, "aidl_include_dirs", info.aidl_include_dirs, isHead)
			isHead = writeJsonModuleContent(f, "aidl_local_include_dirs", info.aidl_local_include_dirs, isHead)
			isHead = writeJsonModuleContent(f, "aidl_export_include_dirs", info.aidl_export_include_dirs, isHead)
			isHead = writeJsonModuleContent(f, "jars", info.jars, isHead)
			isHead = writeJsonModuleContent(f, "jarjar_rules", info.jarjar_rules, isHead)
			writeJsonModuleTail(f, len(moduleInfos)-1, count)
			count++
		}
		writeJsonTail(f)
		defer f.Flush()
		return nil
	} else {
		fmt.Printf("failed to create file: module_bp_java_depend.json\n")
		return err
	}
}

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
