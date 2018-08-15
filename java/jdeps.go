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
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"android/soong/android"
	"android/soong/genrule"
)

// This singleton generates android java dependency into to a json file. It does so for each
// blueprint Android.bp resulting in a java.Module when either make, mm, mma, mmm or mmma is
// called. Dependency info file is generated in $OUT/module_bp_java_depend.json.

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

	// Environment variables used to modify behavior of this singleton.
	envVariableCollectJavaDeps = "SOONG_COLLECT_JAVA_DEPS"
)

type DepsInfo struct {
	deps                     []string
	srcs                     []string
	aidl_include_dirs        []string
	aidl_local_include_dirs  []string
	aidl_export_include_dirs []string
	jarjar_rules             []string
	jars                     []string
}

func (j *jdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if ctx.Config().IsEnvTrue(envVariableCollectJavaDeps) == false {
		return
	}

	moduleInfos := make(map[string]DepsInfo)

	ctx.VisitAllModules(func(module android.Module) {
		name := getModuleName(module)
		dpInfo := moduleInfos[name]
		collectModuleInfo(module, &dpInfo)
        if dpInfo.isEmpty() == true {
            delete(moduleInfos, name)
        } else {
			moduleInfos[name] = dpInfo
		}
	})

	dpInfo := moduleInfos["services.core.priorityboosted"]
	dpInfo.collectSpecialGenrulesDependency()

    jdepsOutDirectory := ctx.Config().Getenv("OUT_DIR") + "/" + "target/product" + "/" + ctx.Config().DeviceName()
	jfpath := jdepsOutDirectory + "/" + jdepsJsonFilename
	createJsonFile(moduleInfos, jfpath)
}

func collectModuleInfo(module android.Module, dpInfo *DepsInfo) {
	dpInfo.collectJavaLibraryModuleInfo(module)
	dpInfo.collectPrebuiltModuleInfo(module)
	dpInfo.collectGenrulesModuleInfo(module)
	dpInfo.collectFilegroupModuleInfo(module)
}

func addProperties(list []string, elms []string) []string {
	if len(list) == 0 {
		list = append(list, elms...)
	} else {
		for _, elm := range elms {
			if android.InList(elm, list) != true {
				list = append(list, elm)
			}
		}
	}
	return list
}

func (dpInfo *DepsInfo) collectJavaLibraryModuleInfo(module android.Module) {
	if lModule, ok := module.(*Library); ok {
		if deps, ok := lModule.CompilerDeps(); ok {
			dpInfo.deps = addProperties(dpInfo.deps, deps)
		}
		if srcs, ok := lModule.CompilerSrcs(); ok {
			dpInfo.srcs = addProperties(dpInfo.srcs, srcs)
		}
		if dirs, ok := lModule.DeviceAidlIncludeDirs(); ok {
			dpInfo.aidl_include_dirs = addProperties(dpInfo.aidl_include_dirs, dirs)
		}
		if dirs, ok := lModule.DeviceAidlLocalIncludeDirs(); ok {
			dpInfo.aidl_local_include_dirs = addProperties(dpInfo.aidl_local_include_dirs, dirs)
		}
		if dirs, ok := lModule.DeviceAidlExportIncludeDirs(); ok {
			dpInfo.aidl_export_include_dirs = addProperties(dpInfo.aidl_export_include_dirs, dirs)
		}
		if lModule.CompilerJarjarRules() != nil {
			dpInfo.jarjar_rules = addProperties(dpInfo.jarjar_rules, []string{*lModule.CompilerJarjarRules()})
		}
	}
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

func (dpInfo *DepsInfo) collectPrebuiltModuleInfo(module android.Module) {
	if iModule, ok := module.(*Import); ok {
		jars := iModule.PrebuiltSrcs()
		if len(jars) > 0 {
			dpInfo.collectPrebuiltJarsProperties(jars)
		}
	}
}

func (dpInfo *DepsInfo) collectPrebuiltJarsProperties(jars []string) {
	dpInfo.jars = addProperties(dpInfo.jars, jars)
}

func (dpInfo *DepsInfo) collectGenrulesModuleInfo(module android.Module) {
	if gModule, ok := module.(*genrule.Module); ok {
		srcs := gModule.Srcs().Strings()
		if len(srcs) > 0 {
			dpInfo.collectGenrulesSrcsProperties(srcs)
		}
	}
}

func (dpInfo *DepsInfo) collectGenrulesSrcsProperties(srcs []string) {
	dpInfo.srcs = addProperties(dpInfo.srcs, srcs)
}

func (dpInfo *DepsInfo) collectFilegroupModuleInfo(module android.Module) {
	switch fModule := module.(type) {
	case android.SourceFileProducer:
		if len(fModule.Srcs()) > 0 {
			for _, src := range fModule.Srcs() {
				// TODO:
				// We have to check if the string contained in Filegroup's Srcs is nil,
				// otherwise it'll cause crash. We'll remove it if the issue is fixed.
				if src != nil {
					dpInfo.collectFilegroupSrcsProperties(src.String())
				}
			}
		}
	}
}

func (dpInfo *DepsInfo) collectFilegroupSrcsProperties(src string) {
	srcs := []string{src}
	dpInfo.srcs = addProperties(dpInfo.srcs, srcs)
}

func (dpInfo *DepsInfo) collectSpecialGenrulesDependency() {
	// TODO:
	// Because in frameworks/base/serviecs/core/Android.bp,
	// java_genrule {
	//    name: "services.core.priorityboosted",
	//    srcs: [":services.core.unboosted"],
	//    ...
	// }
	// "srcs" is its dependency, we get it but it's a path to services.core.unboosted.jar.
	// We hard codes here first until we find a solution.
	dpInfo.deps = addProperties(dpInfo.deps, []string{"services.core.unboosted"})
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

func createJsonFile(moduleInfos map[string]DepsInfo, jfpath string) (err error) {
	if file, err := os.Create(jfpath); err == nil {
		f := bufio.NewWriter(file)
		writeJsonHead(f)
		count := 0
		for name, info := range moduleInfos {
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

func (dpInfo *DepsInfo) isEmpty() bool {
	return len(dpInfo.deps) == 0 &&
		len(dpInfo.srcs) == 0 &&
		len(dpInfo.aidl_include_dirs) == 0 &&
		len(dpInfo.aidl_local_include_dirs) == 0 &&
		len(dpInfo.aidl_export_include_dirs) == 0 &&
		len(dpInfo.jarjar_rules) == 0 &&
		len(dpInfo.jars) == 0
}