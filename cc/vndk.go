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
	"io"
	"strings"
	"sync"

	"android/soong/android"
)

type VndkProperties struct {
	Vndk struct {
		// declared as a VNDK or VNDK-SP module. The vendor variant
		// will be installed in /system instead of /vendor partition.
		//
		// `vendor_available: true` must set to together for VNDK
		// modules.
		Enabled *bool

		// declared as a VNDK-SP module, which is a subset of VNDK.
		//
		// `vndk: { enabled: true }` must set together.
		//
		// All these modules are allowed to link to VNDK-SP or LL-NDK
		// modules only. Other dependency will cause link-type errors.
		//
		// If `support_system_process` is not set or set to false,
		// the module is VNDK-core and can link to other VNDK-core,
		// VNDK-SP or LL-NDK modules only.
		Support_system_process *bool
	}
}

type vndkdep struct {
	Properties VndkProperties
}

func (vndk *vndkdep) props() []interface{} {
	return []interface{}{&vndk.Properties}
}

func (vndk *vndkdep) begin(ctx BaseModuleContext) {}

func (vndk *vndkdep) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (vndk *vndkdep) isVndk() bool {
	return Bool(vndk.Properties.Vndk.Enabled)
}

func (vndk *vndkdep) isVndkSp() bool {
	return Bool(vndk.Properties.Vndk.Support_system_process)
}

func (vndk *vndkdep) typeName() string {
	if !vndk.isVndk() {
		return "native:vendor"
	}
	if !vndk.isVndkSp() {
		return "native:vendor:vndk"
	}
	return "native:vendor:vndksp"
}

func (vndk *vndkdep) vndkCheckLinkType(ctx android.ModuleContext, to *Module) {
	if to.linker == nil {
		return
	}
	if lib, ok := to.linker.(*libraryDecorator); !ok || !lib.shared() {
		// Check only shared libraries.
		// Other (static and LL-NDK) libraries are allowed to link.
		return
	}
	if !to.Properties.UseVndk {
		ctx.ModuleErrorf("(%s) should not link to %q which is not a vendor-available library",
			vndk.typeName(), to.Name())
		return
	}
	if to.vndkdep == nil {
		return
	}
	if (vndk.isVndk() && !to.vndkdep.isVndk()) || (vndk.isVndkSp() && !to.vndkdep.isVndkSp()) {
		ctx.ModuleErrorf("(%s) should not link to %q(%s)",
			vndk.typeName(), to.Name(), to.vndkdep.typeName())
		return
	}
}

var (
	vndkCoreLibraries     = []string{}
	vndkSpLibraries       = []string{}
	llndkLibraries        = []string{}
	vndkCoreLibrariesLock sync.Mutex
	vndkSpLibrariesLock   sync.Mutex
	llndkLibrariesLock    sync.Mutex
)

// gather list of vndk-core, vndk-sp, and ll-ndk libs
func vndkMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok {
		if _, ok := m.linker.(*llndkStubDecorator); ok {
			llndkLibrariesLock.Lock()
			name := strings.TrimSuffix(m.Name(), llndkLibrarySuffix)
			if !inList(name, llndkLibraries) {
				llndkLibraries = append(llndkLibraries, name)
			}
			llndkLibrariesLock.Unlock()
		} else if lib, ok := m.linker.(*libraryDecorator); ok && lib.shared() {
			if m.vndkdep.isVndk() {
				if m.vndkdep.isVndkSp() {
					vndkSpLibrariesLock.Lock()
					if !inList(m.Name(), vndkSpLibraries) {
						vndkSpLibraries = append(vndkSpLibraries, m.Name())
					}
					vndkSpLibrariesLock.Unlock()
				} else {
					vndkCoreLibrariesLock.Lock()
					if !inList(m.Name(), vndkCoreLibraries) {
						vndkCoreLibraries = append(vndkCoreLibraries, m.Name())
					}
					vndkCoreLibrariesLock.Unlock()
				}
			}
		}
	}
}

type vndkPackage struct {
	android.ModuleBase
	requiredModuleNames []string
}

func vndkPackageFactory() android.Module {
	module := &vndkPackage{}
	android.InitAndroidModule(module)
	return module
}

func (v *vndkPackage) DepsMutator(ctx android.BottomUpMutatorContext) {
	// TODO(jiyong): add version support
}

func (v *vndkPackage) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	v.requiredModuleNames = append(v.requiredModuleNames, vndkSpLibraries...)
	v.requiredModuleNames = append(v.requiredModuleNames, vndkCoreLibraries...)
	v.requiredModuleNames = addSuffix(v.requiredModuleNames, vendorSuffix)
	v.requiredModuleNames = append(v.requiredModuleNames, llndkLibraries...)
}

func (v *vndkPackage) AndroidMk() (ret android.AndroidMkData, err error) {
	ret.Custom = func(w io.Writer, name, prefix, moduleDir string) error {
		fmt.Fprintln(w, "\ninclude $(CLEAR_VARS)")
		fmt.Fprintln(w, "LOCAL_PATH :=", moduleDir)
		fmt.Fprintln(w, "LOCAL_MODULE :=", name)
		fmt.Fprintln(w, "LOCAL_REQUIRED_MODULES := "+strings.Join(v.requiredModuleNames, " "))
		fmt.Fprintln(w, "include $(BUILD_PHONY_PACKAGE)")

		return nil
	}
	return
}

func init() {
	android.RegisterModuleType("vndk_package", vndkPackageFactory)
}
