// Copyright 2017 Google Inc. All rights reserved.
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

package cc

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

var (
	// Add flags to ignore warnings about symbols not be found
	// or not allowed to be ordered
	orderfileProfileUseOtherFlags = []string{
		"-Wl,--no-warn-symbol-ordering",
	}

	// Add folder projects for orderfiles
	globalOrderfileProfileProjects = []string{
		"toolchain/pgo-profiles/orderfiles",
		"vendor/google_data/pgo_profile/orderfiles",
	}
)

var orderfileProfileProjectsConfigKey = android.NewOnceKey("OrderfileProfileProjects")

const orderfileProfileInstrumentFlag = "-forder-file-instrumentation"
const orderfileProfileUseInstrumentFormat = "-Wl,--symbol-ordering-file=%s"

func getOrderfileProfileProjects(config android.DeviceConfig) []string {
	return config.OnceStringSlice(orderfileProfileProjectsConfigKey, func() []string {
		return append(globalOrderfileProfileProjects)
	})
}

func recordMissingorderfileProfileFile(ctx BaseModuleContext, missing string) {
	getNamedMapForConfig(ctx.Config(), modulesMissingProfileFileKey).Store(missing, true)
}

type OrderfileProperties struct {
	Orderfile struct {
		Instrumentation    *bool
		Profile_file       *string `android:"arch_variant"`
		Benchmarks         []string
		Enable_profile_use *bool `android:"arch_variant"`
		// Additional compiler flags to use when building this module
		// for orderfile profiling.
		Cflags []string `android:"arch_variant"`
	} `android:"arch_variant"`

	OrderfilePresent          bool `blueprint:"mutated"`
	ShouldProfileModule 	  bool `blueprint:"mutated"`
	OrderfileCompile          bool `blueprint:"mutated"`
	OrderfileInstrLink        bool `blueprint:"mutated"`
}

type orderfile struct {
	Properties OrderfileProperties
}

func (props *OrderfileProperties) isInstrumentation() bool {
	return props.Orderfile.Instrumentation != nil && *props.Orderfile.Instrumentation == true
}

func (orderfile *orderfile) props() []interface{} {
	return []interface{}{&orderfile.Properties}
}

func (props *OrderfileProperties) addInstrumentationProfileGatherFlags(ctx ModuleContext, flags Flags) Flags {
	// Add to C flags iff Orderfile with Orderfile is explicitly enabled for this module.
	orderfileBenchmarks := props.Orderfile.Benchmarks
	fmt.Println("0-------------------------------------------------------------------------------------0")
	fmt.Println("orderfileBenchmarks: ")
	fmt.Println(orderfileBenchmarks)
	fmt.Printf("Inside Add Instrument Profile Gather Flags: %b\n", props.ShouldProfileModule)
	
	fmt.Println("Before Cflags Flags: ")
	fmt.Println(flags.Local.CFlags)
	fmt.Println("Before Linker Flags: ")
	fmt.Println(flags.Local.LdFlags)
	if props.ShouldProfileModule {
		flags.Local.CFlags = append(flags.Local.CFlags, props.Orderfile.Cflags...)
		flags.Local.CFlags = append(flags.Local.CFlags, orderfileProfileInstrumentFlag)
		flags.Local.CFlags = append(flags.Local.CFlags, "-mllvm")
		flags.Local.CFlags = append(flags.Local.CFlags, "-enable-order-file-instrumentation")
	}
	flags.Local.LdFlags = append(flags.Local.LdFlags, orderfileProfileInstrumentFlag)
	fmt.Println("After Cflags Flags: ")
	fmt.Println(flags.Local.CFlags)
	fmt.Println("AFter Linker Flags: ")
	fmt.Println(flags.Local.LdFlags)
	fmt.Println("1-------------------------------------------------------------------------------------1")
	return flags
}

