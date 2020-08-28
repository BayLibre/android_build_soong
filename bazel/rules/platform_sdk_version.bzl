# A quick-and-direct definition of the platform sdk version, which can be used in other rules as needed.

PlatformSdkVersion = provider(fields = ["version"])

def _platform_sdk_version_impl(ctx):
    version = ctx.attr.version
    provider = PlatformSdkVersion(version = version)

    # Also make this available as a Make variable.
    vars = platform_common.TemplateVariableInfo({
        "PLATFORM_SDK_VERSION": str(version),
    })
    return [provider, vars]

platform_sdk_version = rule(
    implementation = _platform_sdk_version_impl,
    attrs = {
        "version": attr.int(mandatory = True),
    },
)
