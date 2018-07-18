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
	"fmt"
	"os"
	"android/soong/android"
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
	moduleInfos map[string]blueprintInfo
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
}

// Instruct generator to trace how header include path and flags were generated.
// This is done to ease investigating bug reports.
//var outputDebugInfo = false
func (j *jdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if getEnvVariable(envVariableCollectJavaDeps, ctx) != envVariableTrue {
		fmt.Printf("No need to create file: module_bp_java_depend.json\n")
		return
	}

	j.moduleInfos = make(map[string]blueprintInfo)
	ctx.VisitAllModules(func(module android.Module) {
		if jModule, ok := module.(*Library); ok {
			name := module.Name()
			if _, ok := j.moduleInfos[name]; !ok {
				bpInfo := &blueprintInfo{}
				bpInfo.deps = append(bpInfo.deps, jModule.properties.Libs...)
				bpInfo.deps = append(bpInfo.deps, jModule.properties.Static_libs...)
				bpInfo.srcs = append(bpInfo.srcs, jModule.properties.Srcs...)
				j.moduleInfos[name] = *bpInfo
			}
		}
	})

	jfpath := jdepsOutDirectory + string(os.PathSeparator) + jdepsJsonFilename
	writeToJson(j.moduleInfos, jfpath)

	return
}

func getEnvVariable(name string, ctx android.SingletonContext) string {
	// Using android.Config.Getenv instead of os.getEnv to guarantee soong will
	// re-run in case this environment variable changes.
	return ctx.Config().Getenv(name)
}

func writeToJson(moduleInfos map[string]blueprintInfo, jfpath string) {
	if f, err := os.Create(jfpath); err == nil {
		fmt.Fprintf(f, "{\n")
		cnt := 0
		for name, info := range moduleInfos {
			fmt.Fprintf(f, "  \"%s\": { \"dependencies\": [", name)
			for i, dep := range info.deps {
				if i < len(info.deps)-1 {
					fmt.Fprintf(f, "\"%s\", ", dep)
				} else {
					fmt.Fprintf(f, "\"%s\"", dep)
				}
			}
			fmt.Fprintf(f, "], \"srcs\":[")
			for i, src := range info.srcs {
				if i < len(info.srcs)-1 {
					fmt.Fprintf(f, "\"%s\", ", src)
				} else {
					fmt.Fprintf(f, "\"%s\"", src)
				}
			}
			if cnt < len(moduleInfos)-1 {
				fmt.Fprintf(f, "]},\n")
			} else {
				fmt.Fprintf(f, "]}\n")
			}
			cnt++
		}
		fmt.Fprintf(f, "}")
		defer f.Close()
	} else {
		fmt.Printf("failed to create file: module_bp_java_depend.json\n")
	}
}
