// Copyright (C) 2019 The Android Open Source Project
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

package sdk

import (
	"strconv"

	"android/soong/android"
	// This package don't depend on the apex package, but importing it to make its mutators to be
	// registered before mutators in this package. See RegisterPostDepsMutators for more details.
	_ "android/soong/apex"
	"android/soong/cc"
	"android/soong/java"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	android.RegisterModuleType("sdk", ModuleFactory)
	android.PreArchMutators(RegisterPreArchMutators)
	android.PostDepsMutators(RegisterPostDepsMutators)
}

type sdk struct {
	android.ModuleBase
	android.DefaultableModuleBase
	// TODO(jiyong): we might need some properties
}

// ModuleFactory creates and initializes an 'sdk' module which is a logical group of modules
// (e.g. native libs, headers, java libs, etc.) which Mainline modules (e.g. apex and apk) can
// choose to build with.
func ModuleFactory() android.Module {
	s := &sdk{}
	android.InitAndroidModule(s)
	android.InitDefaultableModule(s)
	return s
}

func (s *sdk) DepsMutator(ctx android.BottomUpMutatorContext) {
	// Do nothing. The dependencies from an sdk module to its members are created in memberMutator
}

func (s *sdk) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// TODO(jiyong): add build rules for creating stubs from members of this SDK
}

// RegisterPreArchMutators registers pre-arch mutators to support modules implementing SdkAware
// interface and the sdk module type. This function has been made public to be called by tests
// outside of the sdk package
func RegisterPreArchMutators(ctx android.RegisterMutatorsContext) {
	// Note: why pre-arch? The new modules could be created in SdkCreateMissingMember. The newly
	// created modules should also be mutated for arch.
	ctx.BottomUp("SdkMember", memberMutator).Parallel()
	ctx.TopDown("SdkCreateMissingMember", createMissingMemberMutator).Parallel()
	ctx.BottomUp("SdkMemberInterVersion", memberInterVersionMutator).Parallel()
}

// RegisterPostDepshMutators registers post-deps mutators to support modules implementing SdkAware
// interface and the sdk module type. This function has been made public to be called by tests
// outside of the sdk package
func RegisterPostDepsMutators(ctx android.RegisterMutatorsContext) {
	// These must run AFTER apexMutator. Note that the apex package is imported even though there is
	// no direct dependency to the package here. sdkDepsMutator sets the SDK requirements from an
	// APEX to its dependents. Since different versions of the same SDK can be used by different
	// APEXes, the apex and its dependents (which includes the dependencies to the sdk members)
	// should have been mutated for the apex before the SDK requirements are set.
	ctx.TopDown("SdkDepsMutator", sdkDepsMutator).Parallel()
	ctx.BottomUp("SdkDepsReplaceMutator", sdkDepsReplaceMutator).Parallel()
}

type dependencyTag struct {
	blueprint.BaseDependencyTag
}

// For dependencies from an SDK module to its members
// e.g. mysdk -> libfoo and libbar
var sdkMemberDepTag dependencyTag

// For dependencies from a dev version of an SDK member to frozen versions of the same member
// e.g. libfoo -> libfoo.mysdk.11 and libfoo.mysdk.12
var sdkMemberVersionedDepTag dependencyTag

// Step 1: create dependencies from an SDK module to its members.
func memberMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(android.SdkAware); ok && m.IsInAnySdk() {
		sdkName := m.ContainingSdk().Name
		if !mctx.OtherModuleExists(sdkName) {
			mctx.PropertyErrorf("sdk.name", "sdk %q does not exist", sdkName)
			return
		}
		mctx.AddReverseDependency(mctx.Module(), sdkMemberDepTag, sdkName)
	}
}

