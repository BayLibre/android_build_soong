load("//build/soong/bazel/rules:experimental_cc_shared_library.bzl", "cc_shared_library")

# Pulled from build/soong/cc/ndk_library.go
STUB_COPTS = [
    # We're knowingly doing some otherwise unsightly things with builtin
    # functions here. We're just generating stub libraries, so ignore it.
    "-Wno-incompatible-library-redeclaration",
    "-Wno-incomplete-setjmp-declaration",
    "-Wno-builtin-requires-header",
    "-Wno-invalid-noreturn",
    "-Wall",
    "-Werror",
    # These libraries aren't actually used. Don't worry about unwinding
    # (avoids the need to link an unwinder into a fake library).
    "-fno-unwind-tables",
]

def stub_cc_shared_library(name, arch, version_script_template, copts = [], **kwargs):
    stub_src = name + "_stub.c"
    stub_version_script = name + "_stub.map"
    api_levels_json_file = "//build/soong/bazel/rules:api_levels.json"

    # Generate stub.c and stub map
    native.genrule(
        name = name + "_stub_src",
        exec_tools = ["//build/soong:gen_stub_libs"],
        srcs = [api_levels_json_file, version_script_template],
        outs = [stub_src, stub_version_script],
        cmd = "$(location //build/soong:gen_stub_libs) " +
              "--arch " + arch + " " +
              "--api 10000 " +
              "--api-map $(location " + api_levels_json_file + ") " +
              "--apex $(location " + version_script_template + ") " +
              "$(location " + stub_src + ") " +
              "$(location " + stub_version_script + ")",
        visibility = ["//visibility:private"],
    )

    # Generate stub.o
    native.cc_library(
        name = name + "_cclib",
        srcs = [name + "_stub_src"],
        copts = STUB_COPTS + copts,
        visibility = ["//visibility:private"],
    )

    # Generate stub .so
    cc_shared_library(
        name = name,
        exports = [name + "_cclib"],
        visibility_file = stub_version_script,
        nocrt = True,
        **kwargs
    )
    # TODO(cparsons): Strip stub .so. This doesn't seem to practically matter for
    # libc -> libdl, but it may matter for other shared library stubs.