func (props *OrderfileProperties) getOrderfileProfileFile(ctx BaseModuleContext) android.OptionalPath {
	profileFile := *props.Orderfile.Profile_file

	// Test if the profile_file is present in any of the Orderfile profile projects
	for _, profileProject := range getOrderfileProfileProjects(ctx.DeviceConfig()) {
		// Bug: http://b/74395273 If the profile_file is unavailable,
		// use a versioned file named
		// <profile_file>.<arbitrary-version> when available.  This
		// works around an issue where ccache serves stale cache
		// entries when the profile file has changed.
		globPattern := filepath.Join(profileProject, profileFile+".*")
		versionedProfiles, err := ctx.GlobWithDeps(globPattern, nil)
		if err != nil {
			ctx.ModuleErrorf("glob: %s", err.Error())
		}

		path := android.ExistentPathForSource(ctx, profileProject, profileFile)
		if path.Valid() {
			if len(versionedProfiles) != 0 {
				ctx.PropertyErrorf("orderfile.profile_file", "Orderfile has multiple versions: "+filepath.Join(profileProject, profileFile)+", "+strings.Join(versionedProfiles, ", "))
			}
			return path
		}

		if len(versionedProfiles) > 1 {
			ctx.PropertyErrorf("orderfile.profile_file", "Orderfile has multiple versions: "+strings.Join(versionedProfiles, ", "))
		} else if len(versionedProfiles) == 1 {
			return android.OptionalPathForPath(android.PathForSource(ctx, versionedProfiles[0]))
		}
	}

	// Record that this module's profile file is absent
	missing := *props.Orderfile.Profile_file + ":" + ctx.ModuleDir() + "/Android.bp:" + ctx.ModuleName()
	recordMissingorderfileProfileFile(ctx, missing)

	return android.OptionalPathForPath(nil)
}

func (props *OrderfileProperties) profileUseFlags(ctx ModuleContext, file string) []string {
	flags := []string{fmt.Sprintf(orderfileProfileUseInstrumentFormat, file)}
	flags = append(flags, orderfileProfileUseOtherFlags...)
	return flags
}

func (props *OrderfileProperties) addProfileUseFlags(ctx ModuleContext, flags Flags) Flags {
	// Return if 'orderfile' property is not present in this module.
	if !props.OrderfilePresent {
		return flags
	}

	if props.OrderfileCompile {
		profileFile := props.getOrderfileProfileFile(ctx)
		profileFilePath := profileFile.Path()
		profileUseFlags := props.profileUseFlags(ctx, profileFilePath.String())

		flags.Local.LdFlags = append(flags.Local.LdFlags, profileUseFlags...)

		// Update CFlagsDeps and LdFlagsDeps so the module is rebuilt
		// if profileFile gets updated
		flags.CFlagsDeps = append(flags.CFlagsDeps, profileFilePath)
		flags.LdFlagsDeps = append(flags.LdFlagsDeps, profileFilePath)
	}
	return flags
}

func (props *OrderfileProperties) isOrderfile(ctx BaseModuleContext) bool {
	isInstrumentation := props.isInstrumentation()

	profileKindPresent := isInstrumentation
	filePresent := props.Orderfile.Profile_file != nil
	benchmarksPresent := len(props.Orderfile.Benchmarks) > 0

	// If all three properties are absent, Orderfile is OFF for this module
	if !profileKindPresent && !filePresent && !benchmarksPresent {
		return false
	}

	// profileKindPresent and filePresent are mandatory properties.
	if !profileKindPresent || !filePresent {
		var missing []string
		if !profileKindPresent {
			missing = append(missing, "profile kind")
		}
		if !filePresent {
			missing = append(missing, "profile_file property")
		}
		missingProps := strings.Join(missing, ", ")
		ctx.ModuleErrorf("Orderfile specification is missing properties: " + missingProps)
	}

	// Benchmark property is mandatory for instrumentation Orderfile.
	if isInstrumentation && !benchmarksPresent {
		ctx.ModuleErrorf("Instrumentation Orderfile specification is missing benchmark property")
	}

	return true
}

