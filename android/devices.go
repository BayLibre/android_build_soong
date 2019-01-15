// Copyright 2019 Google Inc. All rights reserved.
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

func init() {
	// Arm
	RegisterArchFeatures(Arm,
		"neon")

	RegisterArchVariants(Arm,
		"armv7-a",
		"armv7-a-neon",
		"armv8-a",
		"armv8-2a",
		"cortex-a7",
		"cortex-a8",
		"cortex-a9",
		"cortex-a15",
		"cortex-a53",
		"cortex-a53-a57",
		"cortex-a55",
		"cortex-a72",
		"cortex-a73",
		"cortex-a75",
		"cortex-a76",
		"krait",
		"kryo",
		"kryo385",
		"exynos-m1",
		"exynos-m2")

	RegisterArchVariantFeatures(Arm, "armv7-a-neon", "neon")
	RegisterArchVariantFeatures(Arm, "armv8-a", "neon")
	RegisterArchVariantFeatures(Arm, "armv8-2a", "neon")

	// Arm64
	RegisterArchVariants(Arm64,
		"armv8_a",
		"armv8_2a",
		"cortex-a53",
		"cortex-a55",
		"cortex-a72",
		"cortex-a73",
		"cortex-a75",
		"cortex-a76",
		"kryo",
		"kryo385",
		"exynos-m1",
		"exynos-m2")

	// Mips
	RegisterArchVariants(Mips,
		"mips32_fp",
		"mips32r2_fp",
		"mips32r2_fp_xburst",
		"mips32r2dsp_fp",
		"mips32r2dspr2_fp",
		"mips32r6")
	RegisterArchFeatures(Mips,
		"dspr2",
		"rev6",
		"msa")
	RegisterArchVariantFeatures(Mips, "mips32r2dspr2_fp",
		"dspr2")
	RegisterArchVariantFeatures(Mips, "mips32r6",
		"rev6")

	// Mips64
	RegisterArchVariants(Mips64,
		"mips64r2",
		"mips64r6")
	RegisterArchFeatures(Mips64,
		"rev6",
		"msa")
	RegisterArchVariantFeatures(Mips64, "mips64r6",
		"rev6")

	// X86
	RegisterArchVariants(X86,
		"atom",
		"haswell",
		"ivybridge",
		"sandybridge",
		"silvermont",
		"x86_64")
	RegisterArchFeatures(X86,
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt",
		"movbe")
	RegisterArchVariantFeatures(X86, "x86_64",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"popcnt")
	RegisterArchVariantFeatures(X86, "atom",
		"ssse3",
		"movbe")
	RegisterArchVariantFeatures(X86, "haswell",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt",
		"movbe")
	RegisterArchVariantFeatures(X86, "ivybridge",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	RegisterArchVariantFeatures(X86, "sandybridge",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"popcnt")
	RegisterArchVariantFeatures(X86, "silvermont",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"popcnt",
		"movbe")

	// X86_64
	RegisterArchVariants(X86_64,
		"haswell",
		"ivybridge",
		"sandybridge",
		"silvermont")
	RegisterArchFeatures(X86_64,
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	RegisterArchVariantFeatures(X86_64, "",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"popcnt")
	RegisterArchVariantFeatures(X86_64, "haswell",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	RegisterArchVariantFeatures(X86_64, "ivybridge",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	RegisterArchVariantFeatures(X86_64, "sandybridge",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"popcnt")
	RegisterArchVariantFeatures(X86_64, "silvermont",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"popcnt")
}
