// Copyright 2023 Google Inc. All rights reserved.
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

	"android/soong/android"
)

var (
	// Add flags to ignore warnings about symbols not be found
	// or not allowed to be ordered
	orderfileOtherFlags = []string{
		"-Wl,--no-warn-symbol-ordering",
	}

	// Add folder projects for orderfiles
	globalOrderfileProjects = []string{
		"toolchain/pgo-profiles/orderfiles",
		"vendor/google_data/pgo_profile/orderfiles",
	}
)

var orderfileProjectsConfigKey = android.NewOnceKey("OrderfileProjects")

const orderfileProfileFlag = "-forder-file-instrumentation"
const orderfileUseFormat = "-Wl,--symbol-ordering-file=%s"

func getOrderfileProjects(config android.DeviceConfig) []string {
	return config.OnceStringSlice(orderfileProjectsConfigKey, func() []string {
		return append(globalOrderfileProjects)
	})
}

func recordMissingOrderfile(ctx BaseModuleContext, missing string) {
	getNamedMapForConfig(ctx.Config(), modulesMissingProfileFileKey).Store(missing, true)
}

type OrderfileProperties struct {
	Orderfile struct {
		Instrumentation    *bool
		Order_file_path    *string `android:"arch_variant"`
		Load_order_file    *bool `android:"arch_variant"`
		// Additional compiler flags to use when building this module
		// for orderfile profiling.
		Cflags []string `android:"arch_variant"`
	} `android:"arch_variant"`

	ShouldProfileModule 	  bool `blueprint:"mutated"`
	OrderfileLoad             bool `blueprint:"mutated"`
	OrderfileInstrLink        bool `blueprint:"mutated"`
}

type orderfile struct {
	Properties OrderfileProperties
}

func (props *OrderfileProperties) shouldInstrument() bool {
	return props.Orderfile.Instrumentation != nil && *props.Orderfile.Instrumentation == true
}

// ShouldLoadOrderfile returns true if we need to load the order file rather than
// profile the binary or shared library
func (props *OrderfileProperties) shouldLoadOrderfile() bool {
	return props.Orderfile.Load_order_file != nil && *props.Orderfile.Load_order_file == true && props.Orderfile.Order_file_path != nil
}

// orderfileEnabled returns true for binaries and shared libraries
// if instrument flag is set to true
func (orderfile *orderfile) orderfileEnabled() bool {
	return orderfile != nil && orderfile.Properties.shouldInstrument()
}

// orderfileEnabled returns true for binaries and shared libraries
// if you should instrument dependencies
func (orderfile *orderfile) orderfileLinkEnabled() bool {
	return orderfile != nil && orderfile.Properties.OrderfileInstrLink
}

func (orderfile *orderfile) props() []interface{} {
	return []interface{}{&orderfile.Properties}
}

func (props *OrderfileProperties) getOrderfileFile(ctx BaseModuleContext) android.OptionalPath {
	orderFile := *props.Orderfile.Order_file_path

	// Test if the order file is present in any of the Orderfile projects
	for _, profileProject := range getOrderfileProjects(ctx.DeviceConfig()) {
		path := android.ExistentPathForSource(ctx, profileProject, orderFile)
		if path.Valid() {
			return path
		}
	}

	// Record that this module's order file is absent
	missing := *props.Orderfile.Order_file_path + ":" + ctx.ModuleDir() + "/Android.bp:" + ctx.ModuleName()
	recordMissingOrderfile(ctx, missing)

	return android.OptionalPathForPath(nil)
}

func (props *OrderfileProperties) addInstrumentationProfileGatherFlags(ctx ModuleContext, flags Flags) Flags {
	// Add to C flags iff Orderfile with Orderfile is explicitly enabled for this module.
	flags.Local.CFlags = append(flags.Local.CFlags, props.Orderfile.Cflags...)
	flags.Local.CFlags = append(flags.Local.CFlags, orderfileProfileFlag)
	flags.Local.CFlags = append(flags.Local.CFlags, "-mllvm")
	flags.Local.CFlags = append(flags.Local.CFlags, "-enable-order-file-instrumentation")
	flags.Local.LdFlags = append(flags.Local.LdFlags, orderfileProfileFlag)
	return flags
}

