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

	ctx.FinalDepsMutators(func(ctx RegisterMutatorsContext) {
		// api_domain mutator needs to be run after all variants are created
		ctx.TopDown("api_domain", apiDomainMutator)
	})
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
        fmt.Fprintf(buf, "%-35s api_domain=%-30s %s image=%s\n", reflect.TypeOf(module), module.base().ApiDomain(), module.String(), module.ImageVariation().Variation)
	})


    // Write the file
	outFile := absolutePath(PathForOutput(ctx, "api_domains.txt").String())
    fmt.Printf("apiDomainReportSingleton.GenerateBuildActions outFile=%s\n", outFile)
	if err := pathtools.WriteFileIfChanged(outFile, buf.Bytes(), 0666); err != nil {
		ctx.Errorf(err.Error())
	}
}

// If a module type can be the root of an API Domain, it should implement this interface.
// If a particular module is an API domain root, it should return true from IsApiDomainRoot
// and the name of the Api domain from Module.RootApiDomain.  Root modules differ from
// non-root modules in that they further restrict which API surfaces can be used by a module,
// wherease normal ones pass through the same set of restrictions
type ApiDomainRootModule interface {
	IsApiDomainRoot() bool
	RootApiDomain() string
	// The API Surfaces that this API Domain root is allowed to use. This list will
	// be intersected with any parent API domains -- for example, APKs installed in an APEX
	// are restricted to using the APIs that the APEX itself can use, and can not _increase_
	// the allowable set of API Surfaces.
	// AllowedApiSurfaces() []string
}

type ApiDomainInstallableModule interface {
	IsInstallableInApiDomain() bool
	InstalledInApiDomain() string
}

// Infer API domain from the module's properties
func updateApiDomain(module Module) {
	// If it was already set, use the explicit one
	if module.base().ApiDomain() != "" {
		return
	}

	// TODO: Not sure if we should set this here. I think what we actually want is, when we find
	// a domain root, to further restrict the provider of available API Surfaces.
	rootModule, ok := module.(ApiDomainRootModule)
	if ok {
		if rootModule.IsApiDomainRoot() {
			module.SetApiDomain(rootModule.RootApiDomain())
			return
		}
	}

	installable, ok := module.(ApiDomainInstallableModule)
	if ok {
		if installable.IsInstallableInApiDomain() {
			module.SetApiDomain(installable.InstalledInApiDomain())
		}
	}
}

// Propagates the ApiDomain property and verifies that API rules are being followed, namely, that
// each module in an API domain only links against other modules in that API domain, or against
// stub modules.
func apiDomainMutator(mctx TopDownMutatorContext) {
	module := mctx.Module()

	// For the cases where we haven't explicitly set the ApiDomain in other mutators,
	// it can be inferred from other properties. Do that here.
	updateApiDomain(module)
}


