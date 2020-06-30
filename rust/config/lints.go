// Copyright 2020 The Android Open Source Project
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
	"strings"

	"android/soong/android"
)

var (
	// Default Rust lints that applies to Google-authored modules.
	defaultRustcLints = []string{
		"-A deprecated",
		"-D missing-docs",
		"-D warnings",
	}
	// Default Clippy lints. These are applied on top of defaultRustcLints.
	// It should be assumed that any warning lint will be promote to a deny.
	defaultClippyLints = []string{
		"-A clippy::type-complexity",
	}

	// Default Rust lints for vendor code.
	defaultRustcVendorLints = []string{
		"-A deprecated",
		"-D warnings",
	}
	// Clippy lints for vendor source. These are applied on top of defaultRustcVendorLints.
	// It should be assumed that any warning lint will be promote to a deny.
	defaultClippyVendorLints = []string{
		"-A clippy::complexity",
		"-A clippy::perf",
		"-A clippy::style",
	}

	// Rust lints for prebuilts. Caps the linting as there is no point returning warnings in our toolchain.
	defaultRustcPrebuiltsLints = []string{
		"--cap-lints allow",
	}
)

func init() {
	// Default Rust lints. These apply to all Google-authored modules.
	pctx.VariableFunc("RustDefaultLints", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("RUST_DEFAULT_LINTS"); override != "" {
			return override
		}
		return strings.Join(defaultRustcLints, " ")
	})
	pctx.VariableFunc("ClippyDefaultLints", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("CLIPPY_DEFAULT_LINTS"); override != "" {
			return override
		}
		return strings.Join(defaultClippyLints, " ")
	})

	// Rust lints that only applies to external code.
	pctx.VariableFunc("RustVendorLints", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("RUST_VENDOR_LINTS"); override != "" {
			return override
		}
		return strings.Join(defaultRustcVendorLints, " ")
	})
	pctx.VariableFunc("ClippyVendorLints", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("CLIPPY_VENDOR_LINTS"); override != "" {
			return override
		}
		return strings.Join(defaultClippyVendorLints, " ")
	})
	pctx.StaticVariable("RustPrebuiltsLints", strings.Join(defaultRustcPrebuiltsLints, " "))
}

type PathBasedClippyConfig struct {
	PathPrefix    string
	RustcConfig   string
	ClippyEnabled bool
	ClippyConfig  string
}

const noLint = ""
const rustcDefault = "${config.RustDefaultLints}"
const rustcVendor = "${config.RustVendorLints}"
const rustcPrebuilts = "${config.RustPrebuiltsLints}"
const clippyDefault = "${config.ClippyDefaultLints}"
const clippyVendor = "${config.ClippyVendorLints}"

// This is a map of local path prefixes to a set of parameters for the linting:
// - a string for the lints to apply to rustc.
// - a boolean to indicate if clippy should be executed.
// - a string for the lints to apply to clippy.
// The first entry matching will be used. If no entry is matching,
// (rustcDefault, true, clippyDefault) will be used.
var DefaultLocalTidyChecks = []PathBasedClippyConfig{
	{"external/", noLint, false, noLint},
	{"hardware/", rustcVendor, true, clippyVendor},
	{"prebuilts/", rustcPrebuilts, false, noLint},
	{"vendor/google", rustcDefault, true, clippyDefault},
	{"vendor/", rustcVendor, true, clippyVendor},
}

// ClippyLintsForDir returns the Clippy lints to be used for a repository.
func ClippyLintsForDir(dir string) (bool, string) {
	for _, pathCheck := range DefaultLocalTidyChecks {
		if strings.HasPrefix(dir, pathCheck.PathPrefix) {
			return pathCheck.ClippyEnabled, pathCheck.ClippyConfig
		}
	}
	return true, clippyDefault
}

// RustcLintsForDir returns the standard lints to be used for a repository.
func RustcLintsForDir(dir string) string {
	for _, pathCheck := range DefaultLocalTidyChecks {
		if strings.HasPrefix(dir, pathCheck.PathPrefix) {
			return pathCheck.RustcConfig
		}
	}
	return rustcDefault
}
