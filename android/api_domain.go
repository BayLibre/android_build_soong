// Copyright 2015 Google Inc. All rights reserved.
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

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type ApiDomainKind int

// TODO: Add or remove from this as required
const (
	System ApiDomainKind = iota
	Vendor
	SystemApex
	VendorApex
)

type ApiDomain struct {
	Name string
	Kind ApiDomainKind
	// The API surfaces that are available to this API domain
	AvailableApiSurfaces []ApiSurface
}

func (apiDomain ApiDomain) Equals(otherApiDomain ApiDomain) bool {
	return apiDomain.Name == otherApiDomain.Name
}

// TODO: Add desc
type ApiDomainModule interface {
	SourceApiDomain() ApiDomain
	SetSourceApiDomain(apiDomain ApiDomain)
}

// TODO: These objects should be expressed in a higher-level language (maybe Android.bp) and not hardcoded in Soong
var (
	apiLevel29       = ApiLevel{value: "29", number: 29, isPreview: false}
	PublicApiSurface = ApiSurface{Name: "public", Version: apiLevel29} // TODO: Fix fix fix
	LlndkApiSurface  = ApiSurface{Name: "llndk", Version: apiLevel29}
	SystemApiDomain  = ApiDomain{Name: "system", Kind: System, AvailableApiSurfaces: []ApiSurface{PublicApiSurface}}
	VendorApiDomain  = ApiDomain{Name: "vendor", Kind: Vendor, AvailableApiSurfaces: []ApiSurface{PublicApiSurface, LlndkApiSurface}}

	apiDomainMap = map[string]ApiDomain{
		SystemApiDomain.Name: SystemApiDomain,
		VendorApiDomain.Name: VendorApiDomain,
	}
)

func GetApiDomain(apiDomainName string) ApiDomain {
	apiDomain, exists := apiDomainMap[apiDomainName]
	if !exists {
		panic(fmt.Errorf("Could not find API domain with name: %s\n", apiDomainName))
	}
	return apiDomain
}

type ApiSurface struct {
	Name    string
	Version ApiLevel
	// TODO: Make this an Android.bp module?
	// Write down pros and cons
	// CcLibraries []ApiSurfaceStubLibrary
	// TODO: Expand this list to headers, java libraries, resources libraries etc.
}

type ApiSurfaceStubLibrary interface {
	Name() string
	Stem() string
	ApiSurfaceName() string
	Version() string
	// The API domain that hosts the library on the device
	SourceApiDomain() ApiDomain

	// TODO: What's the right way to do this
	LibraryFactory() ModuleFactory
}

func init() {
	RegisterApiDomainBuildComponents(InitRegistrationContext)
}

func RegisterApiDomainBuildComponents(ctx RegistrationContext) {
	ctx.PreArchMutators(RegisterPreArchMutators)
	ctx.PreDepsMutators(RegisterPreDepsMutators)
	ctx.PostDepsMutators(RegisterPostDepsMutators)
}

// Mutator order is important - handle with care
// TODO: Add more desc
func RegisterPreArchMutators(ctx RegisterMutatorsContext) {
	ctx.TopDown("synthetic_source_library", SyntheticSourceLibraryMutator).Parallel()
}

// TODO: Add more desc
// This ensures that the stem module exists
// Cannot do ctx.Rename since the stem module can have different API surface variants
func SyntheticSourceLibraryMutator(ctx TopDownMutatorContext) {
	module := ctx.Module()
	if stub, ok := module.(ApiSurfaceStubLibrary); ok {
		stem := stub.Stem()
		if stem == "" {
			ctx.PropertyErrorf("stem", "stem is a required field")
		}
		if !ctx.OtherModuleExists(stem) {
			// TODO: Fix this, using globals is bad
			// This creates a single libfoo for (surface,version) libfoo API surface variants
			newKey := NewCustomOnceKey(stem)
			ctx.Config().Once(newKey, func() interface{} {
				props := struct {
					Name             *string
					Vendor_available *bool
				}{
					Name:             &stem,
					Vendor_available: proptools.BoolPtr(true), // TODO: fix
				}
				return ctx.CreateModule(stub.LibraryFactory(), &props)
			})
		}
	}
}

// TODO: come up with better names
func RegisterPreDepsMutators(ctx RegisterMutatorsContext) {
	// TODO: Add more desc
	// Step 1: Create a dependency from "source" to "api_surface" stub
	ctx.BottomUp("source_to_api_surface_dep", SourceToApiSurfaceDepMutator).Parallel()
	// Step 2: Mark the stem library with the ApiDomain it originates from
	ctx.TopDown("source_api_domain", SourceApiDomainMutator).Parallel()
}

type apiSurfaceDependencyTag struct {
	blueprint.BaseDependencyTag
}

var apiSurfaceDepTag apiSurfaceDependencyTag

// TODO: Add desc
func SourceToApiSurfaceDepMutator(ctx BottomUpMutatorContext) {
	module := ctx.Module()
	if stub, ok := module.(ApiSurfaceStubLibrary); ok {
		stem := stub.Stem()
		// TODO: Add version to surface tag?
		ctx.AddReverseDependency(ctx.Module(), apiSurfaceDepTag, stem)
	}
}

// TODO: Add desc
func SourceApiDomainMutator(ctx TopDownMutatorContext) {
	module := ctx.Module()
	if apiDomainModule, ok := module.(ApiDomainModule); ok {
		ctx.VisitDirectDepsWithTag(apiSurfaceDepTag, func(child Module) {
			stub, ok := child.(ApiSurfaceStubLibrary)
			if !ok {
				ctx.ModuleErrorf("Failed to determine Source API domain of %v. Stub from API surface %v does not implement ApiSurfaceStubLibrary interface\n", module.Name(), stub.ApiSurfaceName())
			}
			apiDomainModule.SetSourceApiDomain(stub.SourceApiDomain())
		})
	}
}

func RegisterPostDepsMutators(ctx RegisterMutatorsContext) {
	ctx.BottomUp("replace_source_with_stubs", ReplaceSourceWithStubsMutator).Parallel()
}

func ReplaceSourceWithStubsMutator(ctx BottomUpMutatorContext) {
	module := ctx.Module()
	if stub, ok := module.(ApiSurfaceStubLibrary); ok {
		stem := stub.Stem()
		ctx.ReplaceDependenciesIf(stem, func(from blueprint.Module, tag blueprint.DependencyTag, to blueprint.Module) bool {
			// TODO: There can be more than API surface / API domain, make this more generic
			// TODO: Make "to" an ApiSurfaceStubLinbrary interface
			// TODO: Handle versions
			fromApiDomainModule := from.(Module)
			if toApiDomainModule, ok := to.(Module).(ApiDomainModule); ok {
				// Always use stubs if dependency crosses API domain boundary
				return !fromApiDomainModule.ApiDomain().Equals(toApiDomainModule.SourceApiDomain())
			}
			// TODO: desc
			return false
		})
	}
}
