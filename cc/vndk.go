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
	"errors"
	"sort"
	"strings"
	"sync"

	"android/soong/android"
)

// List of VNDK libraries that have different core variant and vendor variant.
// For these libraries, the vendor variants must be installed even if the device
// has VndkUseCoreVariant set.
var vndkMustUseVendorVariantList []string = []string{
	"android.frameworks.sensorservice@1.0",
	"android.hardware.atrace@1.0",
	"android.hardware.audio.common@5.0",
	"android.hardware.audio.effect@2.0",
	"android.hardware.audio.effect@4.0",
	"android.hardware.audio.effect@5.0",
	"android.hardware.audio@2.0",
	"android.hardware.audio@4.0",
	"android.hardware.audio@5.0",
	"android.hardware.automotive.evs@1.0",
	"android.hardware.automotive.vehicle@2.0",
	"android.hardware.bluetooth.audio@2.0",
	"android.hardware.boot@1.0",
	"android.hardware.broadcastradio@1.0",
	"android.hardware.broadcastradio@1.1",
	"android.hardware.broadcastradio@2.0",
	"android.hardware.camera.device@1.0",
	"android.hardware.camera.device@3.2",
	"android.hardware.camera.device@3.3",
	"android.hardware.camera.device@3.4",
	"android.hardware.camera.provider@2.4",
	"android.hardware.cas.native@1.0",
	"android.hardware.cas@1.0",
	"android.hardware.configstore@1.0",
	"android.hardware.configstore@1.1",
	"android.hardware.contexthub@1.0",
	"android.hardware.drm@1.0",
	"android.hardware.drm@1.1",
	"android.hardware.fastboot@1.0",
	"android.hardware.gatekeeper@1.0",
	"android.hardware.gnss@1.0",
	"android.hardware.graphics.allocator@2.0",
	"android.hardware.graphics.bufferqueue@1.0",
	"android.hardware.graphics.composer@2.1",
	"android.hardware.graphics.composer@2.2",
	"android.hardware.health@1.0",
	"android.hardware.health@2.0",
	"android.hardware.ir@1.0",
	"android.hardware.keymaster@3.0",
	"android.hardware.keymaster@4.0",
	"android.hardware.light@2.0",
	"android.hardware.media.bufferpool@1.0",
	"android.hardware.media.omx@1.0",
	"android.hardware.memtrack@1.0",
	"android.hardware.neuralnetworks@1.0",
	"android.hardware.neuralnetworks@1.1",
	"android.hardware.neuralnetworks@1.2",
	"android.hardware.nfc@1.1",
	"android.hardware.nfc@1.2",
	"android.hardware.oemlock@1.0",
	"android.hardware.power.stats@1.0",
	"android.hardware.power@1.0",
	"android.hardware.power@1.1",
	"android.hardware.radio@1.4",
	"android.hardware.secure_element@1.0",
	"android.hardware.sensors@1.0",
	"android.hardware.soundtrigger@2.0",
	"android.hardware.soundtrigger@2.0-core",
	"android.hardware.soundtrigger@2.1",
	"android.hardware.tetheroffload.config@1.0",
	"android.hardware.tetheroffload.control@1.0",
	"android.hardware.thermal@1.0",
	"android.hardware.tv.cec@1.0",
	"android.hardware.tv.input@1.0",
	"android.hardware.vibrator@1.0",
	"android.hardware.vibrator@1.1",
	"android.hardware.vibrator@1.2",
	"android.hardware.weaver@1.0",
	"android.hardware.wifi.hostapd@1.0",
	"android.hardware.wifi.offload@1.0",
	"android.hardware.wifi.supplicant@1.0",
	"android.hardware.wifi.supplicant@1.1",
	"android.hardware.wifi@1.0",
	"android.hardware.wifi@1.1",
	"android.hardware.wifi@1.2",
	"android.hardwareundtrigger@2.0",
	"android.hardwareundtrigger@2.0-core",
	"android.hardwareundtrigger@2.1",
	"android.hidl.allocator@1.0",
	"android.hidl.token@1.0",
	"android.hidl.token@1.0-utils",
	"android.system.net.netd@1.0",
	"android.system.wifi.keystore@1.0",
	"libaudioroute",
	"libaudioutils",
	"libbinder",
	"libcamera_metadata",
	"libcrypto",
	"libdiskconfig",
	"libdumpstateutil",
	"libexpat",
	"libfmq",
	"libgui",
	"libhidlcache",
	"libmedia_helper",
	"libmedia_omx",
	"libmemtrack",
	"libnetutils",
	"libpuresoftkeymasterdevice",
	"libradio_metadata",
	"libselinux",
	"libsoftkeymasterdevice",
	"libsqlite",
	"libssl",
	"libstagefright_bufferqueue_helper",
	"libstagefright_flacdec",
	"libstagefright_foundation",
	"libstagefright_omx",
	"libstagefright_omx_utils",
	"libstagefright_soft_aacdec",
	"libstagefright_soft_aacenc",
	"libstagefright_soft_amrdec",
	"libstagefright_soft_amrnbenc",
	"libstagefright_soft_amrwbenc",
	"libstagefright_soft_avcdec",
	"libstagefright_soft_avcenc",
	"libstagefright_soft_flacdec",
	"libstagefright_soft_flacenc",
	"libstagefright_soft_g711dec",
	"libstagefright_soft_gsmdec",
	"libstagefright_soft_hevcdec",
	"libstagefright_soft_mp3dec",
	"libstagefright_soft_mpeg2dec",
	"libstagefright_soft_mpeg4dec",
	"libstagefright_soft_mpeg4enc",
	"libstagefright_soft_opusdec",
	"libstagefright_soft_rawdec",
	"libstagefright_soft_vorbisdec",
	"libstagefright_soft_vpxdec",
	"libstagefright_soft_vpxenc",
	"libstagefright_xmlparser",
	"libsysutils",
	"libui",
	"libvorbisidec",
	"libxml2",
	"libziparchive",
}

