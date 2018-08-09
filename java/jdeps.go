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
// when either make, mm, mma, mmm or mmma is called. Dependency info file is generated in out/target/product/generic_x86_64/module_java_depend.json.
func init() {
	android.RegisterSingletonType("jdeps_generator", jDepsGeneratorSingleton)
}

func jDepsGeneratorSingleton() android.Singleton {
	return &jdepsGeneratorSingleton{}
}

type jdepsGeneratorSingleton struct{
}

const (
	jdepsJsonFilename = "module_bp_java_depend.json"
	jdepsOutDirectory = "out" + string(os.PathSeparator) + "target" + string(os.PathSeparator) + "product" + string(os.PathSeparator) + "generic_x86_64"
	// Environment variables used to modify behavior of this singleton.
	envVariableCollectJavaDeps = "SOONG_COLLECT_JAVA_DEPS"
	envVariableCollectJavaDepsDebugInfo = "SOONG_COLLECT_JAVA_DEPS_DEBUG"
	envVariableTrue = "1"
)

type blueprintInfo struct {
	deps []string
	srcs []string
	exclude_srcs []string
	aidl_include_dirs []string
	aidl_local_include_dirs []string
	aidl_export_include_dirs []string
	jarjar_rules []string
	jars []string
}

// Instruct generator to trace how header include path and flags were generated.
// This is done to ease investigating bug reports.
func (j *jdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if getEnvVariable(envVariableCollectJavaDeps, ctx) != envVariableTrue {
		fmt.Printf("No need to create file: module_bp_java_depend.json\n")
		return
	}

	moduleInfos := make(map[string]blueprintInfo)

	specialGenrulesDependency(moduleInfos)

	ctx.VisitAllModules(func(module android.Module) {
	    collectModuleInfo(module, moduleInfos)
	})

	jfpath := jdepsOutDirectory + string(os.PathSeparator) + jdepsJsonFilename
	createJsonFile(moduleInfos, jfpath)
}

func getEnvVariable(name string, ctx android.SingletonContext) string {
	// Using android.Config.Getenv instead of os.getEnv to guarantee soong will
	// re-run in case this environment variable changes.
	return ctx.Config().Getenv(name)
}

func specialGenrulesDependency(moduleInfos map[string]blueprintInfo) {
	// TODO:
	// Because in frameworks/base/serviecs/core/Android.bp,
	// java_genrule {
    //    name: "services.core.priorityboosted",
    //    srcs: [":services.core.unboosted"],
    //    ...
    // }
    // "srcs" is its dependency, we get it but it's a path to services.core.unboosted.jar.
    // We hard codes here first until we find solution.
    name := "services.core.priorityboosted"
    if _, ok := moduleInfos[name]; !ok {
        bpInfo := &blueprintInfo{}
        bpInfo.deps = append(bpInfo.deps, "services.core.unboosted")
        moduleInfos[name] = *bpInfo
    } else {
        bpInfo := moduleInfos[name]
        bpInfo.deps = append(bpInfo.deps, "services.core.unboosted")
        moduleInfos[name] = bpInfo
    }
}

func collectModuleInfo(module android.Module, moduleInfos map[string]blueprintInfo) {
    collectJavaLibrayModuleInfo(module, moduleInfos)
    collectPrebuiltModuleInfo(module, moduleInfos)
    collectGenrulesModuleInfo(module, moduleInfos)
    collectFilegroupModuleInfo(module, moduleInfos)
}

func collectJavaLibrayModuleInfo(module android.Module, moduleInfos map[string]blueprintInfo) {
    name := module.Name()
    if lModule, ok := module.(*Library); ok {
        if _, ok := moduleInfos[name]; !ok {
            bpInfo := &blueprintInfo{}
            if ok := collectJavaLibraryProperties(lModule, bpInfo); ok {
                moduleInfos[name] = *bpInfo
            }
        } else {
            bpInfo := moduleInfos[name]
            if ok := collectJavaLibraryProperties(lModule, &bpInfo); ok {
                moduleInfos[name] = bpInfo
            }
        }
    }
}

