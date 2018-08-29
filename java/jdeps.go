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
	"fmt"
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
	// Environment variables used to modify behavior of this singleton.
	envVariableCollectJavaDeps = "SOONG_COLLECT_JAVA_DEPS"
	jdepsJsonFileName          = "module_bp_java_deps.json"
	removedPrefix              = "prebuilt_"
	android_variant            = "android_common"
)

func (j *jdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if ctx.Config().IsEnvTrue(envVariableCollectJavaDeps) == false {
		return
	}

	moduleInfos := make(map[string]android.IdeInfo)

	ctx.VisitAllModules(func(module android.Module) {
		if ctx.ModuleSubDir(module) != android_variant {
			return
		}
		ideInfoProvider, ok := module.(android.IDEInfo)
		if !ok {
			return
		}
		name := ideInfoProvider.BaseModuleName()
		dpInfo := moduleInfos[name]
		ideInfoProvider.IDEInfo(&dpInfo)
		moduleInfos[name] = dpInfo

		// Make does not understand LinuxBionic, LinuxBionic case isn't written
		// in out/soong/Android-aosp_x86_64.mk. We only need the installed path
		// written in Android-aosp_x86_64.mk, so we filter out LinuxBionic case.
		if module.Target().Os == android.LinuxBionic {
			return
		}

		mkProvider, ok := module.(android.AndroidMkDataProvider)
		if !ok {
			return
		}
		data := mkProvider.AndroidMk()
		if data.Class != "" {
			dpInfo.Classes = addProperties(dpInfo.Classes, data.Class)
		}
		out := data.OutputFile.String()
		if out != "" {
			dpInfo.Installed_paths = addProperties(dpInfo.Installed_paths, out)
		}
		moduleInfos[name] = dpInfo
	})

	jfpath := android.PathForOutput(ctx, jdepsJsonFileName).String()
	createJsonFile(moduleInfos, jfpath)
}

func addProperties(slice []string, elems ...string) []string {
	if len(slice) == 0 {
		if len(elems) > 0 {
			slice = append(slice, elems...)
		}
	} else {
		for _, elem := range elems {
			if android.InList(elem, slice) != true {
				slice = append(slice, elem)
			}
		}
	}
	return slice
}

func (module *Module) IDEInfo(dpInfo *android.IdeInfo) {
	if deps, ok := module.CompilerDeps(); ok {
		dpInfo.Deps = addProperties(dpInfo.Deps, deps...)
	}
	if srcs, ok := module.CompilerSrcs(); ok {
		dpInfo.Srcs = addProperties(dpInfo.Srcs, srcs...)
	}
	if dirs, ok := module.DeviceAidlIncludeDirs(); ok {
		dpInfo.Aidl_include_dirs = addProperties(dpInfo.Aidl_include_dirs, dirs...)
	}
	if module.CompilerJarjarRules() != nil {
		dpInfo.Jarjar_rules = addProperties(dpInfo.Jarjar_rules, *module.CompilerJarjarRules())
	}
}

func (module *Import) IDEInfo(dpInfo *android.IdeInfo) {
	jars := module.PrebuiltSrcs()
	if len(jars) > 0 {
		collectPrebuiltJarsProperties(dpInfo, jars)
	}
}

func collectPrebuiltJarsProperties(dpInfo *android.IdeInfo, jars []string) {
	dpInfo.Jars = addProperties(dpInfo.Jars, jars...)
}

func (module *Import) BaseModuleName() string {
	// TODO:
	// Extract the base module name from the Import name.
	// Often the Import name has a prefix "prebuilt_".
	// Remove the prefix explicitly if needed
	// until we find a better solution to get the Import name.
	name := module.Name()
	if strings.HasPrefix(name, removedPrefix) {
		name = strings.Trim(name, removedPrefix)
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

func createJsonFile(moduleInfos map[string]android.IdeInfo, jfpath string) (err error) {
	file, err := os.Create(jfpath)
	if err != nil {
		return fmt.Errorf("Failed to create file: %s, relative: %v", jdepsJsonFileName, err)
	}
	defer file.Close()
	f := bufio.NewWriter(file)
	defer f.Flush()
	buf, err := json.MarshalIndent(moduleInfos, "", "\t")
	if err != nil {
		return fmt.Errorf("Write file failed: %s, relative: %v", jdepsJsonFileName, err)
	}
	str := fmt.Sprintf("%s", buf)
	fmt.Fprintf(f, str)
	return nil
}