type VndkProperties struct {
	Vndk struct {
		// declared as a VNDK or VNDK-SP module. The vendor variant
		// will be installed in /system instead of /vendor partition.
		//
		// `vendor_vailable` must be explicitly set to either true or
		// false together with `vndk: {enabled: true}`.
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

		// Extending another module
		Extends *string
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

func (vndk *vndkdep) isVndkExt() bool {
	return vndk.Properties.Vndk.Extends != nil
}

func (vndk *vndkdep) getVndkExtendsModuleName() string {
	return String(vndk.Properties.Vndk.Extends)
}

func (vndk *vndkdep) typeName() string {
	if !vndk.isVndk() {
		return "native:vendor"
	}
	if !vndk.isVndkExt() {
		if !vndk.isVndkSp() {
			return "native:vendor:vndk"
		}
		return "native:vendor:vndksp"
	}
	if !vndk.isVndkSp() {
		return "native:vendor:vndkext"
	}
	return "native:vendor:vndkspext"
}

func (vndk *vndkdep) vndkCheckLinkType(ctx android.ModuleContext, to *Module, tag dependencyTag) {
	if to.linker == nil {
		return
	}
	if !vndk.isVndk() {
		// Non-VNDK modules (those installed to /vendor) can't depend on modules marked with
		// vendor_available: false.
		violation := false
		if lib, ok := to.linker.(*llndkStubDecorator); ok && !Bool(lib.Properties.Vendor_available) {
			violation = true
		} else {
			if _, ok := to.linker.(libraryInterface); ok && to.VendorProperties.Vendor_available != nil && !Bool(to.VendorProperties.Vendor_available) {
				// Vendor_available == nil && !Bool(Vendor_available) should be okay since
				// it means a vendor-only library which is a valid dependency for non-VNDK
				// modules.
				violation = true
			}
		}
		if violation {
			ctx.ModuleErrorf("Vendor module that is not VNDK should not link to %q which is marked as `vendor_available: false`", to.Name())
		}
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
	if tag == vndkExtDepTag {
		// Ensure `extends: "name"` property refers a vndk module that has vendor_available
		// and has identical vndk properties.
		if to.vndkdep == nil || !to.vndkdep.isVndk() {
			ctx.ModuleErrorf("`extends` refers a non-vndk module %q", to.Name())
			return
		}
		if vndk.isVndkSp() != to.vndkdep.isVndkSp() {
			ctx.ModuleErrorf(
				"`extends` refers a module %q with mismatched support_system_process",
				to.Name())
			return
		}
		if !Bool(to.VendorProperties.Vendor_available) {
			ctx.ModuleErrorf(
				"`extends` refers module %q which does not have `vendor_available: true`",
				to.Name())
			return
		}
	}
	if to.vndkdep == nil {
		return
	}

	// Check the dependencies of VNDK shared libraries.
	if err := vndkIsVndkDepAllowed(vndk, to.vndkdep); err != nil {
		ctx.ModuleErrorf("(%s) should not link to %q (%s): %v",
			vndk.typeName(), to.Name(), to.vndkdep.typeName(), err)
		return
	}
}

func vndkIsVndkDepAllowed(from *vndkdep, to *vndkdep) error {
	// Check the dependencies of VNDK, VNDK-Ext, VNDK-SP, VNDK-SP-Ext and vendor modules.
	if from.isVndkExt() {
		if from.isVndkSp() {
			if to.isVndk() && !to.isVndkSp() {
				return errors.New("VNDK-SP extensions must not depend on VNDK or VNDK extensions")
			}
			return nil
		}
		// VNDK-Ext may depend on VNDK, VNDK-Ext, VNDK-SP, VNDK-SP-Ext, or vendor libs.
		return nil
	}
	if from.isVndk() {
		if to.isVndkExt() {
			return errors.New("VNDK-core and VNDK-SP must not depend on VNDK extensions")
		}
		if from.isVndkSp() {
			if !to.isVndkSp() {
				return errors.New("VNDK-SP must only depend on VNDK-SP")
			}
			return nil
		}
		if !to.isVndk() {
			return errors.New("VNDK-core must only depend on VNDK-core or VNDK-SP")
		}
		return nil
	}
	// Vendor modules may depend on VNDK, VNDK-Ext, VNDK-SP, VNDK-SP-Ext, or vendor libs.
	return nil
}

var (
	vndkCoreLibraries             []string
	vndkSpLibraries               []string
	llndkLibraries                []string
	vndkPrivateLibraries          []string
	vndkUsingCoreVariantLibraries []string
	vndkLibrariesLock             sync.Mutex
)

// gather list of vndk-core, vndk-sp, and ll-ndk libs
func VndkMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.Enabled() {
		if lib, ok := m.linker.(*llndkStubDecorator); ok {
			vndkLibrariesLock.Lock()
			defer vndkLibrariesLock.Unlock()
			name := strings.TrimSuffix(m.Name(), llndkLibrarySuffix)
			if !inList(name, llndkLibraries) {
				llndkLibraries = append(llndkLibraries, name)
				sort.Strings(llndkLibraries)
			}
			if !Bool(lib.Properties.Vendor_available) {
				if !inList(name, vndkPrivateLibraries) {
					vndkPrivateLibraries = append(vndkPrivateLibraries, name)
					sort.Strings(vndkPrivateLibraries)
				}
			}
		} else {
			lib, is_lib := m.linker.(*libraryDecorator)
			prebuilt_lib, is_prebuilt_lib := m.linker.(*prebuiltLibraryLinker)
			if (is_lib && lib.shared()) || (is_prebuilt_lib && prebuilt_lib.shared()) {
				name := strings.TrimPrefix(m.Name(), "prebuilt_")
				if m.vndkdep.isVndk() && !m.vndkdep.isVndkExt() {
					vndkLibrariesLock.Lock()
					defer vndkLibrariesLock.Unlock()
					if mctx.DeviceConfig().VndkUseCoreVariant() && !inList(name, vndkMustUseVendorVariantList) {
						if !inList(name, vndkUsingCoreVariantLibraries) {
							vndkUsingCoreVariantLibraries = append(vndkUsingCoreVariantLibraries, name)
							sort.Strings(vndkUsingCoreVariantLibraries)
						}
					}
					if m.vndkdep.isVndkSp() {
						if !inList(name, vndkSpLibraries) {
							vndkSpLibraries = append(vndkSpLibraries, name)
							sort.Strings(vndkSpLibraries)
						}
					} else {
						if !inList(name, vndkCoreLibraries) {
							vndkCoreLibraries = append(vndkCoreLibraries, name)
							sort.Strings(vndkCoreLibraries)
						}
					}
					if !Bool(m.VendorProperties.Vendor_available) {
						if !inList(name, vndkPrivateLibraries) {
							vndkPrivateLibraries = append(vndkPrivateLibraries, name)
							sort.Strings(vndkPrivateLibraries)
						}
					}
				}
			}
		}
	}
}
