# Clang-Tidy Rules and Checks

Android C/C++ source files can be checked by clang-tidy for issues like
coding style, error-prone/performance patterns, and static flow analysis.
See the official
[clang-tidy document](https://clang.llvm.org/extra/clang-tidy)
and list of
[clang-tidy checks](https://clang.llvm.org/extra/clang-tidy/checks/list.html).

To use clang-tidy, an Android project has three options:
1. Use the Android build system's **global defaults** without
   changing any existing Android.bp files.
2. If the global defaults do not work, a project can adds `.clang-tidy`
   config files to specify its own checks
   and check options for all source files under a directory,
   see the [clang-tidy document](https://clang.llvm.org/extra/clang-tidy).
3. If some modules or files in a project should never be
   checked by clang-tidy, or should always be checked,
   or should use some different checks or flags,
   some `tidy_*` properties can be set in the local Android.bp file.

## Global defaults for Android builds

The simplest way to enable clang-tidy checks is
to set environment variable `WITH_TIDY`.
```
$ WITH_TIDY=1 make
```

This will turn on the global default to run clang-tidy for every required
C/C++ source file compilation. The global default clang-tidy checks
do not include time-consuming static analyzer checks. To enable those
checks, set the `CLANG_ANALYZER_CHECKS` variable.
```
$ WITH_TIDY=1 CLANG_ANALYZER_CHECKS=1 make
```

The default global clang-tidy checks and flags are defined in
[build/soong/cc/config/tidy.go](https://android.googlesource.com/platform/build/soong/+/refs/heads/master/cc/config/tidy.go).


## Phony tidy-* targets

### The tidy-*directory* targets

Setting `WITH_TIDY=1` is easy to enable clang-tidy globally for any build.
However, it adds extra compilation time.

For developers focusing on just one directory, they only want to compile
their files with clang-tidy and wish to build other Android components as
fast as possible. Changing the `WITH_TIDY=1` variable setting is also expensive
since the build.ninja file will be regenerated due to any such variable change.

To select only some directories or modules to compile with clang-tidy,
do not set the `WITH_TIDY=1` variable, but use the special `tidy-<directory>`
phony target. For example, a person working on `system/libbase` can build
Android quickly with
```
unset WITH_TIDY # Optional, not if you haven't set WITH_TIDY
make droid tidy-system-libbase
```

For any directory `d1/d2/d3`, a phony target tidy-d1-d2-d3 is generated
if there is any C/C++ source file under `d1/d2/d3`.

Note that with `make droid tidy-system-libbase`, some C/C++ files
that are not needed by the `droid` target will be passed to clang-tidy
if they are under `system/libbase`. This is like a `checkbuild`
under `system/libbase` to include all modules, but only C/C++
files of those modules are compiled with clang-tidy.

### The tidy-soong target

A special `tidy-soong` target is defined to include all C/C++
source files in *all* directories. This phony target is sometimes
used to test if all source files compile with a new clang-tidy release.

### The tidy-*_subset targets

A *subset* of each tidy-* phony target is defined to reduce test time.
Since any Android module, a C/C++ library or binary, can be built
for many different *variants*, one C/C++ source file is usually
compiled multiple times with different compilation flags.
Many of such *variant* flags have little or no effect on clang-tidy
checks. To reduce clang-tidy check time, a *subset* target like
`tidy-soong_subset` or `tidy-system-libbase_subset` is generated
to include only a subset, the first variant, of each module in
the directory.

Hence, for C/C++ source code quality, instead of a long
"make checkbuild", we can use "make tidy-soong_subset".


## Clang-tidy config files

When clang-tidy is invoked it looks up a *config file*,
which can contain *default* checks and options.
The Android build system checks the existence of default
and user specified tidy config files. If such files are
found the **global default** checks will not be supplied.
Hence adding `.clang-tidy` config files is the simplest way
to provide different tidy checks and options for a project.

The clang-tidy config files are useful in several other situations:
* To specify more clang-tidy config options that are not easy to
  do with command line flags or Android.bp properties.
  For example, some .clang-tidy files include a long list of `CheckOptions`
  that will be awkward to be passed through command line flags.
* For projects with many subdirectories and modules sharing the same clang-tidy
  checks and options, it is easier to put all such checks and options in one
  root directory .clang-tidy file. This will avoid changing many Android.bp
  files to share new `tidy_*` properties.
* For projects that are compiled on both Android and other platforms,
  the clang-tidy config file is the only way to work on all platforms.
* For faster edit-compile cycle, changing .clang-tidy files only recompile
  dependent C/C++ files with clang-tidy, but changing tidy properties in
  Android.bp files will trigger a slow regeneration of the build.ninja file.

### The default `.clang-tidy` config file

When clang-tidy is invoked to compile a source file in directory `d1/d2/d3/`,
it will look up the default config file `.clang-tidy` in directory `d1/d2/d3/`
and `d3`'s parent directories.
The first found `.clang-tidy` will be used.

If a clang-tidy config file sets `InheritParentConfig` to true,
clang-tidy will try to find and include a `.clang-tidy` file
in the parent directories.

Current Android build system will automatically look up all `.clang-tidy` files
in the source file directory and parent directories, and add them into the
implicit dependent files. This is a trade-off of fast generation of clang-tidy
build rules and accuracy of dependent list. Changing/touching a `.clang-tidy`
file in a parent directory might trigger more clang-tidy compilations,
but usually `.clang-tidy` files are very stable.

If such a `.clang-tidy` file is not found in the source file directory or
its parent directory, the Android build system will also look up such
files in the Android.bp directory and parent directories.
This will further reduce users the burden of specifying the `.clang-tidy`
file path to compile source files in other directories.
In this case, the `.clang-tidy` file path will be passed to clang-tidy
through the '-config-file' flag.

Current Android projects mostly do not have `.clang-tidy` file and rely on
the default global flags plus local tidy properties in the Android.bp file.
We believe that many such cases and future cases will use more `.clang-tidy`
files instead of local `tidy_*` properties.

### User defined config files and `tidy_config_file`

When clang-tidy is invoked with the `--config-file=` flag, the given
config file will be used instead of the default `.clang-tidy` file.
Adding `--config-file=` flag manually into the `tidy_flags` list
is error prone and disallowed.
Instead, a `tidy_config_file` property is provided to specify
the user tidy config file.

Note that `tidy_config_file` should contain a file path relative to
the Android *source tree root*. This path will be passed to clang-tidy
through the `--config_file=` flag.

The main use cases of `tidy_config_file` are rare and might include:
1. to work around any `.clang-tidy` file look up problems,
2. to avoid touching the `.clang-tidy` file imported from
   an upstream project.

For example, `project/xyz/tidy.config.txt` can be used with a declaration like
```
cc_defaults {
  name: "xyz_tidy_defaults",
  tidy_config_file: "project/xyz/tidy.config.txt",
}
```

Any module that inherits the `xyz_tidy_defaults` will use
`project/xyz/tidy.config.txt` as the clang-tidy config file,
for any module or source file location.


## Module clang-tidy properties

The global default can be overwritten by module properties in Android.bp.

### `tidy`, `tidy_checks`, and `ALLOW_LOCAL_TIDY_TRUE`

For example, in
[system/bpf/Android.bp](https://android.googlesource.com/platform/system/bpf/+/refs/heads/master/Android.bp),
clang-tidy is enabled explicitly and with a different check list:
```
cc_defaults {
    name: "bpf_defaults",
    // snipped
    tidy: true,
    tidy_checks: [
        "android-*",
        "cert-*",
        "-cert-err34-c",
        "clang-analyzer-security*",
        // Disabling due to many unavoidable warnings from POSIX API usage.
        "-google-runtime-int",
    ],
}
```
That means in normal builds, even without `WITH_TIDY=1`,
the modules that use `bpf_defaults` _should_ run clang-tidy
over C/C++ source files with the given `tidy_checks`.

Note that this *local* `tidy_checks` list is appended **after**
the **global default** checks, or the `Checks` list in
a tidy config file. To ignore all such global and config checks,
a local `tidy_checks` can use a `-*` check pattern followed
by the only checks for a local module.

However since clang-tidy warnings and its runtime cost might
not be wanted by all people, the default is to ignore the
`tidy:true` property unless the environment variable
`ALLOW_LOCAL_TIDY_TRUE` is set to true or 1.
To run clang-tidy on all modules that should be tested with clang-tidy,
`ALLOW_LOCAL_TIDY_TRUE` or `WITH_TIDY` should be set to true or 1.

Note that `clang-analyzer-security*` is included in `tidy_checks`
but not all `clang-analyzer-*` checks. Check `cert-err34-c` is
disabled, although `cert-*` is selected.

Some modules might want to disable clang-tidy even when
environment variable `WITH_TIDY=1` is set.
Examples can be found in
[system/netd/tests/Android.bp](https://android.googlesource.com/platform/system/netd/+/refs/heads/master/tests/Android.bp)
```
cc_test {
    name: "netd_integration_test",
    // snipped
    defaults: ["netd_defaults"],
    tidy: false,  // cuts test build time by almost 1 minute
```
and in
[bionic/tests/Android.bp](https://android.googlesource.com/platform/bionic/+/refs/heads/master/tests/Android.bp).
```
cc_test_library {
    name: "fortify_disabled_for_tidy",
    // snipped
    srcs: ["clang_fortify_tests.cpp"],
    tidy: false,
}
```

Note that `tidy:false` always disables clang-tidy, no matter
`ALLOW_LOCAL_TIDY_TRUE` is set or not.

### `tidy_checks_as_errors`

The global tidy checks are enabled as warnings.
If a C/C++ module wants to be free of certain clang-tidy warnings,
it can chose those checks to be treated as errors.
For example
[system/core/libsysutils/Android.bp](https://android.googlesource.com/platform/system/core/+/refs/heads/master/libsysutils/Android.bp)
has enabled clang-tidy explicitly, selected its own tidy checks,
and set three groups of tidy checks as errors:
```
cc_library {
    name: "libsysutils",
    // snipped
    tidy: true,
    tidy_checks: [
        "-*",
        "cert-*",
        "clang-analyzer-security*",
        "android-*",
    ],
    tidy_checks_as_errors: [
        "cert-*",
        "clang-analyzer-security*",
        "android-*",
    ],
    // snipped
}
```

An alternative of setting `tidy_checks_as_errors` in multiple modules and `.bp` files
is to add the `WarniningsAsErrors` list into a tidy config file.
Note that the `tidy_checks_as_errors` list is converted to command line
flag `--warnings-as-errors`, and this list is appended to
the `WarningsAsErrors` list in the tidy config file.

Note that Android build system will always append some global "disabled checks"
or *allowed-checks-as-warnings* to a clang-tidy command line through the
`--warnings-as-errors` flag.
This will let global builds to avoid stopping at buggy or
very noisy tidy checks.

### `tidy_flags` and `tidy_disabled_srcs`

Extra clang-tidy flags can be passed with the `tidy_flags` property.

Some Android modules use the `tidy_flags` to pass "-warnings-as-errors="
to clang-tidy. This usage should now be replaced with the
`tidy_checks_as_errors` property.

Some other tidy flags examples are `-format-style=` and `-header-filter=`
For example, in
[art/odrefresh/Android.bp](https://android.googlesource.com/platform/art/+/refs/heads/master/odrefresh/Android.bp),
we found
```
cc_defaults {
    name: "odrefresh-defaults",
    srcs: [
        "odrefresh.cc",
        "odr_common.cc",
        "odr_compilation_log.cc",
        "odr_fs_utils.cc",
        "odr_metrics.cc",
        "odr_metrics_record.cc",
    ],
    // snipped
    generated_sources: [
        "apex-info-list-tinyxml",
        "art-apex-cache-info",
        "art-odrefresh-operator-srcs",
    ],
    // snipped
    tidy: true,
    tidy_disabled_srcs: [":art-apex-cache-info"],
    tidy_flags: [
        "-format-style=file",
        "-header-filter=(art/odrefresh/|system/apex/)",
    ],
}
```
That means all modules with the `odrefresh-defaults` will
have clang-tidy enabled, but not for generated source
files in `art-apex-cache-info`.
The clang-tidy is called with extra flags to specify the
format-style and header-filter.

Note that the globally set default for header-filter is to
include only the module directory. So, the default clang-tidy
warnings for `art/odrefresh` modules will include source files
under that directory. Now `odrefresh-defaults` is interested
in seeing warnings from both `art/odrefresh/` and `system/apex/`
and it redefines `-header-filter` in its `tidy_flags`.

## Limit clang-tidy runtime

Some Android modules have large files that take a long time to compile
with clang-tidy, with or without the clang-analyzer checks.
To limit clang-tidy time, an environment variable can be set as
```base
WITH_TIDY=1 TIDY_TIMEOUT=90 make
```
This 90-second limit is actually the default time limit
in several Android continuous builds where `WITH_TIDY=1` and
`CLANG_ANALYZER_CHECKS=1` are set.

Similar to `tidy_disabled_srcs` a `tidy_timeout_srcs` list
can be used to include all source files that took too much time to compile
with clang-tidy. Files listed in `tidy_timeout_srcs` will not
be compiled by clang-tidy when `TIDY_TIMEOUT` is defined.
This can save global build time, when it is necessary to set some
time limit globally to finish in an acceptable time.
For developers who want to find all clang-tidy warnings and
are willing to spend more time on all files in a project,
they should not define `TIDY_TIMEOUT` and build only the wanted project directories.

## Capabilities for Android.bp and Android.mk

Some of the previously mentioned features are defined only
for modules in Android.bp files, not for Android.mk modules yet.

* The global `WITH_TIDY=1` variable will enable clang-tidy for all C/C++
  modules in Android.bp or Android.mk files.

* The global `TIDY_TIMEOUT` variable is recognized by Android prebuilt
  clang-tidy, so it should work for any clang-tidy invocation.

* The clang-tidy module level properties are defined for Android.bp modules.
  For Android.mk modules, old `LOCAL_TIDY`, `LOCAL_TIDY_CHECKS`,
  `LOCAL_TIDY_FLAGS` work similarly, but it would be better to convert
  those modules to use Android.bp files.

* The `tidy-*` phony targets are only generated for Android.bp modules.

* The look-up of default `.clang-tidy` files and adding them to implicit
  dependent files are available only to Android.bp modules.
  However, when clang-tidy is invoked to compile a .bp or .mk module, the
  clang-tidy tool always look up for the default `.clang-tidy` config file.
