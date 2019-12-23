package apex

import (
	"android/soong/android"
	"github.com/google/blueprint/proptools"
	"testing"
)

func TestVndkApexUsesVendorVariantNotApex(t *testing.T) {
	ctx, _ := testApex(t, `
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
	`+vndkLibrariesTxtFiles("current"))

	files := parseCopyCommands(t, ctx, "myapex")
	// libfoo is copied from vendor variant, instead of apex variant
	ensureContains(t, files["lib/libfoo.so"], "libfoo/android_vendor.VER_arm_armv7-a-neon_shared/libfoo.so")
}

func TestVndkApexUsesVendorVariantNotProduct(t *testing.T) {
	ctx, _ := testApex(t, `
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
	`+vndkLibrariesTxtFiles("current"), func(fs map[string][]byte, config android.Config) {
		config.TestProductVariables.ProductVndkVersion = proptools.StringPtr("current")
	})

	files := parseCopyCommands(t, ctx, "myapex")
	// libfoo is copied from vendor variant, instead of apex variant
	ensureContains(t, files["lib/libfoo.so"], "libfoo/android_vendor.VER_arm_armv7-a-neon_shared/libfoo.so")
}
