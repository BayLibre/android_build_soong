// Copyright 2022 Google Inc. All rights reserved.
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

// Handles all of the logic about which modules are in which API domains

import (
    "bytes"
	"fmt"
    "reflect"

    "github.com/google/blueprint/pathtools"

    //
    //"path/filepath"
	//"strings"

	//"github.com/google/blueprint"
	//"github.com/google/blueprint/proptools"
)

func init() {
	registerApiDomainBuildComponents(InitRegistrationContext)
}

func registerApiDomainBuildComponents(ctx RegistrationContext) {
	ctx.RegisterSingletonType("api_domain_report", func() Singleton { return &apiDomainReportSingleton{} })
}

type apiDomainReportSingleton struct {
}

func (ks *apiDomainReportSingleton) GenerateBuildActions(ctx SingletonContext) {

    /*
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
    */


    // List all modules and their API Domains
	buf := &bytes.Buffer{}
	ctx.VisitAllModules(func(module Module) {
        fmt.Fprintf(buf, "%s %s image=%s api_domain=%s\n", module.String(), reflect.TypeOf(module), module.ImageVariation().Variation, module.ApiDomain())
	})


    // Write the file
	outFile := absolutePath(PathForOutput(ctx, "api_domains.txt").String())
    fmt.Printf("apiDomainReportSingleton.GenerateBuildActions outFile=%s\n", outFile)
	if err := pathtools.WriteFileIfChanged(outFile, buf.Bytes(), 0666); err != nil {
		ctx.Errorf(err.Error())
	}

}


