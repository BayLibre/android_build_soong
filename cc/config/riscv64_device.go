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

package config

import (
	"fmt"
	"strings"

	"android/soong/android"
)

var (
	riscv64Cflags = []string{
		// Help catch common 32/64-bit errors.
		// Common to all LP64 architectures.
		"-Werror=implicit-function-declaration",

		// For stack allocations larger than a page, touch each page immediately
		// to ensure we hit the guard page on stack overflow.
		// Common to all LP64 architectures.
		"-fstack-clash-protection",

		// This is already the driver's Android default, but duplicated here (and
		// below) for ease of experimentation with additional extensions.
		"-march=rv64gcv_zba_zbb_zbs_zvbb",
		// TODO: remove when qemu V works (https://gitlab.com/qemu-project/qemu/-/issues/1976)
		// (Note that we'll probably want to wait for berberis to be good enough
		// that most people don't care about qemu's V performance either!)
		"-mno-implicit-float",
	}

	riscv64ArchVariantCflags = map[string][]string{
		// SpaceMit X60 (BananaPi F3 / SpaceMit K1).  Adds every ISA
		// extension the X60 advertises in /proc/cpuinfo on top of the
		// rv64gcv_zba_zbb_zbs AOSP baseline.  Binaries built with this
		// variant will not run on RISC-V cores lacking these extensions.
		"x60": {
			"-march=rv64gcv_zba_zbb_zbs_zicond_zfh_zvfh_zicboz_zicbop_zbc_zkt",
			"-mcpu=spacemit-x60",
			"-mtune=spacemit-x60",
		},
		// SpaceMit X100 (SpaceMit K3).  clang does not know
		// -mcpu=spacemit-x100 yet, so name the extensions the core
		// advertises in its device tree instead.  The X100 is
		// out-of-order, so tune for that rather than leaving the
		// in-order default.
		"x100": {
			"-march=rv64imafdc_b_v_za64rs_zawrs_zba_zbb_zbc_zbs_zca_zcb_zcd_zcmop_zfa_zfbfmin_zfh_zfhmin_zicbom_zicbop_zicboz_ziccamoa_ziccif_zicclsm_zicntr_zicond_zicsr_zifencei_zihintntl_zihintpause_zihpm_zimop_zkt_zvbb_zvbc_zvfbfmin_zvfbfwma_zvfh_zvfhmin_zvkb_zvkg_zvkn_zvknc_zvkned_zvkng_zvknha_zvknhb_zvks_zvksc_zvksed_zvksg_zvksh_zvkt",
			"-mtune=generic-ooo",
			// Overrides the -mno-implicit-float in the riscv64
			// baseline, which is there only to keep RVV out of qemu.
			// Arch variant cflags come after the baseline ones.
			"-mimplicit-float",
		},
		// Alibaba/T-Head ZhiHe A210.  Extension list taken from /proc/cpuinfo
		"a210": {
			"-march=rv64imafdcv_zicntr_zicsr_zifencei_zihpm_zaamo_zalrsc_zca_zcd_zba_zbb_zbc_zbs_zve32f_zve32x_zve64d_zve64f_zve64x_sscofpmf_svpbmt",
			"-mimplicit-float",
		},
	}

	riscv64Ldflags = []string{
		// This is already the driver's Android default, but duplicated here (and
		// above) for ease of experimentation with additional extensions.
		"-march=rv64gcv_zba_zbb_zbs_zvbb",
		"-Wl,-z,max-page-size=4096",
	}

	riscv64ArchVariantLdflags = map[string][]string{
		"x60": {
			"-march=rv64gcv_zba_zbb_zbs_zicond_zfh_zvfh_zicboz_zicbop_zbc_zkt",
		},
		"x100": {
			"-march=rv64imafdc_b_v_za64rs_zawrs_zba_zbb_zbc_zbs_zca_zcb_zcd_zcmop_zfa_zfbfmin_zfh_zfhmin_zicbom_zicbop_zicboz_ziccamoa_ziccif_zicclsm_zicntr_zicond_zicsr_zifencei_zihintntl_zihintpause_zihpm_zimop_zkt_zvbb_zvbc_zvfbfmin_zvfbfwma_zvfh_zvfhmin_zvkb_zvkg_zvkn_zvknc_zvkned_zvkng_zvknha_zvknhb_zvks_zvksc_zvksed_zvksg_zvksh_zvkt",
		},
		"a210": {
			"-march=rv64imafdcv_zicntr_zicsr_zifencei_zihpm_zaamo_zalrsc_zca_zcd_zba_zbb_zbc_zbs_zve32f_zve32x_zve64d_zve64f_zve64x_sscofpmf_svpbmt",
		},
	}

	riscv64Cppflags = []string{}

	riscv64CpuVariantCflags = map[string][]string{}
)

