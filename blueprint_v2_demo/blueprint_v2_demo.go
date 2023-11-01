package blueprint_v2_demo

import "android/soong/android"

func init() {
	android.RegisterBlueprintV2ModuleType("//build/soong/blueprint_v2_demo/blueprint_v2_demo.bpi", "blueprint_v2_demo")
}