func (props *OrderfileProperties) loadOrderfileFlags(ctx ModuleContext, file string) []string {
	flags := []string{fmt.Sprintf(orderfileUseFormat, file)}
	flags = append(flags, orderfileOtherFlags...)
	return flags
}

func (props *OrderfileProperties) addLoadFlags(ctx ModuleContext, flags Flags) Flags {
	orderFile := props.getOrderfileFile(ctx)
	orderFilePath := orderFile.Path()
	loadFlags := props.loadOrderfileFlags(ctx, orderFilePath.String())

	flags.Local.CFlags = append(flags.Local.CFlags, loadFlags...)
	flags.Local.LdFlags = append(flags.Local.LdFlags, loadFlags...)

	// Update CFlagsDeps and LdFlagsDeps so the module is rebuilt
	// if orderfile gets updated
	flags.CFlagsDeps = append(flags.CFlagsDeps, orderFilePath)
	flags.LdFlagsDeps = append(flags.LdFlagsDeps, orderFilePath)
	return flags
}

func (orderfile *orderfile) begin(ctx BaseModuleContext) {
	if ctx.Host() {
		return
	}

	if ctx.static() && !ctx.staticBinary() {
		return
	}

	if ctx.DeviceConfig().ClangCoverageEnabled() {
		return
	}

	if !orderfile.orderfileEnabled() {
		return
	}

	orderfile.Properties.OrderfileLoad = orderfile.Properties.shouldLoadOrderfile()
	orderfile.Properties.ShouldProfileModule = !orderfile.Properties.shouldLoadOrderfile()
	orderfile.Properties.OrderfileInstrLink = orderfile.orderfileEnabled() && !orderfile.Properties.shouldLoadOrderfile()
}

func (orderfile *orderfile) flags(ctx ModuleContext, flags Flags) Flags {
	props := orderfile.Properties
	// If orderfile is enabled, we will load the orderfile using the path in its Android.bp
	if orderfile.Properties.OrderfileLoad {
		flags = props.addLoadFlags(ctx, flags)

		return flags
	}

	// Add flags to profile this module if it came from above dependencies or extracted
	// from Android.bp. If we reach here, it means we do not need to load any orderfile
	// because profiling and loading cannot happen at the same time
	if props.ShouldProfileModule {
		flags = props.addInstrumentationProfileGatherFlags(ctx, flags)

		return flags
	}

	return flags
}

// Propagate orderfile requirements down from binaries and shared libraries
func orderfileDepsMutator(mctx android.TopDownMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok {
		if !m.orderfile.orderfileLinkEnabled() {
			return
		}
		mctx.WalkDeps(func(dep android.
			Module, parent android.Module) bool {
			tag := mctx.OtherModuleDependencyTag(dep)
			libTag, isLibTag := tag.(libraryDependencyTag)

			// Do not recurse down non-static dependencies
			if isLibTag {
				if !libTag.static() {
					return false
				}
			} else {
				if tag != objDepTag && tag != reuseObjTag {
					return false
				}
			}

			if dep, ok := dep.(*Module); ok {
				if m.orderfile.Properties.OrderfileInstrLink {
					dep.orderfile.Properties.OrderfileInstrLink = true;
				}
			}

			return true
		})
	}
}

// Create orderfile variants for modules that need them
func orderfileMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.orderfile != nil {
		if !m.static() && m.orderfile.orderfileEnabled() {
			mctx.SetDependencyVariation("orderfile")
			return
		}

		variationNames := []string{""}
		if m.orderfile.Properties.OrderfileInstrLink {
			variationNames = append(variationNames, "orderfile")
		}

		if len(variationNames) > 1 {
			modules := mctx.CreateVariations(variationNames...)
			for i, name := range variationNames {
				if name == "" {
					continue
				}
				variation := modules[i].(*Module)
				variation.Properties.PreventInstall = true
				variation.Properties.HideFromMake = true
				variation.orderfile.Properties.ShouldProfileModule = true
				variation.orderfile.Properties.OrderfileLoad = false
			}
		}
	}
}