// Step 2: ensure that there always exists the in-development version of an SDK member, by creating
// one if the member module is not defined elsewhere. The module is required because the names of
// the frozen version of the sdk member are mangled (libfoo.mysdk.22 instead of libfoo) and thus
// any reference to the unmangled name will cause dependency error if a module having the unmangled
// name (which is the case for in-development version) does not exist.
func createMissingMemberMutator(mctx android.TopDownMutatorContext) {
	if _, ok := mctx.Module().(*sdk); ok {
		sdkName := mctx.ModuleName()

		// map from an SDK member name to a bool telling if the dev version of the member is defined
		hasDevVersion := make(map[string]bool)
		mctx.VisitDirectDepsWithTag(sdkMemberDepTag, func(m android.Module) {
			if member, ok := m.(android.SdkAware); ok {
				memberName := m.BaseModuleName()
				if _, ok := hasDevVersion[memberName]; !ok {
					hasDevVersion[memberName] = false
				}
				if member.ContainingSdk().IsDevVersionOf(sdkName) {
					hasDevVersion[memberName] = true
				}
			}
		})

		// map from an SDK member name to its module type
		moduleTypeOf := make(map[string]string)
		mctx.VisitDirectDepsWithTag(sdkMemberDepTag, func(m android.Module) {
			if member, ok := m.(android.SdkAware); ok {
				memberName := m.BaseModuleName()
				// If we have the dev version of an SDK member defined, no need to care about its
				// type because we won't create a module here
				if hasDevVersion[memberName] {
					return
				}

				typeName := mctx.OtherModuleType(member)
				if _, ok := moduleTypeOf[memberName]; !ok {
					moduleTypeOf[memberName] = typeName
				}

				// TODO(jiyong): have a bottom-up mutator to ensure that all modules having the same
				// member name are with the same module type.
			}
		})

		for memberName, isDev := range hasDevVersion {
			if !isDev {
				moduleType := moduleTypeOf[memberName]
				switch moduleType {
				case "java_import":
					createJavaModule(mctx, memberName, sdkName)
				case "cc_prebuilt_library_shared":
					createCcLibrarySharedModule(mctx, memberName, sdkName)
				case "cc_prebuilt_library_static":
					createCcLibraryStaticModule(mctx, memberName, sdkName)
				}
				// TODO(jiyong): add support for more module types
			}
		}
	}
}

func createJavaModule(mctx android.TopDownMutatorContext, memberName string, sdkName string) {
	props := struct {
		Name         *string
		Provides_sdk struct {
			Name *string
			Fake bool
		}
	}{}
	props.Name = proptools.StringPtr(memberName)
	props.Provides_sdk.Name = proptools.StringPtr(sdkName)
	props.Provides_sdk.Fake = true
	// provides_sdk.version is omitted; defaults to 'dev'
	mctx.CreateModule(android.ModuleFactoryAdaptor(java.ImportFactory), &props)
}

func createCcLibrarySharedModule(mctx android.TopDownMutatorContext, memberName string, sdkName string) {
	props := struct {
		Name               *string
		System_shared_libs []string
		Stl                *string
		Provides_sdk       struct {
			Name *string
			Fake bool
		}
		Enabled *bool
	}{}
	props.Name = proptools.StringPtr(memberName)
	props.System_shared_libs = []string{}
	props.Stl = proptools.StringPtr("none")
	props.Provides_sdk.Name = proptools.StringPtr(sdkName)
	props.Provides_sdk.Fake = true
	// provides_sdk.version is omitted; defaults to 'dev'
	mctx.CreateModule(android.ModuleFactoryAdaptor(cc.PrebuiltSharedLibraryFactory), &props)
}

func createCcLibraryStaticModule(mctx android.TopDownMutatorContext, memberName string, sdkName string) {
	props := struct {
		Name               *string
		System_shared_libs []string
		Stl                *string
		Provides_sdk       struct {
			Name *string
			Fake bool
		}
	}{}
	props.Name = proptools.StringPtr(memberName)
	props.System_shared_libs = []string{}
	props.Stl = proptools.StringPtr("none")
	props.Provides_sdk.Name = proptools.StringPtr(sdkName)
	props.Provides_sdk.Fake = true
	// provides_sdk.version is omitted; defaults to 'dev'
	mctx.CreateModule(android.ModuleFactoryAdaptor(cc.PrebuiltStaticLibraryFactory), &props)
}