const ()

func init() {

	pctx.StaticVariable("Riscv64Ldflags", strings.Join(riscv64Ldflags, " "))

	pctx.StaticVariable("Riscv64Cflags", strings.Join(riscv64Cflags, " "))
	pctx.StaticVariable("Riscv64Cppflags", strings.Join(riscv64Cppflags, " "))

	for variant, flags := range riscv64ArchVariantCflags {
		pctx.StaticVariable("Riscv64"+variant+"VariantCflags",
			strings.Join(flags, " "))
		riscv64ArchVariantCflagsVar[variant] =
			"${config.Riscv64" + variant + "VariantCflags}"
	}

	for variant, flags := range riscv64ArchVariantLdflags {
		pctx.StaticVariable("Riscv64"+variant+"VariantLdflags",
			strings.Join(flags, " "))
		riscv64ArchVariantLdflagsVar[variant] =
			"${config.Riscv64" + variant + "VariantLdflags}"
	}
}

var (
	riscv64ArchVariantCflagsVar = map[string]string{}

	riscv64ArchVariantLdflagsVar = map[string]string{}

	riscv64CpuVariantCflagsVar = map[string]string{}

	riscv64CpuVariantLdflags = map[string]string{}
)

type toolchainRiscv64 struct {
	toolchainBionic
	toolchain64Bit

	ldflags         string
	toolchainCflags string
}

func (t *toolchainRiscv64) Name() string {
	return "riscv64"
}

func (t *toolchainRiscv64) IncludeFlags() string {
	return ""
}

func (t *toolchainRiscv64) ClangTriple() string {
	return "riscv64-linux-android"
}

func (t *toolchainRiscv64) Cflags() string {
	return "${config.Riscv64Cflags}"
}

func (t *toolchainRiscv64) Cppflags() string {
	return "${config.Riscv64Cppflags}"
}

func (t *toolchainRiscv64) Ldflags(ctx ToolchainFlagsContext) FlagsWithDeps {
	return FlagsWithDeps{
		Flags: t.ldflags,
	}
}

func (t *toolchainRiscv64) ToolchainCflags() string {
	return t.toolchainCflags
}

func (toolchainRiscv64) LibclangRuntimeLibraryArch() string {
	return "riscv64"
}

func riscv64ToolchainFactory(arch android.Arch) Toolchain {
	switch arch.ArchVariant {
	case "", "x60", "x100", "a210":
	default:
		panic(fmt.Sprintf("Unknown Riscv64 architecture version: %q", arch.ArchVariant))
	}

	toolchainCflags := []string{riscv64ArchVariantCflagsVar[arch.ArchVariant]}
	toolchainCflags = append(toolchainCflags,
		variantOrDefault(riscv64CpuVariantCflagsVar, arch.CpuVariant))

	extraLdflags := variantOrDefault(riscv64CpuVariantLdflags, arch.CpuVariant)
	archLdflags := riscv64ArchVariantLdflagsVar[arch.ArchVariant]
	return &toolchainRiscv64{
		ldflags: strings.Join([]string{
			"${config.Riscv64Ldflags}",
			archLdflags,
			extraLdflags,
		}, " "),
		toolchainCflags: strings.Join(toolchainCflags, " "),
	}
}

func init() {
	registerToolchainFactory(android.Android, android.Riscv64, riscv64ToolchainFactory)
}
