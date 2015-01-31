package main

import (
	"android/soong/androidmk/parser"
)

const (
	clear_vars                = "__android_mk_clear_vars"
	build_shared_library      = "android_shared_library"
	build_static_library      = "android_static_library"
	build_host_static_library = "android_host_static_library"
	build_executable          = "android_executable"
	build_host_executable     = "android_host_executable"
	build_prebuilt            = "android_prebuilt"
)

var stringProperties = map[string]string{
	"LOCAL_MODULE":       "name",
	"LOCAL_MODULE_STEM":  "stem",
	"LOCAL_MODULE_CLASS": "class",
	"LOCAL_MODULE_TAGS":  "tags",
}

var listProperties = map[string]string{
	"LOCAL_SRC_FILES":              "srcs",
	"LOCAL_SHARED_LIBRARIES":       "shared_libs",
	"LOCAL_STATIC_LIBRARIES":       "static_libs",
	"LOCAL_WHOLE_STATIC_LIBRARIES": "whole_static_libs",
	"LOCAL_C_INCLUDES":             "include_dirs",
	"LOCAL_EXPORT_C_INCLUDE_DIRS":  "export_include_dirs",
	"LOCAL_CFLAGS":                 "cflags",
}

var boolProperties = map[string]string{
	"LOCAL_IS_HOST_MODULE": "host",
}

func mydir(args []string) string {
	return "."
}

func androidScope() parser.Scope {
	globalScope := parser.NewScope(nil)
	globalScope.Set("CLEAR_VARS", clear_vars)
	globalScope.Set("BUILD_HOST_EXECUTABLE", build_host_executable)
	globalScope.Set("BUILD_SHARED_LIBRARY", build_shared_library)
	globalScope.Set("BUILD_STATIC_LIBRARY", build_static_library)
	globalScope.Set("BUILD_HOST_STATIC_LIBRARY", build_host_static_library)
	globalScope.Set("BUILD_EXECUTABLE", build_executable)
	globalScope.Set("BUILD_PREBUILT", build_prebuilt)
	globalScope.SetFunc("my-dir", mydir)

	return globalScope
}
