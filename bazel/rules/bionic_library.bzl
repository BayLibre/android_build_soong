"""A macro to handle building and postprocessing of bionic shared libraries."""

load(":experimental_cc_shared_library.bzl", "CcSharedLibraryInfo", "cc_shared_library")

def _gen_sort_symbols_impl(ctx):
    gcc_toolchain = ctx.toolchains["//:gcc_toolchain_type"].gccinfo

    ctx.actions.run(
        env = {"CROSS_COMPILE": gcc_toolchain.path_prefix},
        inputs = ctx.files.src,
        tools = [gcc_toolchain.nm, ctx.executable._gen_symbols_script],
        outputs = [ctx.outputs.out],
        executable = ctx.executable._gen_symbols_script,
        arguments = [
            ctx.files.src[0].path,
            ctx.outputs.out.path,
        ],
    )

_gen_sort_symbols = rule(
    implementation = _gen_sort_symbols_impl,
    attrs = {
        "src": attr.label(mandatory = True, allow_single_file = True),
        "out": attr.output(mandatory = True),
        "_gen_symbols_script": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
            default = "//build/soong/scripts:gen_sorted_bss_symbols.sh",
        ),
    },
    toolchains = ["//:gcc_toolchain_type"],
)

def _stripped_shared_library_impl(ctx):
    gcc_toolchain = ctx.toolchains["//:gcc_toolchain_type"].gccinfo

    d_file = ctx.actions.declare_file(ctx.attr.name + ".d")

    ctx.actions.run(
        env = {
            "CROSS_COMPILE": gcc_toolchain.path_prefix,
            "XZ": ctx.executable._xz.path,
            "CLANG_BIN": ctx.executable._llvm.dirname,
        },
        inputs = ctx.files.src,
        tools = [
            gcc_toolchain.readelf,
            gcc_toolchain.objcopy,
            ctx.executable._xz,
            ctx.executable._strip_script,
            ctx.executable._llvm,
        ],
        outputs = [ctx.outputs.out, d_file],
        executable = ctx.executable._strip_script,
        arguments = ctx.attr.strip_script_flags + [
            "-i",
            ctx.files.src[0].path,
            "-o",
            ctx.outputs.out.path,
            "-d",
            d_file.path,
        ],
    )
    return [DefaultInfo(files = depset([ctx.outputs.out])), ctx.attr.src[CcSharedLibraryInfo]]

_stripped_shared_library = rule(
    implementation = _stripped_shared_library_impl,
    attrs = {
        "src": attr.label(mandatory = True, providers = [CcSharedLibraryInfo]),
        "out": attr.output(mandatory = True),
        "strip_script_flags": attr.string_list(),
        "_xz": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
            default = "//:prebuilts/build-tools/linux-x86/bin/xz",
        ),
        "_strip_script": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
            default = "//build/soong/scripts:strip.sh",
        ),
        "_llvm": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
            default = "//prebuilts/clang/host/linux-x86:llvm-ar",
        ),
    },
    toolchains = ["//:gcc_toolchain_type"],
)

def bionic_library(name, strip_script_flags, sort_symbols = False, **kwargs):
    visibility = kwargs.pop("visibility", [])

    unstripped_name = name + "_unstripped"

    if sort_symbols:
        unsorted_name = name + "_unsorted"
        symbol_order_filename = name + ".so.symbol_order"

        cc_shared_library(
            name = unsorted_name,
            features = ["disable_rpath"],
            visibility = ["//visibility:private"],
            tags = ["manual"],
            **kwargs
        )

        _gen_sort_symbols(
            name = name + "_symbol_order",
            src = unsorted_name,
            out = symbol_order_filename,
            visibility = ["//visibility:private"],
            tags = ["manual"],
        )

        cc_shared_library(
            name = unstripped_name,
            features = ["disable_rpath"],
            symbol_order = symbol_order_filename,
            visibility = ["//visibility:private"],
            tags = ["manual"],
            **kwargs
        )
    else:
        cc_shared_library(
            name = unstripped_name,
            visibility = ["//visibility:private"],
            tags = ["manual"],
            **kwargs
        )

    _stripped_shared_library(
        name = name,
        src = unstripped_name,
        out = name + ".so",
        strip_script_flags = strip_script_flags,
        visibility = visibility,
    )
