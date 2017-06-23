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
	"android/soong/android"
	"github.com/google/blueprint"
)

type VndkTag string

const (
	VNDK_TAG_NONE VndkTag = "native:platform"
	VNDK_TAG_VENDOR VndkTag = "native:vendor"
	VNDK_TAG_VNDK VndkTag = "native:vndk"
	VNDK_TAG_VNDKSP VndkTag = "native:vndksp"
)

var vndkAllowedDeps = map[VndkTag][]VndkTag{
	VNDK_TAG_NONE: {},
	VNDK_TAG_VENDOR: {VNDK_TAG_VENDOR, VNDK_TAG_VNDK, VNDK_TAG_VNDKSP},
	VNDK_TAG_VNDK: {VNDK_TAG_VNDK, VNDK_TAG_VNDKSP},
	VNDK_TAG_VNDKSP: {VNDK_TAG_VNDKSP},
}

type VndkProperties struct {
	VndkTag VndkTag `blueprint:"mutated"`
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

func (vndk *vndkdep) vndkTag() VndkTag {
	if vndk.Properties.VndkTag != "" {
		return vndk.Properties.VndkTag
	}
	return VNDK_TAG_VENDOR
}

func (vndk *vndkdep) verifyVndkDep(tag VndkTag) bool {
	for _, l := range vndkAllowedDeps[vndk.vndkTag()] {
		if tag == l {
			return true
		}
	}
	return false
}

func vndkDepsMutator(mctx android.TopDownMutatorContext) {
	if c, ok := mctx.Module().(*Module); ok && c.vndk() {
		mctx.VisitDirectDeps(func(m blueprint.Module) {
			if cc, ok := m.(*Module); ok {
				if cc.linker == nil {
					return
				}
				if lib, ok := cc.linker.(*libraryDecorator); ok && lib.shared() {
					//Check only shared_libraries.
					if !c.vndkdep.verifyVndkDep(cc.vndkTag()) {
						mctx.ModuleErrorf("(%s) should not link to %q(%s)",
							c.vndkTag(), cc.Name(), cc.vndkTag())
						return
					}
				}
			}
		})
	}
}
