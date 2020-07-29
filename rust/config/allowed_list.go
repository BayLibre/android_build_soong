package config

var (
	RustAllowedPaths = []string{
		"external/adhd",
		"external/crosvm",
		"external/minijail",
		"external/rust",
		"external/sqlite/rust",
		"prebuilts/rust",
	}

	RustModuleTypes = []string{
		"rust_binary",
		"rust_binary_host",
		"rust_library",
		"rust_library_dylib",
		"rust_library_rlib",
		"rust_ffi",
		"rust_ffi_shared",
		"rust_ffi_static",
		"rust_library_host",
		"rust_library_host_dylib",
		"rust_library_host_rlib",
		"rust_ffi_host",
		"rust_ffi_host_shared",
		"rust_ffi_host_static",
		"rust_proc_macro",
		"rust_test",
		"rust_test_host",
	}
)