func collectJavaLibraryProperties(lModule *Library, bpInfo *blueprintInfo) bool {
    var collected bool
    collected = false
    if deps, ok := lModule.CompilerDeps(); ok {
        for _, dep := range deps {
            if Contains(bpInfo.deps, dep) == true {
                bpInfo.srcs = append(bpInfo.deps, dep)
            }
        }
        collected = true
    }
    if srcs, ok := lModule.CompilerSrcs(); ok {
        for _, src := range srcs {
            if Contains(bpInfo.srcs, src) == true {
                bpInfo.srcs = append(bpInfo.srcs, src)
            }
        }
        collected = true
    }
    if srcs, ok := lModule.CompilerExcludeSrcs(); ok {
        for _, src := range srcs {
            if Contains(bpInfo.srcs, src) == true {
                bpInfo.exclude_srcs = append(bpInfo.exclude_srcs, src)
            }
        }
        collected = true
    }
    if dirs, ok := lModule.DeviceAidlIncludeDirs(); ok {
        for _, dir := range dirs {
            if Contains(bpInfo.aidl_include_dirs, dir) == true {
                bpInfo.aidl_include_dirs = append(bpInfo.aidl_include_dirs, dir)
            }
        }
        collected = true
    }
    if dirs, ok := lModule.DeviceAidlLocalIncludeDirs(); ok {
        for _, dir := range dirs {
            if Contains(bpInfo.aidl_include_dirs, dir) == true {
                bpInfo.aidl_local_include_dirs = append(bpInfo.aidl_local_include_dirs, dir)
            }
        }
        collected = true
    }
    if dirs, ok := lModule.DeviceAidlExportIncludeDirs(); ok {
        for _, dir := range dirs {
            if Contains(bpInfo.aidl_include_dirs, dir) == true {
                bpInfo.aidl_export_include_dirs = append(bpInfo.aidl_export_include_dirs, dir)
            }
        }
        collected = true
    }
    if lModule.CompilerJarjarRules() != nil {
        if Contains(bpInfo.jarjar_rules, *lModule.CompilerJarjarRules()) == true {
            bpInfo.jarjar_rules = append(bpInfo.jarjar_rules, *lModule.CompilerJarjarRules())
        }
        collected = true
    }
    return collected
}

func collectPrebuiltModuleInfo(module android.Module, moduleInfos map[string]blueprintInfo) {
    name := module.Name()
    if iModule, ok := module.(*Import); ok {
        name = iModule.Name()
        // TODO:
        // We get the Import's name, actually a base module name, but find it's
        // the same as module name with prefix "prebuilt_". We hard codes to trim
        // the prefix first until we find a solution.
        if strings.HasPrefix(name, "prebuilt_") {
            name = strings.Trim(name, "prebuilt_")
        }
        jars := iModule.PrebuiltSrcs()
        if len(jars) > 0 {
            if _, ok := moduleInfos[name]; !ok {
                bpInfo := &blueprintInfo{}
                bpInfo.jars = append(bpInfo.jars, jars...)
                moduleInfos[name] = *bpInfo
            } else {
                bpInfo := moduleInfos[name]
                for _, jar := range jars {
                    // TODO:
                    // We find VisitAllModules will be re-enter multi-times due to
                    // multi-thread. We filter out repeated jar here first until we
                    // find a solution.
                    if Contains(bpInfo.jars, jar) == true {
                        bpInfo.jars = append(bpInfo.jars, jar)
                    }
                }
                moduleInfos[name] = bpInfo
            }
        }
    }
}

func collectGenrulesModuleInfo(module android.Module, moduleInfos map[string]blueprintInfo) {
    name := module.Name()
    if gModule, ok := module.(*genrule.Module); ok {
        srcs := gModule.Srcs().Strings()
        if len(srcs) > 0 {
            if _, ok := moduleInfos[name]; !ok {
                bpInfo := &blueprintInfo{}
                bpInfo.srcs = append(bpInfo.srcs, srcs...)
                moduleInfos[name] = *bpInfo
            } else {
                bpInfo := moduleInfos[name]
                for _, src := range srcs {
                    if Contains(bpInfo.srcs, src) == true {
                        bpInfo.srcs = append(bpInfo.srcs, src)
                    }
                }
                moduleInfos[name] = bpInfo
            }
        }
    }
}

