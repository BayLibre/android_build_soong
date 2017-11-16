package cc

import (
	"testing"
)

func TestGen(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		ctx := testCc(t, `
		cc_library_shared {
			name: "libfoo",
			srcs: [
				"foo.c",
				"b.aidl",
			],
		}`)

		aidl := ctx.ModuleForTests("libfoo", "android_arm_armv7-a-neon_core_shared").Rule("aidl")
		libfoo := ctx.ModuleForTests("libfoo", "android_arm_armv7-a-neon_core_shared").Module().(*Module)

		if !inList("-I"+aidl.Args["outDir"], libfoo.flags.GlobalFlags) {
			t.Errorf("missing aidl includes in global flags")
		}
	})

	t.Run("filegroup", func(t *testing.T) {
		ctx := testCc(t, `
		filegroup {
			name: "fg",
			srcs: ["b.aidl"],
		}

		cc_library_shared {
			name: "libfoo",
			srcs: [
				"foo.c",
				":fg",
			],
		}`)

		aidl := ctx.ModuleForTests("libfoo", "android_arm_armv7-a-neon_core_shared").Rule("aidl")
		libfoo := ctx.ModuleForTests("libfoo", "android_arm_armv7-a-neon_core_shared").Module().(*Module)

		if !inList("-I"+aidl.Args["outDir"], libfoo.flags.GlobalFlags) {
			t.Errorf("missing aidl includes in global flags")
		}
	})

}