func (orderfile *orderfile) begin(ctx BaseModuleContext) {
	// Currently PGO does not support host modules so we are doing the same for Orderfiles
	if ctx.Host() {
		return
	}

	// Check if Orderfile is needed for this module
	orderfile.Properties.OrderfilePresent = orderfile.Properties.isOrderfile(ctx)

	if !orderfile.Properties.OrderfilePresent {
		return
	}

	// This module should be instrumented if ANDROID_ORDERFILE_INSTRUMENT is set
	// and includes 'all', 'ALL' or a benchmark listed for this module.
	orderfile.Properties.ShouldProfileModule = false
	orderfileBenchmarks := ctx.Config().Getenv("ANDROID_ORDERFILE_INSTRUMENT")
	orderfileBenchmarksMap := make(map[string]bool)
	for _, b := range strings.Split(orderfileBenchmarks, ",") {
		orderfileBenchmarksMap[b] = true
	}
	if orderfileBenchmarksMap["all"] == true || orderfileBenchmarksMap["ALL"] == true {
		orderfile.Properties.ShouldProfileModule = true
		orderfile.Properties.OrderfileInstrLink = orderfile.Properties.isInstrumentation()
	} else {
		for _, b := range orderfile.Properties.Orderfile.Benchmarks {
			if orderfileBenchmarksMap[b] == true {
				orderfile.Properties.ShouldProfileModule = true
				orderfile.Properties.OrderfileInstrLink = orderfile.Properties.isInstrumentation()
				break
			}
		}
	}
	fmt.Printf("Orderfile Should Profile: %b\n", orderfile.Properties.ShouldProfileModule);

	// Orderfile profile use is not feasible for a Clang coverage build because
	// -forderfile-file-instrumentation are incompatible.
	if ctx.DeviceConfig().ClangCoverageEnabled() {
		return
	}

	if !ctx.Config().IsEnvTrue("ANDROID_ORDERFILE_NO_PROFILE_USE") &&
		proptools.BoolDefault(orderfile.Properties.Orderfile.Enable_profile_use, true) {
		if profileFile := orderfile.Properties.getOrderfileProfileFile(ctx); profileFile.Valid() {
			orderfile.Properties.OrderfileCompile = true
		}
	}
}

func (orderfile *orderfile) flags(ctx ModuleContext, flags Flags) Flags {
	if ctx.Host() {
		return flags
	}

	// Deduce OrderfileInstrLink property i.e. whether this module needs to be
	// linked with profile-generation flags.  Here, we're setting it if any
	// dependency needs Orderfile instrumentation.  It is initially set in
	// begin() if Orderfile is directly enabled for this module.
	if ctx.static() && !ctx.staticBinary() {
		// For static libraries, check if any whole_static_libs are
		// linked with profile generation
		ctx.VisitDirectDeps(func(m android.Module) {
			if depTag, ok := ctx.OtherModuleDependencyTag(m).(libraryDependencyTag); ok {
				if depTag.static() && depTag.wholeStatic {
					if cc, ok := m.(*Module); ok {
						if cc.orderfile.Properties.OrderfileInstrLink {
							orderfile.Properties.OrderfileInstrLink = true
						}
					}
				}
			}
		})
	} else {
		// For executables and shared libraries, check all static dependencies.
		ctx.VisitDirectDeps(func(m android.Module) {
			if depTag, ok := ctx.OtherModuleDependencyTag(m).(libraryDependencyTag); ok {
				if depTag.static() {
					if cc, ok := m.(*Module); ok {
						if cc.orderfile.Properties.OrderfileInstrLink {
							orderfile.Properties.OrderfileInstrLink = true
						}
					}
				}
			}
		})
	}

	props := orderfile.Properties
	// Add flags to profile this module based on its profile_kind
	if (props.ShouldProfileModule && props.isInstrumentation()) || props.OrderfileInstrLink {
		fmt.Printf("Adding Instrumentation Flags\n")
		// Instrumentation Orderfile use and gather flags cannot coexist.
		return props.addInstrumentationProfileGatherFlags(ctx, flags)
	}

	if !ctx.Config().IsEnvTrue("ANDROID_ORDERFILE_NO_PROFILE_USE") {
		flags = props.addProfileUseFlags(ctx, flags)
	}

	return flags
}
