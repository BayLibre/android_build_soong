package main

var stringProperties = map[string]string{
	"name":             "LOCAL_MODULE",
	"stem":             "LOCAL_MODULE_STEM",
	"class":            "LOCAL_MODULE_CLASS",
	"stl":              "LOCAL_CXX_STL",
	"strip":            "LOCAL_STRIP_MODULE",
	"compile_multilib": "LOCAL_MULTILIB",
	"instruction_set":  "LOCAL_ARM_MODE_HACK",
	"sdk_version":      "LOCAL_SDK_VERSION",
	//"stl":              "LOCAL_NDK_STL_VARIANT", TODO
	"manifest":     "LOCAL_JAR_MANIFEST",
	"jarjar_rules": "LOCAL_JARJAR_RULES",
	"certificate":  "LOCAL_CERTIFICATE",
	//"name":             "LOCAL_PACKAGE_NAME", TODO
}

var listProperties = map[string]string{
	"srcs":                "LOCAL_SRC_FILES",
	"shared_libs":         "LOCAL_SHARED_LIBRARIES",
	"static_libs":         "LOCAL_STATIC_LIBRARIES",
	"whole_static_libs":   "LOCAL_WHOLE_STATIC_LIBRARIES",
	"system_shared_libs":  "LOCAL_SYSTEM_SHARED_LIBRARIES",
	"include_dirs":        "LOCAL_C_INCLUDES",
	"export_include_dirs": "LOCAL_EXPORT_C_INCLUDE_DIRS",
	"asflags":             "LOCAL_ASFLAGS",
	"clang_asflags":       "LOCAL_CLANG_ASFLAGS",
	"cflags":              "LOCAL_CFLAGS",
	"conlyflags":          "LOCAL_CONLYFLAGS",
	"cppflags":            "LOCAL_CPPFLAGS",
	"ldflags":             "LOCAL_LDFLAGS",
	"required":            "LOCAL_REQUIRED_MODULES",
	"tags":                "LOCAL_MODULE_TAGS",
	"host_ldlibs":         "LOCAL_LDLIBS",
	"clang_cflags":        "LOCAL_CLANG_CFLAGS",
	"yaccflags":           "LOCAL_YACCFLAGS",

	"java_resource_dirs": "LOCAL_JAVA_RESOURCE_DIRS",
	"javacflags":         "LOCAL_JAVACFLAGS",
	"dxflags":            "LOCAL_DX_FLAGS",
	"java_libs":          "LOCAL_JAVA_LIBRARIES",
	"java_static_libs":   "LOCAL_STATIC_JAVA_LIBRARIES",
	"aidl_includes":      "LOCAL_AIDL_INCLUDES",
	"aaptflags":          "LOCAL_AAPT_FLAGS",
	"package_splits":     "LOCAL_PACKAGE_SPLITS",
}

var boolProperties = map[string]string{
	"host":                    "LOCAL_IS_HOST_MODULE",
	"clang":                   "LOCAL_CLANG",
	"static":                  "LOCAL_FORCE_STATIC_EXECUTABLE",
	"asan":                    "LOCAL_ADDRESS_SANITIZER",
	"native_coverage":         "LOCAL_NATIVE_COVERAGE",
	"nocrt":                   "LOCAL_NO_CRT",
	"allow_undefined_symbols": "LOCAL_ALLOW_UNDEFINED_SYMBOLS",
	"rtti": "LOCAL_RTTI_FLAG",

	"no_standard_libraries": "LOCAL_NO_STANDARD_LIBRARIES",

	"export_package_resources": "LOCAL_EXPORT_PACKAGE_RESOURCES",
}

var moduleTypes = map[string]string{
	"cc_library_shared":        "BUILD_SHARED_LIBRARY",
	"cc_library_static":        "BUILD_STATIC_LIBRARY",
	"cc_library_host_shared":   "BUILD_HOST_SHARED_LIBRARY",
	"cc_library_host_static":   "BUILD_HOST_STATIC_LIBRARY",
	"cc_binary":                "BUILD_EXECUTABLE",
	"cc_binary_host":           "BUILD_HOST_EXECUTABLE",
	"cc_test":                  "BUILD_NATIVE_TEST",
	"cc_test_host":             "BUILD_HOST_NATIVE_TEST",
	"java_library":             "BUILD_JAVA_LIBRARY",
	"java_library_static":      "BUILD_STATIC_JAVA_LIBRARY",
	"java_library_host":        "BUILD_HOST_JAVA_LIBRARY",
	"java_library_host_dalvik": "BUILD_HOST_DALVIK_JAVA_LIBRARY",
	"android_app":              "BUILD_PACKAGE",
	"prebuilt":                 "BUILD_PREBUILT",
}
