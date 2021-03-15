// Copyright 2021 Google Inc. All rights reserved.
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

import "android/soong/android"

// This file contains the module implementations for kythe_java_extract.

func init() {
	RegisterKytheBuildComponents(android.InitRegistrationContext)
}

func RegisterKytheBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("kythe_java_extract", kytheExtractJavaFactory)
}

type xref interface {
	XrefJavaFiles() android.Paths
}

func kytheExtractJavaFactory() android.Singleton {
	return &kytheExtractJavaSingleton{}
}

type kytheExtractJavaSingleton struct {
}

func (ks *kytheExtractJavaSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	var xrefTargets android.Paths
	ctx.VisitAllModules(func(module android.Module) {
		if javaModule, ok := module.(xref); ok {
			xrefTargets = append(xrefTargets, javaModule.XrefJavaFiles()...)
		}
	})
	// TODO(asmundak): perhaps emit a rule to output a warning if there were no xrefTargets
	if len(xrefTargets) > 0 {
		ctx.Phony("xref_java", xrefTargets...)
	}
}
