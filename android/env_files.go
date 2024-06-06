// Copyright 2024 The Android Open Source Project
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
	"strings"
)

func init() {
	RegisterModuleType("env_files", envFilesFactory)
}

// se_build_files gathers policy files from sepolicy dirs, and acts like a filegroup. A tag with
// partition(plat, system_ext, product) and scope(public, private) is used to select directories.
// Supported tags are: "plat_public", "plat_private", "system_ext_public", "system_ext_private",
// "product_public", "product_private", and "reqd_mask".
func envFilesFactory() Module {
	module := &envFiles{}
	module.AddProperties(&module.properties)
	InitAndroidModule(module)
	return module
}

type envFilesProperties struct {
	Env_name string
}

type envFiles struct {
	ModuleBase
	properties envFilesProperties

	srcs       Paths
}

/**
func (e *envFiles) setSrcs(ctx ModuleContext, dirs ...string) Paths {
	result := Paths{}
	
	return result
}
**/

func (e *envFiles) DepsMutator(ctx BottomUpMutatorContext) {
	// do nothing
}

func (e *envFiles) OutputFiles(tag string) (Paths, error) {
	if tag == "" {
		return e.srcs, nil
	}
	return nil, fmt.Errorf("unsupported module reference tag %q", tag)
}

var _ OutputFileProducer = (*envFiles)(nil)

func (e *envFiles) GenerateAndroidBuildActions(ctx ModuleContext) {
    result := Paths{}
    //envStr := ctx.Config().Getenv(e.properties.Env_name)
    envStr := "device/google/gs-common/camera/device_framework_matrix_product.xml"
    
    for _, f := range strings.Fields(envStr) {
        fmt.Printf("Bill %s\n", f)
        result = append(result, PathForSource(ctx, f))
        fmt.Printf("Bill %s\n", result)
    }
    
    e.srcs = result
}
