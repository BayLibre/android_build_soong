package apex

import (
	"android/soong/android"
	"github.com/google/blueprint/proptools"
	"testing"
)

func TestVndkApexUsesVendorVariant(t *testing.T) {
	bp := `
		apex_vndk {
			name: "myapex",
			key: "mykey",
		}
		apex_key {
			name: "mykey",
		}
		cc_library {
			name: "libfoo",
			vendor_available: true,
			vndk: {
				enabled: true,
			},
			system_shared_libs: [],
			stl: "none",
			notice: "custom_notice",
		}
		` + vndkLibrariesTxtFiles("current")

	t.Run("VNDK lib doesn't have apex variant", func(t *testing.T) {
		ctx, _ := testApex(t, bp)

		// libfoo doesn't have apex variants
		for _, variant := range ctx.ModuleVariantsForTests("libfoo") {
			ensureNotContains(t, variant, "_myapex")
		}

		// VNDK APEX doesn't create apex variant
		files := parseCopyCommands(t, ctx, "myapex", "android_common_image")
		ensureContains(t, files["lib/libfoo.so"], "libfoo/android_vendor.VER_arm_armv7-a-neon_shared/libfoo.so")
	})

	t.Run("VNDK APEX gathers only vendor variants even if product variant availabe", func(t *testing.T) {
		ctx, _ := testApex(t, bp, func(fs map[string][]byte, config android.Config) {
			// Now product variant is available
			config.TestProductVariables.ProductVndkVersion = proptools.StringPtr("current")
		})

		files := parseCopyCommands(t, ctx, "myapex", "android_common_image")
		ensureContains(t, files["lib/libfoo.so"], "libfoo/android_vendor.VER_arm_armv7-a-neon_shared/libfoo.so")
	})

	t.Run("VNDK APEX supports coverage variant", func(t *testing.T) {
		ctx, _ := testApex(t, bp+`
			cc_library {
				name: "libprofile-extras",
				vendor_available: true,
				native_coverage: false,
				system_shared_libs: [],
				stl: "none",
				notice: "custom_notice",
			}
		`, func(fs map[string][]byte, config android.Config) {
			config.TestProductVariables.NativeCoverage = proptools.BoolPtr(true)
		})

		files := parseCopyCommands(t, ctx, "myapex", "android_common_image")
		ensureContains(t, files["lib/libfoo.so"], "libfoo/android_vendor.VER_arm_armv7-a-neon_shared/libfoo.so")

		files = parseCopyCommands(t, ctx, "myapex", "android_common_cov_image")
		ensureContains(t, files["lib/libfoo.so"], "libfoo/android_vendor.VER_arm_armv7-a-neon_shared_cov/libfoo.so")
	})
}
