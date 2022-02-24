# Enable Remote Build Execution

Soong is integrated with Google's Remote Build Execution(RBE) service, which
implements the
[Remote Executaion API](https://github.com/bazelbuild/remote-apis).

With RBE enabled, it can speed up the Android Platform builds by distribute
build actions through a worker pool sharing a central cache of build results.

## Configuration

To enable RBE, you need to set several environment variables before triggering
the build. You can set them through a
[environment variables config file](https://android.googlesource.com/platform/build/soong/+/master/README.md#environment-variables-config-file). [rbe.json](rbe.json) is an example config:

```json
{
    "env": {
        "USE_RBE": "1",

        "RBE_R8_EXEC_STRATEGY": "remote_local_fallback",
        "RBE_CXX_EXEC_STRATEGY": "remote_local_fallback",
        "RBE_D8_EXEC_STRATEGY": "remote_local_fallback",
        "RBE_JAVAC_EXEC_STRATEGY": "remote_local_fallback",
        "RBE_JAVAC": "1",
        "RBE_R8": "1",
        "RBE_D8": "1",

        "RBE_instance": "[replace with your RBE instance]",
        "RBE_service": "[replace with your RBE service endpoint]",

        "RBE_DIR": "prebuilts/remoteexecution-client/live",

        "RBE_use_application_default_credentials": "true",

        "RBE_log_dir": "/tmp",
        "RBE_output_dir": "/tmp",
        "RBE_proxy_log_dir": "/tmp"
    }
}
```
