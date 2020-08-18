// Copyright 2019 The Android Open Source Project
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

package rust

import (
	"strings"
	"testing"

	"android/soong/android"
)

// Test that the prefer_dynamic property is handled correctly.
func TestPreferDynamicBinary(t *testing.T) {
	ctx := testRust(t, `
		rust_binary {
			name: "fizz-buzz-dynamic",
			srcs: ["foo.rs"],
			prefer_dynamic: true,
			rustlibs: ["libfoo"],
			host_supported: true,
		}

		rust_binary {
			name: "fizz-buzz",
			srcs: ["foo.rs"],
			rustlibs: ["libfoo"],
			host_supported: true,
		}

		rust_binary {
			name: "fizz-buzz-static",
			srcs: ["foo.rs"],
			rustlibs: ["libfoo"],
			prefer_dynamic: false,
			host_supported: true,
		}

		rust_library {
			name: "libfoo",
			srcs: ["foo.rs"],
			crate_name: "foo",
			host_supported: true,
		}`)

	fizzBuzzHost := ctx.ModuleForTests("fizz-buzz", "linux_glibc_x86_64").Module().(*Module)
	fizzBuzzDynamicHost := ctx.ModuleForTests("fizz-buzz-dynamic", "linux_glibc_x86_64").Module().(*Module)
	fizzBuzzStaticHost := ctx.ModuleForTests("fizz-buzz-static", "linux_glibc_x86_64").Module().(*Module)

	fizzBuzzDevice := ctx.ModuleForTests("fizz-buzz", "android_arm64_armv8-a").Module().(*Module)
	fizzBuzzDynamicDevice := ctx.ModuleForTests("fizz-buzz-dynamic", "android_arm64_armv8-a").Module().(*Module)
	fizzBuzzStaticDevice := ctx.ModuleForTests("fizz-buzz-static", "android_arm64_armv8-a").Module().(*Module)

	if !android.InList("libfoo", fizzBuzzHost.Properties.AndroidMkRlibs) {
		t.Errorf("libfoo should be an rlib dep when prefer_dynamic is undefined for host modules")
	}
	if !android.InList("libfoo", fizzBuzzDynamicHost.Properties.AndroidMkDylibs) {
		t.Errorf("libfoo should be an dylib dep when prefer_dynamic is true for host modules")
	}
	if !android.InList("libfoo", fizzBuzzStaticHost.Properties.AndroidMkRlibs) {
		t.Errorf("libfoo should be an rlib dep when prefer_dynamic is false for host modules")
	}

	if !android.InList("libfoo", fizzBuzzDevice.Properties.AndroidMkDylibs) {
		t.Errorf("libfoo should be an dylib dep when prefer_dynamic is undefined for device modules")
	}
	if !android.InList("libfoo", fizzBuzzDynamicDevice.Properties.AndroidMkDylibs) {
		t.Errorf("libfoo should be an dylib dep when prefer_dynamic is true for device modules")
	}
	if !android.InList("libfoo", fizzBuzzStaticDevice.Properties.AndroidMkRlibs) {
		t.Errorf("libfoo should be an rlib dep when prefer_dynamic is false for device modules")
	}
}

// Test that the path returned by HostToolPath is correct
func TestHostToolPath(t *testing.T) {
	ctx := testRust(t, `
		rust_binary_host {
			name: "fizz-buzz",
			srcs: ["foo.rs"],
		}`)

	path := ctx.ModuleForTests("fizz-buzz", "linux_glibc_x86_64").Module().(*Module).HostToolPath()
	if g, w := path.String(), "/host/linux-x86/bin/fizz-buzz"; !strings.Contains(g, w) {
		t.Errorf("wrong host tool path, expected %q got %q", w, g)
	}
}

// Test that the flags being passed to rust_binary modules are as expected
func TestBinaryFlags(t *testing.T) {
	ctx := testRust(t, `
		rust_binary_host {
			name: "fizz-buzz",
			srcs: ["foo.rs"],
			prefer_dynamic: true,
		}`)

	fizzBuzz := ctx.ModuleForTests("fizz-buzz", "linux_glibc_x86_64").Output("fizz-buzz")

	flags := fizzBuzz.Args["rustcFlags"]
	if strings.Contains(flags, "--test") {
		t.Errorf("extra --test flag, rustcFlags: %#v", flags)
	}
	if !strings.Contains(flags, "prefer-dynamic") {
		t.Errorf("missing prefer-dynamic flag, rustcFlags: %#v", flags)
	}
}