// Step 3: create dependencies from the in-development version of an SDK member to frozen versions
// of the same member. By having these dependencies, they are mutated for multiple Mainline modules
// (apex and apk), each of which might want different sdks to be built with. For example, if both
// apex A and B are referencing libfoo which is a member of sdk 'mysdk', the two APEXes can be
// built with libfoo.mysdk.11 and libfoo.mysdk.12, respectively depending on which sdk they are
// using.
func memberInterVersionMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(android.SdkAware); ok && m.IsInAnySdk() {
		if !m.ContainingSdk().IsDevVersion() {
			memberName := m.BaseModuleName()
			mctx.AddReverseDependency(mctx.Module(), sdkMemberVersionedDepTag, memberName)
		}
	}
}

// Step 4: transitively ripple down the SDK requirements from the root modules like APEX to its
// descendants
func sdkDepsMutator(mctx android.TopDownMutatorContext) {
	if m, ok := mctx.Module().(android.SdkAware); ok {
		// Module types for Mainline modules (e.g. APEX) are expected to implement RequiredSdks()
		// by reading its own properties like `uses_sdks`.
		requiredSdks := m.RequiredSdks()
		if len(requiredSdks) > 0 {
			mctx.VisitDirectDeps(func(m android.Module) {
				if dep, ok := m.(android.SdkAware); ok {
					dep.BuildWithSdks(requiredSdks)
				}
			})
		}
		// This is a tricky part. I wish I could remove this. Code below is to prevent the auto
		// generated in-development module (if ever created) from being actually used. The module is
		// created only when there is no 'real' module defined for the in-development version for
		// an SDK, just to satisfy dependencies to the module name until the dependencies are
		// replaced to the correct prebuilts by the sdkDepsReplaceMutator below. This means that
		// after sdkDepsRelaceMutator is finished, there should be no dependency to the auto
		// generated module. If there is, the build won't be successful. For example the cc
		// package will complain since the auto-generated module does not produce any output.
		//
		// But note that the dependency replacement is only when there is a sdk requirement;
		// such as when building modules for an APEX having uses_sdks property set. So, for the
		// modules built without sdk requirements, the dependency to the auto-generated module
		// could persist.
		//
		// To prevent this, the auto-generated module (that we call a fake module here) tries to
		// find the latest prebuilt for the SDK and tell it to replace me (the fake module) in
		// sdkDepsReplaceMutator.
		if m.IsFakeModule() && !m.ContainedInRequiredSdks( /* ignoreVersion */ true) {
			latestVerNum := 0
			var latestSdkMember android.SdkAware
			mctx.VisitDirectDeps(func(d android.Module) {
				if dep, ok := d.(android.SdkAware); ok && m.IsInSameSdk(dep) {
					verNum, err := strconv.Atoi(dep.ContainingSdk().Version)
					if err != nil {
						mctx.ModuleErrorf("SDK version %q of %q is not a number",
							dep.ContainingSdk().Version, dep.Name())
						return
					}
					if verNum > latestVerNum {
						latestVerNum = verNum
						latestSdkMember = dep
					}
				}
			})
			if latestSdkMember != nil {
				latestSdkMember.ReplaceFakeModule()
			}
		}
	}
}

// Step 5: if libfoo.mysdk.11 is in the context where version 11 of mysdk is requested, the
// versioned module is used instead of the un-versioned (in-development) module libfoo
func sdkDepsReplaceMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(android.SdkAware); ok && m.IsInAnySdk() {
		sdk := m.ContainingSdk()
		if !sdk.IsDevVersion() {
			if m.ContainedInRequiredSdks( /* ignoreVersion */ false) || m.ShouldReplaceFakeModule() {
				// Note that this replacement is done only for the modules that have the same
				// variations as the current module. Since current module is already mutated for
				// apex references in other APEXes are not affected by this replacement.
				memberName := m.BaseModuleName()
				mctx.ReplaceDependencies(memberName)
			}
		}
	}
}
