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
	"github.com/google/blueprint"

	"android/soong/android"
)

const (
	ltoCFlags  = "-flto"
	ltoLdFlags = "-flto"
	ltoArFlags = "--plugin ${config.ClangBin}/../lib64/LLVMgold.so"
)

type LTOProperties struct {
	Lto    *bool `android:"arch_variant"`
	LtoDep bool  `blueprint:"mutated"`
}

type lto struct {
	Properties LTOProperties
}

func (lto *lto) props() []interface{} {
	return []interface{}{&lto.Properties}
}

func (lto *lto) begin(ctx BaseModuleContext) {
}

func (lto *lto) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (lto *lto) flags(ctx BaseModuleContext, flags Flags) Flags {
	if Bool(lto.Properties.Lto) {
		flags.CFlags = append(flags.CFlags, ltoCFlags)
		flags.LdFlags = append(flags.LdFlags, ltoLdFlags)
		flags.ArFlags = append(flags.ArFlags, ltoArFlags)

		// Clang passes -m armelf_linux_eabi, which gold cannot handle
		if ctx.Arch().ArchType == android.Arm {
			flags.LdFlags = append(flags.LdFlags, "-Wl,-m,armelf")
		}
	}
	return flags
}

func (lto *lto) LTO() bool {
	if lto == nil {
		return false
	}

	return Bool(lto.Properties.Lto)
}

// Propagate lto requirements down from binaries
func ltoDepsMutator(mctx android.TopDownMutatorContext) {
	if c, ok := mctx.Module().(*Module); ok && c.lto.LTO() {
		mctx.VisitDepsDepthFirst(func(m blueprint.Module) {
			tag := mctx.OtherModuleDependencyTag(m)
			switch tag {
			case staticDepTag, staticExportDepTag, lateStaticDepTag, wholeStaticDepTag:
				cc, _ := m.(*Module)
				if cc == nil {
					return
				}
				cc.lto.Properties.LtoDep = true
			}
		})
	}
}

// Create lto variants for modules that need them
func ltoMutator(mctx android.BottomUpMutatorContext) {
	if c, ok := mctx.Module().(*Module); ok && c.lto != nil {
		if c.lto.LTO() {
			mctx.SetDependencyVariation("lto")
		} else if c.lto.Properties.LtoDep {
			modules := mctx.CreateVariations("", "lto")
			modules[0].(*Module).lto.Properties.Lto = boolPtr(false)
			modules[1].(*Module).lto.Properties.Lto = boolPtr(true)
			modules[0].(*Module).lto.Properties.LtoDep = false
			modules[1].(*Module).lto.Properties.LtoDep = false
			modules[1].(*Module).Properties.PreventInstall = true
			modules[1].(*Module).Properties.HideFromMake = true
		}
		c.lto.Properties.LtoDep = false
	}
}
