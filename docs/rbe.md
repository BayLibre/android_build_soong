# Build Android Platform on Remote Build Execution

Soong is integrated with Google's Remote Build Execution(RBE) service, which
implements the
[Remote Executaion API](https://github.com/bazelbuild/remote-apis).

With RBE enabled, it can speed up the Android Platform builds by distributing
build actions through a worker pool sharing a central cache of build results.

## Configuration

To enable RBE, you need to set several environment variables before triggering
the build. You can set them through a
[environment variables config file](https://android.googlesource.com/platform/build/soong/+/master/README.md#environment-variables-config-file).
As an example, [build/soong/docs/rbe.json](rbe.json) is a config that enables
RBE in the build. Once the config file is created, you need to let Soong load
the config file by specifying `ANDROID_BUILD_ENVIRONMENT_CONFIG_DIR` environment
variable and `ANDROID_BUILD_ENVIRONMENT_CONFIG` environment variable. The
following command starts Soong with [build/soong/docs/rbe.json](rbe.json)
loaded:

```shell
ANDROID_BUILD_ENVIRONMENT_CONFIG=rbe \
ANDROID_BUILD_ENVIRONMENT_CONFIG_DIR=build/soong/doc \
  build/soong/soong_ui.bash
```
