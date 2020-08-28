"""Toolchain rule for gcc tools typically used for binary postprocessing."""

GccToolchainInfo = provider(fields = {
    "nm": "'nm' tool file",
    "objcopy": "'objcopy' tool file",
    "path_prefix": "the string path prefix of all tools that this toolchain is responsible for",
    "readelf": "'readelf' tool file",
})

def _tool_path_prefix(full_path, tool_name):
    if not full_path.endswith(tool_name):
        fail("Expected " + tool_name + " tool filename to end in '" + tool_name + "'")
    return full_path[0:-len(tool_name)]

def _gcc_toolchain_impl(ctx):
    path_prefix = _tool_path_prefix(ctx.executable.nm.path, "nm")
    objcopy_prefix = _tool_path_prefix(ctx.executable.objcopy.path, "objcopy")
    if not objcopy_prefix == path_prefix:
        fail("gcc tool prefix mismatch: [" + path_prefix + ", " + objcopy_prefix + "]")
    readelf_prefix = _tool_path_prefix(ctx.executable.readelf.path, "readelf")
    if not readelf_prefix == path_prefix:
        fail("gcc tool prefix mismatch: [" + path_prefix + ", " + readelf_prefix + "]")

    toolchain_info = platform_common.ToolchainInfo(
        gccinfo = GccToolchainInfo(
            nm = ctx.executable.nm,
            objcopy = ctx.executable.objcopy,
            path_prefix = path_prefix,
            readelf = ctx.executable.readelf,
        ),
    )
    return [toolchain_info]

gcc_toolchain = rule(
    implementation = _gcc_toolchain_impl,
    attrs = {
        "nm": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
        ),
        "objcopy": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
        ),
        "readelf": attr.label(
            cfg = "host",
            executable = True,
            allow_single_file = True,
        ),
    },
)
