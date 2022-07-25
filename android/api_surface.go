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

import (
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type ApiSurface int

// TODO: Reconcile names with java android.SdkKind
// TODO: Add/remove more from this list
const (
	PublicApi ApiSurface = iota
	SystemApi
	VendorApi
)

func (s ApiSurface) String() string {
	switch s {
	case PublicApi:
		return "public"
	case SystemApi:
		return "system"
	case VendorApi:
		return "vendor"
	default:
		return "invalid"
	}
}

type ApiSurfaceStubLibrary interface {
	Name(stem string) string
	ApiSurfaceName() string
	Version() string

	// TODO: fix this
	LibraryFactory() ModuleFactory
}

func init() {
	RegisterApiSurfaceComponents(InitRegistrationContext)
}

func RegisterApiSurfaceComponents(ctx RegistrationContext) {
	ctx.PreArchMutators(RegisterPreArchMutators)
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

// Register a PostDeps mutator that creates a dependency edge from the source module variant to the appropriate stub module
// This must run after the PreDeps mutators `sdk` and `image` have created variations of the source module
func RegisterPostDepsMutators(ctx RegisterMutatorsContext) {
	ctx.BottomUp("stub_to_source_rdep", StubToSourceRdepMutator).Parallel()
}

type apiSurfaceDependencyTag struct {
	blueprint.BaseDependencyTag
}

var ApiSurfaceDepTag apiSurfaceDependencyTag

func StubToSourceRdepMutator(ctx BottomUpMutatorContext) {
	module := ctx.Module()

	if shadowed, ok := module.(ApiSurfaceStubReplacable); ok {
		variations := []blueprint.Variation{
			{Mutator: "os", Variation: ctx.Os().String()},
			{Mutator: "arch", Variation: ctx.Arch().String()},
		}
		stubLibrary := shadowed.ReplaceWith()
		if stubLibrary != "" {
			// TODO: Make this an error?
			if ctx.OtherModuleExists(stubLibrary) {
				ctx.AddFarVariationDependencies(variations, ApiSurfaceDepTag, stubLibrary)
			}
		}
	}
}

type ApiSurfaceStubReplacable interface {
	ReplaceWith() string
}
