# Soong Compliance

"Compliance" presently refers to opensource license compliance, but could refer
to other forms of compliance in the future.

## Project Metadata

The `project_metadata_proto/project_metadata.proto` protobuf defines the fields
used in METADATA or METADATA.android files to name and to describe software
from external sources. It is a proper subset of the similar protobuf used
internally at Google.

The file format answers questions like: What is the project called? Where does
the source code come from? Where can I find more information? What version are
we using? What sort of licensing broadly applies?

## License Metadata

### Background

To be useful, the license metadata has to form a parallel or shadow graph of
the primary build graph. Because the shadow graph does not depend on any of the
artifacts of the primary build graph, any (or every) node of the build graph
can depend on any (or every) node of the shadow graph without causing cycles.

At least conceptually.

In this way, an apex file can contain a notice file that depends on the apex
file's own license metadata graph, for example.

In Bazel, the shadow license metadata graph is implemented as something called
an `aspect`.

The Android build system represents the primary build graph with three
different build languages: Make, Soong, and Ninja. Each have slightly different
models:

*   Make has a model of a single output artifact file that depends on any
    number of input dependency files. Rules can create other side-effect files
    but if anything depends on a side-effect file that does not yet exist, Make
    might stop with an error not knowing which rule to run to build it.

*   Soong has a model of `modules` with unspecified internal complexity that
    depend on other modules. At a file level, a module can depend on any number
    of input files and output any number of derived artifact files.

*   Ninja has a model of rules with any number of input dependency files and
    any number of output artifact files.

The Make part of the Android build system uses a lot of text variables and
macro expansion meta-programming to map between the `module` and `file`
abstraction boundaries so that it is compatible with Soong. But it also has
numerous ways to create non-module, regular Make build rules. For example,
during the product config stage, it is common to append patterns to variables
that basically say: "When you get around to creating build rules, define a rule
that creates file B by copying file A" etc.

The license metadata graph for Android has to be expressive enough to capture
the meaning of all of these representations. Because Make and Ninja ultimately
operate on files, the license metadata graph has to create file artifacts.

### Implementation

#### File format

The `license_metadata_proto/license_metadata.proto` protobuf defines the fields
that appear in license metadata files. Under the `out/` directory, license
metadata filenames always end in `meta_lic`. In some cases, the name ends with
`.meta_lic`, and in other cases, the entire name is `meta_lic` inside a
target-specific directory. But after running a build, it is possible to locate
all of the license metadata by running:

```shell
$ find out/ -name '*meta_lic' -type f
```

in the `$ANDROID_BUILD_TOP` directory.

For Soong modules and their Make equivalents, each module has one license
metadata file. Non-module targets in Make have a license metadata file for each
target file. When implemented, Ninja rules will have a license metadata file
for each instantiated ninja rule.

The deps sub-message establishes the dependency graph among license metadata
files. The annotations on the deps sub-message differentiate between build
tools, runtime dependencies, and derived works.

A module can have multiple output artifacts and it can matter which is actually
used so the license metadata file also has a `sources` field. When matched with
the `deps` that produce the `sources`, the dependency graph of files is
established.

Copyleft licenses require special handling. Their source-sharing requirements
generally propagate to anything combined with them, but they make exceptions
for mere aggregation on distribution media. The `is_container` field is true
for targets that are mere aggregations for distribution. e.g. for .img disk
images or for .zip compressed archives etc. The `install_map` sub-message maps
between names of source files and their paths inside the distribution medium.

The remainder of the fields are straightforward. The `built` and `installed`
fields identify output artifacts. The `projects` field identifies the git
projects that contain the blueprints or make files that define the module.
In an ideal world, there would be at most 1 such project, but Android sometimes
has both the source code for a target and a prebuilt image for the same target
in different projects.

The remainder of the fields describe the target and its associated license
information.

#### Binaries

The `build_license_metadata` binary constructs a license metadata file based on
the arguments given by the build system. The `copy_license_metadata` binary
constructs a license metadata for a file that is a copy of another file based
on the other file's license metadata. Some fields are preserved, and some are
erased.