func collectFilegroupModuleInfo(module android.Module, moduleInfos map[string]blueprintInfo) {
    name := module.Name()
    switch fModule := module.(type) {
    case android.SourceFileProducer:
        if len(fModule.Srcs()) > 0 {
            if _, ok := moduleInfos[name]; !ok {
                bpInfo := &blueprintInfo{}
                for _, src := range fModule.Srcs() {
                    if src != nil {
                        bpInfo.srcs = append(bpInfo.srcs, src.String())
                    }
                }
                //bpInfo.srcs = append(bpInfo.srcs, srcs...)
                moduleInfos[name] = *bpInfo
            } else {
                bpInfo := moduleInfos[name]
                for _, src := range fModule.Srcs() {
                    if src != nil {
                        if Contains(bpInfo.srcs, src.String()) == true {
                            bpInfo.srcs = append(bpInfo.srcs, src.String())
                        }
                    }
                }
                moduleInfos[name] = bpInfo
            }
        }
    }
}

func writeJsonHead(file io.Writer) {
    fmt.Fprintf(file, "{\n")
}

func writeJsonModuleHead(file io.Writer, module_name string) {
    fmt.Fprintf(file, "  \"%s\": { ", module_name)
}

func writeJsonContains(file io.Writer, lebal_name string, list []string, ishead bool) bool {
    if len(list) > 0 {
        if ishead == true {
            fmt.Fprintf(file, "\"%s\": [", lebal_name)
        } else {
            fmt.Fprintf(file, ", \"%s\": [", lebal_name)
        }
        for i, elm := range list {
            if i < len(list)-1 {
                fmt.Fprintf(file, "\"%s\", ", elm)
            } else {
                fmt.Fprintf(file, "\"%s\"", elm)
            }
        }
        fmt.Fprintf(file, "]")
        return true
    }
    return false
}

func writeJsonModuleTail(file io.Writer, module_number int, count int) {
    if count < module_number {
        fmt.Fprintf(file, " },\n")
    } else {
        fmt.Fprintf(file, " }\n")
    }
}

func writeJsonTail(file io.Writer) {
    fmt.Fprintf(file, "}")
}

func createJsonFile(moduleInfos map[string]blueprintInfo, jfpath string) (err error) {
	if file, err := os.Create(jfpath); err == nil {
        f := bufio.NewWriter(file)
        writeJsonHead(f)
		var count int
		count = 0
		for name, info := range moduleInfos {
			writeJsonModuleHead(f, name)
			var ishead bool
			ishead = true
			if written := writeJsonContains(f, "dependencies", info.deps, ishead); written {
                ishead = false
			}
			if written := writeJsonContains(f, "srcs", info.srcs, ishead); written {
                ishead = false
			}
			if written := writeJsonContains(f, "exclude_srcs", info.exclude_srcs, ishead); written {
                ishead = false
			}
			if written := writeJsonContains(f, "aidl_include_dirs", info.aidl_include_dirs, ishead); written {
                ishead = false
			}
			if written := writeJsonContains(f, "aidl_local_include_dirs", info.aidl_local_include_dirs, ishead); written {
                ishead = false
			}
			if written := writeJsonContains(f, "aidl_export_include_dirs", info.aidl_export_include_dirs, ishead); written {
                ishead = false
			}
			if written := writeJsonContains(f, "jars", info.jars, ishead); written {
                ishead = false
			}
			if written := writeJsonContains(f, "jarjar_rules", info.jarjar_rules, ishead); written {
                ishead = false
			}
            writeJsonModuleTail(f, len(moduleInfos)-1, count)
			count++
		}
		writeJsonTail(f)
		defer f.Flush()
		//defer file.Close()
		return nil
	} else {
		fmt.Printf("failed to create file: module_bp_java_depend.json\n")
		return err
	}
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

func (j *Module) CompilerExcludeSrcs() ([]string, bool) {
     	return j.properties.Exclude_srcs, len(j.properties.Exclude_srcs) > 0
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

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
