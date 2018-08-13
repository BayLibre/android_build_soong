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
	"encoding/json"
	"path/filepath"
	"fmt"
	"io"
	"os"
	"strings"

	"android/soong/android"
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
	jdepsJsonFileName = "module_bp_java_depend.json"
	specialModuleName = "services.core.priorityboosted"
	specialModuleDepend = "services.core.unboosted"
	removedPrefix = "prebuilt_"

	// Environment variables used to modify behavior of this singleton.
	envVariableCollectJavaDeps = "SOONG_COLLECT_JAVA_DEPS"
)

func (j *jdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if ctx.Config().IsEnvTrue(envVariableCollectJavaDeps) == false {
		return
	}

	moduleInfos := make(map[string]android.IdeInfo)

	ctx.VisitAllModules(func(module android.Module) {
		name := getModuleName(module)
		dpInfo := moduleInfos[name]
		collectModuleInfo(module, &dpInfo)
		if isEmpty(&dpInfo) == true {
			delete(moduleInfos, name)
		} else {
			moduleInfos[name] = dpInfo
		}
	})

	dpInfo := moduleInfos[specialModuleName]
	collectSpecialGenrulesDependency(&dpInfo)
	moduleInfos[specialModuleName] = dpInfo

	jfpath := filepath.Join(ctx.Config().Getenv("OUT"), jdepsJsonFileName)
	createJsonFile(moduleInfos, jfpath)
}

func collectModuleInfo(module android.Module, dpInfo *android.IdeInfo) {
	ideInfoProvider, ok := module.(android.IDEInfo)
	if !ok {
		return
	}
	ideInfoProvider.IDEInfo(dpInfo)
}

func addProperties(list []string, elms []string) []string {
	if len(list) == 0 {
		if len(elms) > 0 {
			list = append(list, elms...)
		}
	} else {
		for _, elm := range elms {
			if android.InList(elm, list) != true {
				list = append(list, elm)
			}
		}
	}
	return list
}

func (module *Module) IDEInfo(dpInfo *android.IdeInfo) {
	if deps, ok := module.CompilerDeps(); ok {
		dpInfo.Deps = addProperties(dpInfo.Deps, deps)
	}
	if srcs, ok := module.CompilerSrcs(); ok {
		dpInfo.Srcs = addProperties(dpInfo.Srcs, srcs)
	}
	if dirs, ok := module.DeviceAidlIncludeDirs(); ok {
		dpInfo.Aidl_include_dirs = addProperties(dpInfo.Aidl_include_dirs, dirs)
	}
	if dirs, ok := module.DeviceAidlLocalIncludeDirs(); ok {
		dpInfo.Aidl_local_include_dirs = addProperties(dpInfo.Aidl_local_include_dirs, dirs)
	}
	if dirs, ok := module.DeviceAidlExportIncludeDirs(); ok {
		dpInfo.Aidl_export_include_dirs = addProperties(dpInfo.Aidl_export_include_dirs, dirs)
	}
	if module.CompilerJarjarRules() != nil {
		dpInfo.Jarjar_rules = addProperties(dpInfo.Jarjar_rules, []string{*module.CompilerJarjarRules()})
	}
}

func (module *Import) IDEInfo(dpInfo *android.IdeInfo) {
	jars := module.PrebuiltSrcs()
	if len(jars) > 0 {
		collectPrebuiltJarsProperties(dpInfo, jars)
	}
}

func collectPrebuiltJarsProperties(dpInfo *android.IdeInfo, jars []string) {
	dpInfo.Jars = addProperties(dpInfo.Jars, jars)
}

func collectSpecialGenrulesDependency(dpInfo *android.IdeInfo) {
	// TODO:
	// Because in frameworks/base/serviecs/core/Android.bp,
	// java_genrule {
	//    name: "services.core.priorityboosted",
	//    srcs: [":services.core.unboosted"],
	//    ...
	// }
	// "srcs" is its dependency, we get it but it's a path to services.core.unboosted.jar.
	// We hard codes here first until we find a solution.
	dpInfo.Deps = addProperties(dpInfo.Deps, []string{specialModuleDepend})
}

func getModuleName(module android.Module) string {
	name := module.Name()
	if _, ok := module.(*Import); ok {
		// TODO:
		// We get the Import's name, actually a base module name, but find it's
		// the same as module name with prefix "prebuilt_". We hard codes to trim
		// the prefix first until we find a solution.
		if strings.HasPrefix(name, removedPrefix) {
			name = strings.Trim(name, removedPrefix)
		}
	}
	return name
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
	fmt.Fprintf(file, "  \"%s\": ", name)
}

func writeJsonModuleTail(file io.Writer, number int, count int) {
	if count < number {
		fmt.Fprintf(file, ",\n")
	} else if count == number {
		fmt.Fprintf(file, "\n")
	} else {
		panic(fmt.Sprintf("Unexpected item number: %d > total items: %d", number, count))
	}
}

func writeJsonTail(file io.Writer) {
	fmt.Fprintf(file, "}")
}

func createJsonFile(moduleInfos map[string]android.IdeInfo, jfpath string) (err error) {
	if file, err := os.Create(jfpath); err != nil {
		fmt.Printf("failed to create file: module_bp_java_depend.json\n")
		return err
	} else {
		defer file.Close()
		f := bufio.NewWriter(file)
		defer f.Flush()
		writeJsonHead(f)
		count := 0
		for name, info := range moduleInfos {
			writeJsonModuleHead(f, name)
			buf, err := json.Marshal(info)
			if err != nil {
				fmt.Printf("write file failed: module_bp_java_depend.json\n")
				return err
			}
			str := fmt.Sprintf("%s", buf)
			fmt.Fprintf(f, str)
			writeJsonModuleTail(f, len(moduleInfos)-1, count)
			count++
		}
		writeJsonTail(f)
	}
	return nil
}

func isEmpty(dpInfo *android.IdeInfo) bool {
	return len(dpInfo.Deps) == 0 &&
		len(dpInfo.Srcs) == 0 &&
		len(dpInfo.Aidl_include_dirs) == 0 &&
		len(dpInfo.Aidl_local_include_dirs) == 0 &&
		len(dpInfo.Aidl_export_include_dirs) == 0 &&
		len(dpInfo.Jarjar_rules) == 0 &&
		len(dpInfo.Jars) == 0
}
