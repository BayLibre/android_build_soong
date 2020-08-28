"""Build rule to match the functionality of the toolchain_library Soong module."""

def _toolchain_library_impl(ctx):
    ctx.actions.run_shell(
        env = {"CLANG_BIN": ctx.executable._llvm.dirname},
        inputs = ctx.files.src,
        tools = [ctx.executable._repack_script, ctx.executable._llvm],
        outputs = [ctx.outputs.out],
        command = (
            ctx.executable._repack_script.path + " -i " + ctx.files.src[0].path +
            " -o " + ctx.outputs.out.path + " " +
            " ".join(ctx.attr.repack_objects_to_keep)
        ),
    )
    return [DefaultInfo(files = depset([ctx.outputs.out]))]

toolchain_library = rule(
    implementation = _toolchain_library_impl,
    attrs = {
        "src": attr.label(mandatory = True, allow_single_file = True),
        "repack_objects_to_keep": attr.string_list(allow_empty = False),
        "out": attr.output(mandatory = True),
        "_repack_script": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
            default = "//build/soong/scripts:archive_repack.sh",
        ),
        "_llvm": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
            default = "//prebuilts/clang/host/linux-x86:llvm-ar",
        ),
    },
)
