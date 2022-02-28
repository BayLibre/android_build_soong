#!/usr/bin/env -S python -u
#
# Copyright (C) 2022 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Analyze bootclasspath_fragment usage."""
import argparse
import dataclasses
import json
import os
import re
import shutil
import subprocess
import tempfile
import typing
import sys

from signature_trie import signature_trie

_STUB_FLAGS_FILE = "out/soong/hiddenapi/hiddenapi-stub-flags.txt"

_FLAGS_FILE = "out/soong/hiddenapi/hiddenapi-flags.csv"

_INCONSISTENT_FLAGS = "ERROR: Hidden API flags are inconsistent:"


class BuildOperation:

    def __init__(self, popen):
        self.popen = popen
        self.returncode = None

    def lines(self):
        return iter(self.popen.stdout.readline, "")

    def wait(self, *args, **kwargs):
        self.popen.wait(*args, **kwargs)
        self.returncode = self.popen.returncode


@dataclasses.dataclass()
class FlagDiffs:
    """Encapsulates differences in flags reported by the build"""

    # Map from member signature to the (module flags, monolithic flags)
    diffs: typing.Dict[str, typing.Tuple[str, str]]


@dataclasses.dataclass()
class ModuleInfo:
    """Provides access to the generated module-info.json file.

    This is used to find the location of the file within which specific modules
    are defined.
    """

    modules: typing.Dict[str, typing.Dict[str, typing.Any]]

    @staticmethod
    def load(filename):
        with open(filename, "r", encoding="utf8") as f:
            j = json.load(f)
            return ModuleInfo(j)

    def _module(self, module_name):
        """Find module by name in module-info.json file"""
        if module_name in self.modules:
            return self.modules[module_name]

        raise Exception(f"Module {module_name} could not be found")

    def module_path(self, module_name):
        module = self._module(module_name)
        # The "path" is actually a list of paths, one for each class of module
        # but as the modules are all created from bp files if a module does
        # create multiple classes of make modules they should all have the same
        # path.
        paths = module["path"]
        unique_paths = set(paths)
        if len(unique_paths) != 1:
            raise Exception(f"Expected module '{module_name}' to have a "
                            f"single unique path but found {unique_paths}")
        return paths[0]


@dataclasses.dataclass()
class BcpfAnalyzer:
    # Directory pointed to by ANDROID_BUILD_OUT
    top_dir: str

    # Directory pointed to by OUT_DIR of {top_dir}/out if that is not set.
    out_dir: str

    # Directory pointed to by ANDROID_PRODUCT_OUT.
    product_out_dir: str

    # The name of the bootclasspath_fragment module.
    bcpf: str

    # The name of the apex module containing {bcpf}, only used for
    # informational purposes.
    apex: str

    # The name of the sdk module containing {bcpf}, only used for
    # informational purposes.
    sdk: str

    # The log file into which additional debug output, including output from
    # sub commands is logged.
    log_file: typing.TextIO = None

    # All the signatures, loaded from all-flags.csv, initialized by
    # load_all_flags().
    _signatures: typing.Set[str] = dataclasses.field(default_factory=set)

    # All the classes, loaded from all-flags.csv, initialized by
    # load_all_flags().
    _classes: typing.Set[str] = dataclasses.field(default_factory=set)

    # Information loaded from module-info.json, initialized by
    # load_module_info().
    module_info: ModuleInfo = None

    @staticmethod
    def reformat_report_test(text):
        return re.sub(r"(.)\n([^\s])", r"\1 \2", text)

    def report(self, text, **kwargs):
        # Concatenate lines that are not separated by a blank line together to
        # eliminate formatting applied to the supplied text to adhere to python
        # line length limitations.
        text = self.reformat_report_test(text)
        self.log(text, **kwargs)
        print(text, **kwargs)

    def log(self, text, **kwargs):
        if self.log_file:
            print(f"DEBUG: {text}", file=self.log_file, **kwargs)

    def run_command(self, cmd, *args, **kwargs):
        cmd_line = " ".join(cmd)
        self.log(f"Running {cmd_line}")
        subprocess.run(
            *args,
            check=True,
            cwd=self.top_dir,
            stderr=subprocess.STDOUT,
            stdout=self.log_file,
            text=True,
            **kwargs)

    @property
    def signatures(self):
        if not self._signatures:
            raise Exception("signatures has not been initialized")
        return self._signatures

    @property
    def classes(self):
        if not self._classes:
            raise Exception("classes has not been initialized")
        return self._classes

    def load_all_flags(self):
        all_flags = self.find_bootclasspath_fragment_output_file(
            "all-flags.csv")

        # Extract the set of signatures and a separate set of classes produced
        # by the bootclasspath_fragment.
        with open(all_flags, "r", encoding="utf8") as f:
            for line in iter(f.readline, ""):
                signature = self.line_to_signature(line)
                self._signatures.add(signature)
                class_name = self.signature_to_class(signature)
                self._classes.add(class_name)

    def load_module_info(self):
        module_info_file = os.path.join(self.product_out_dir,
                                        "module-info.json")
        self.report(f"""
Making sure that {module_info_file} is up to date.
""")
        output = self.build_file_read_output(module_info_file)
        lines = output.lines()
        for line in lines:
            self.log(line, end="")
        abs_module_info_file = os.path.join(self.top_dir, module_info_file)
        self.module_info = ModuleInfo.load(abs_module_info_file)

    @staticmethod
    def line_to_signature(line):
        return line.split(",")[0]

    @staticmethod
    def signature_to_class(signature):
        return signature.split(";->")[0]

    @staticmethod
    def to_parent_package(pkg_or_class):
        return pkg_or_class.rsplit("/", 1)[0]

    def module_path(self, module_name):
        return self.module_info.module_path(module_name)

    def module_out_dir(self, module_name):
        module_path = self.module_path(module_name)
        return os.path.join(self.out_dir, "soong/.intermediates", module_path,
                            module_name)

    def find_bootclasspath_fragment_output_file(self, basename):
        # Find the output file of the bootclasspath_fragment with the specified
        # base name.
        found_file = ""
        bcpf_out_dir = self.module_out_dir(self.bcpf)
        for (dirpath, _, filenames) in os.walk(bcpf_out_dir):
            for f in filenames:
                if f == basename:
                    found_file = os.path.join(dirpath, f)
                    break
        if not found_file:
            raise Exception(f"Could not find {basename} in {bcpf_out_dir}")
        return found_file

    def analyze(self):
        """Analyze a bootclasspath_fragment module.

        Provides help in resolving any existing issues and provides
        optimizations that can be applied.
        """
        self.report(f"Analyzing bootclasspath_fragment module {self.bcpf}")
        self.report(f"""
Run this tool to help initialize a bootclasspath_fragment module. Before you
start make sure that:

1. The current checkout is up to date.

2. The environment has been initialized using lunch, e.g.
   lunch aosp_arm64-userdebug

3. You have added a bootclasspath_fragment module to the appropriate Android.bp
file. Something like this:

   bootclasspath_fragment {{
     name: "{self.bcpf}",
     contents: [
       "...",
     ],

     // The bootclasspath_fragments that provide APIs on which this depends.
     fragments: [
       {{
         apex: "com.android.art",
         module: "art-bootclasspath-fragment",
       }},
     ],
   }}

4. You have added it to the platform_bootclasspath module in
frameworks/base/boot/Android.bp. Something like this:

   platform_bootclasspath {{
     name: "platform-bootclasspath",
     fragments: [
       ...
       {{
         apex: "{self.apex}",
         module: "{self.bcpf}",
       }},
     ],
   }}

5. You have added an sdk module. Something like this:

   sdk {{
     name: "{self.sdk}",
     bootclasspath_fragments: ["{self.bcpf}"],
   }}
""")

        # Make sure that the module-info.json file is up to date.
        self.load_module_info()

        self.report("""
Cleaning potentially stale files.
""")
        # Remove the out/soong/hiddenapi files.
        shutil.rmtree(f"{self.out_dir}/soong/hiddenapi", ignore_errors=True)

        # Remove any bootclasspath_fragment output files.
        shutil.rmtree(self.module_out_dir(self.bcpf), ignore_errors=True)

        self.build_monolithic_stubs_flags()
        self.build_monolithic_flags()
        self.analyze_hiddenapi_package_properties()

    def check_inconsistent_flag_lines(self, significant, module_line,
                                      monolithic_line, separator_line):
        if not (module_line.startswith("< ") and
                monolithic_line.startswith("> ") and not separator_line):
            # Something went wrong.
            self.report(f"""Invalid build output detected:
  module_line: "{module_line}"
  monolithic_line: "{monolithic_line}"
  separator_line: "{separator_line}"
""")
            sys.exit(1)

        if significant:
            self.log(module_line)
            self.log(monolithic_line)
            self.log(separator_line)

    def scan_inconsistent_flags_report(self, lines):
        """Scans a hidden API flags report

        The hidden API inconsistent flags report which looks something like
        this.

        < out/soong/.intermediates/.../filtered-stub-flags.csv
        > out/soong/hiddenapi/hiddenapi-stub-flags.txt

        < Landroid/compat/Compatibility;->clearOverrides()V
        > Landroid/compat/Compatibility;->clearOverrides()V,core-platform-api

        """
        module_line = next(lines).rstrip()
        significant = False
        bcpf_dir = self.module_info.module_path(self.bcpf)
        if os.path.join(bcpf_dir, self.bcpf) in module_line:
            # These errors are related to the bcpf being analyzed so
            # keep them.
            significant = True
        else:
            self.report(f"Filtering out errors related to {module_line}")

        monolithic_line = next(lines).rstrip()
        separator_line = next(lines).rstrip()

        self.check_inconsistent_flag_lines(significant, module_line,
                                           monolithic_line, separator_line)

        diffs = {}
        for line in lines:
            module_line = line.rstrip()
            monolithic_line = next(lines).rstrip()
            separator_line = next(lines).rstrip()

            self.check_inconsistent_flag_lines(significant, module_line,
                                               monolithic_line, "")

            module_parts = module_line.removeprefix("< ").split(",")
            module_signature = module_parts[0]
            module_flags = module_parts[1:]

            monolithic_parts = monolithic_line.removeprefix("> ").split(",")
            monolithic_signature = monolithic_parts[0]
            monolithic_flags = monolithic_parts[1:]

            if module_signature != monolithic_signature:
                # Something went wrong.
                self.report(f"""Inconsistent signatures detected:
  module_signature: "{module_signature}"
  monolithic_signature: "{monolithic_signature}"
""")
                sys.exit(1)

            diffs[module_signature] = (module_flags, monolithic_flags)

            if separator_line:
                # If the separator line is not blank then it is the end of the
                # current report, and possibly the start of another.
                return separator_line, diffs

        return "", diffs

    def build_file_read_output(self, filename):
        # Make sure the filename is relative to top if possible as the build
        # may be using relative paths as the target.
        rel_filename = filename.removeprefix(self.top_dir)
        cmd = ["build/soong/soong_ui.bash", "--make-mode", rel_filename]
        cmd_line = " ".join(cmd)
        self.log(f"{cmd_line}")
        # pylint: disable=consider-using-with
        output = subprocess.Popen(
            cmd,
            cwd=self.top_dir,
            stderr=subprocess.STDOUT,
            stdout=subprocess.PIPE,
            text=True,
        )
        return BuildOperation(popen=output)

    def build_hiddenapi_flags(self, filename):
        output = self.build_file_read_output(filename)

        lines = output.lines()
        diffs = None
        for line in lines:
            line = line.rstrip()
            self.log(line)
            while line == _INCONSISTENT_FLAGS:
                line, diffs = self.scan_inconsistent_flags_report(lines)

        output.wait(timeout=10)
        if output.returncode != 0:
            self.log(f"Command failed with {output.returncode}")
        else:
            self.log("Command succeeded")

        return diffs

    def build_monolithic_stubs_flags(self):
        self.report(f"""
Attempting to build {_STUB_FLAGS_FILE} to verify that the
bootclasspath_fragment has the correct API stubs available...
""")

        # Build the hiddenapi-stubs-flags.txt file.
        diffs = self.build_hiddenapi_flags(_STUB_FLAGS_FILE)
        if diffs:
            self.report(f"""
There is a discrepancy between the stub API derived flags created by the
bootclasspath_fragment and the platform_bootclasspath. See preceding error
messages to see which flags are inconsistent. The inconsistencies can occur for
a couple of reasons:

If you are building against prebuilts of the Android SDK, e.g. by using
TARGET_BUILD_APPS then the prebuilt versions of the APIs this
bootclasspath_fragment depends upon are out of date and need updating. See
go/update-prebuilts for help.

Otherwise, this is happening because there are some stub APIs that are either
provided by or used by the contents of the bootclasspath_fragment but which are
not available to it. There are 4 ways to handle this:

1. A java_sdk_library in the contents property will automatically make its stub
   APIs available to the bootclasspath_fragment so nothing needs to be done.

2. If the API provided by the bootclasspath_fragment is created by an api_only
   java_sdk_library (or a java_library that compiles files generated by a
   separate droidstubs module then it cannot be added to the contents and
   instead must be added to the api.stubs property, e.g.

   bootclasspath_fragment {{
     name: "{self.bcpf}",
     ...
     api: {{
       stubs: ["$MODULE-api-only"],"
     }},
   }}

3. If the contents use APIs provided by another bootclasspath_fragment then
   it needs to be added to the fragments property, e.g.

   bootclasspath_fragment {{
     name: "{self.bcpf}",
     ...
     // The bootclasspath_fragments that provide APIs on which this depends.
     fragments: [
       ...
       {{
         apex: "com.android.other",
         module: "com.android.other-bootclasspath-fragment",
       }},
     ],
   }}

4. If the contents use APIs from a module that is not part of another
   bootclasspath_fragment then it must be added to the additional_stubs
   property, e.g.

   bootclasspath_fragment {{
     name: "{self.bcpf}",
     ...
     additional_stubs: ["android-non-updatable"],
   }}

   Like the api.stubs property these are typically java_sdk_library modules but
   can be java_library too.

   Note: The "android-non-updatable" is treated as if it was a java_sdk_library
   which it is not at the moment but will be in future.
""")

        return diffs

    def build_monolithic_flags(self):
        self.report(f"""
Attempting to build {_FLAGS_FILE} to verify that the
bootclasspath_fragment has the correct hidden API flags...
""")

        property_snippet = ""

        # Build the hiddenapi-flags.csv file and extract any differences in
        # the flags between this bootclasspath_fragment and the monolithic
        # files.
        diffs = self.build_hiddenapi_flags(_FLAGS_FILE)

        # Load information from the bootclasspath_fragment's all-flags.csv file.
        self.load_all_flags()

        if diffs:
            self.report(f"""
There is a discrepancy between the hidden API flags created by the
bootclasspath_fragment and the platform_bootclasspath. See preceding error
messages to see which flags are inconsistent. The inconsistencies can occur for
a couple of reasons:

If you are building against prebuilts of this bootclasspath_fragment then the
prebuilt version of the sdk snapshot (specifically the hidden API flag files)
are inconsistent with the prebuilt version of the apex {self.apex}. Please
ensure that they are both updated from the same build.

1. There are custom hidden API flags specified in the one of the files in
   frameworks/base/boot/hiddenapi which apply to the bootclasspath_fragment but
   which are not supplied to the bootclasspath_fragment module.

2. The bootclasspath_fragment specifies invalid "package_prefixes" or
   "split_packages" properties that match packages and classes that it does not
   provide.

""")

            # Check to see if there are any hiddenapi related properties that
            # need to be added to the
            self.report("""
Checking custom hidden API flags....
""")
            bcpf_properties = self.check_frameworks_base_boot_hidden_api_files()

            # If there were any flags that needed to be copied across then
            # output instructions on what to do.
            if bcpf_properties:
                property_snippet = "".join(
                    [f"        {x},\n" for x in bcpf_properties]).rstrip()

                bcpf_dir = self.module_info.module_path(self.bcpf)
                self.report(f"""
Add the following snippet into the {self.bcpf} bootclasspath_fragment module
in the {bcpf_dir}/Android.bp file. If the hidden_api block already exists then
merge these properties into it.

    hidden_api: {{
{property_snippet},
    }},
""")

        return diffs, property_snippet

    basenameToBcpfProperty = {
        "hiddenapi-max-target-o.txt":
            "hiddenapi-max-target-o-low-priority.txt",
        "hiddenapi-max-target-r-loprio.txt":
            "hiddenapi-max-target-r-low-priority.txt",
    }

    def check_frameworks_base_boot_hidden_api_files(self):
        bcpf_dir = self.module_info.module_path(self.bcpf)
        bcpf_hidden_api_dir = os.path.join(self.top_dir, bcpf_dir, "hiddenapi")
        os.makedirs(bcpf_hidden_api_dir, exist_ok=True)

        hiddenapi_dir = os.path.join(self.top_dir,
                                     "frameworks/base/boot/hiddenapi")
        bcpf_properties = []
        for basename in os.listdir(hiddenapi_dir):
            if not (basename.startswith("hiddenapi-") and
                    basename.endswith(".txt")):
                continue

            flags_file = os.path.join(hiddenapi_dir, basename)

            self.log(f"Checking {flags_file} for flags related to {self.bcpf}")

            # Map the file name in frameworks/base/boot/hiddenapi into a
            # slightly more meaningful name for use by the
            # bootclasspath_fragment.
            if basename == "hiddenapi-max-target-o.txt":
                basename = "hiddenapi-max-target-o-low-priority.txt"
            elif basename == "hiddenapi-max-target-r-loprio.txt":
                basename = "hiddenapi-max-target-r-low-priority.txt"

            # Read the file in frameworks/base/boot/hiddenapi/<file> copy any
            # flags that relate to the bootclasspath_fragment into a local
            # file in the hiddenapi subdirectory.
            tmp_flags_file = flags_file + ".tmp"
            bcpf_flags_file = os.path.join(bcpf_hidden_api_dir, basename)
            matched_signature = False
            # Open the flags file to read the flags from.
            with open(flags_file, "r", encoding="utf8") as f:
                # Open a temporary file to write the flags (minus any removed
                # flags).
                with open(tmp_flags_file, "w", encoding="utf8") as t:
                    # Open the bootclasspath_fragment file for append just in
                    # case it already exists.
                    with open(bcpf_flags_file, "a", encoding="utf8") as b:
                        for line in iter(f.readline, ""):
                            signature = line.rstrip()
                            if signature in self.signatures:
                                # The signature is provided by the
                                # bootclasspath_fragment so copy it to the
                                # bootclasspath_fragment specific file.
                                print(line, file=b, end="")
                                matched_signature = True
                            else:
                                # The signature is NOT provided by the
                                # bootclasspath_fragment so copy it to the
                                # new monolithic specific file.
                                print(line, file=t, end="")

            # If the bootclasspath_fragment specific flags file is not empty
            # then it contains flags. That could either be new flags just moved
            # from frameworks/base or previous contents of the file. In either
            # case the file must not be removed.
            if os.path.getsize(bcpf_flags_file):
                # There are custom flags related to the bootclasspath_fragment
                # so replace the frameworks/base/boot/hiddenapi file with the
                # file that does not contain those flags.
                shutil.move(tmp_flags_file, flags_file)

                property_name = basename.removeprefix("hiddenapi-")
                property_name = property_name.removesuffix(".txt")
                property_name = property_name.replace("-", "_")

                bcpf_properties.append(
                    f'{property_name}: ["hiddenapi/{basename}"]')

                if matched_signature:
                    # Make sure that the files are sorted.
                    self.run_command([
                        "tools/platform-compat/hiddenapi/sort_api.sh",
                        bcpf_flags_file,
                    ])
                    self.report(f"Moved additional custom hidden API flags for "
                                f"{self.bcpf} from frameworks/base. Please "
                                f"make sure to upload changes from there.")

            else:
                # There are no custom flags related to the
                # bootclasspath_fragment so clean up the working files.
                os.remove(tmp_flags_file)
                os.remove(bcpf_flags_file)

        return bcpf_properties

    BCPF = "bcpf"
    OTHER = "other"
    FAKE_MEMBER = ";->fake()V"

    def analyze_hiddenapi_package_properties(self):
        split_packages, single_packages, package_prefixes = \
            self.compute_hiddenapi_package_properties()

        # TODO(b/202154151): Find those classes in split packages that are not
        #  part of an API, i.e. are an internal implementation class, and so
        #  can, and should, be safely moved out of the split packages.

        hiddenapi_snippet = ""
        if split_packages:
            split_packages_snippet = "\n".join(
                [f"""            "{x}",""" for x in split_packages])
            hiddenapi_snippet += f"""
        // The following packages contain classes from other modules on the
        // bootclasspath. That means that the hidden API flags for this module
        // has to explicitly list every single class this module provides in
        // that package to differentiate them from the classes provided by other
        // modules. That can include private classes that are not part of the
        // API.
        split_packages: [
{split_packages_snippet}
        ],
"""
            self.report(f"""
bootclasspath_fragment {self.bcpf} contains classes in packages that also
contain classes provided by other sources, those packages are called split
packages. Split packages should be avoided where possible but are often
unavoidable when modularizing existing code.

The hidden api processing needs to know which packages are split (and conversely
which are not) so that it can optimize the hidden API flags to remove
unnecessary implementation details.
""")
        else:
            hiddenapi_snippet += """
        // This module does not contain any split packages.
        split_packages: [],
"""

        self.report("""
By default (for backwards compatibility) the bootclasspath_fragment assumes that
all packages are split unless one of the package_prefixes or split_packages
properties are specified. While that is safe it is not optimal and can lead to
unnecessary implementation details leaking into the hidden API flags. Adding an
empty split_packages property allows the flags to be optimized and remove any
unnecessary implementation details.
""")

        if single_packages:
            single_packages_snippet = "".join(
                [f"""            "{x}",""" for x in single_packages])
            hiddenapi_snippet += f"""
        // The following packages currently only contain classes from this
        // bootclasspath_fragment but some of their sub-packages contain classes
        // from other bootckasspath modules. Packages should only be listed here
        // when necessary for legacy purposes, new packages should match a
        // package prefix.
        single_packages: [
{single_packages_snippet}
        ],
"""

        if package_prefixes:
            package_prefixes_snippet = "\n".join(
                [f"""            "{x}",""" for x in package_prefixes])
            hiddenapi_snippet += f"""
        // The following packages and all their subpackages currently only
        // contain classes from this bootclasspath_fragment. Listing a package
        // here won't prevent other bootclasspath modules from adding classes in
        // any of those packages but it will prevent them from adding those
        // classes into an API surface, e.g. public, system, etc.. Doing so will
        // result in a build failure due to inconsistent flags.
        package_prefixes: [
{package_prefixes_snippet}
        ],
"""

        # Remove leading and trailing blank lines.
        hiddenapi_snippet = hiddenapi_snippet.strip("\n")

        bcpf_dir = self.module_info.module_path(self.bcpf)
        self.report(f"""
Add the following snippet into the {self.bcpf} bootclasspath_fragment module
in the {bcpf_dir}/Android.bp file. If the hidden_api block already exists then
merge these properties into it.

    hidden_api: {{
{hiddenapi_snippet}
    }},
""")

        signature_patterns_files = self.find_bootclasspath_fragment_output_file(
            "signature-patterns.csv").removeprefix(self.top_dir)

        self.report(f"""
The purpose of the hiddenapi split_packages and package_prefixes properties is
to allow the removal of implementation details from the hidden API flags to
reduce the coupling between sdk snapshots and the APEX runtime. It cannot
eliminate that coupling completely though. Doing so may require changes to the
code.

This tool provides support for managing those properties but it cannot decide
whether the set of package prefixes suggested is appropriate that needs the
input of the developer.

Please run the following command:
    m {signature_patterns_files}

And then check the '{signature_patterns_files}' for any mention of
implementation classes and packages (i.e. those classes/packages that do not
contain any part of an API surface, including the hidden API). If they are
found then the code should ideally be moved to a package unique to this module
that is contained within a package that is part of an API surface.

The format of the file is a list of patterns:

* Patterns for split packages will list every class in that package.

* Patterns for package prefixes will end with .../**.

* Patterns for packages which are not split but cannot use a package prefix
because there are sub-packages which are provided by another module will end
with .../*.
""")

    def compute_hiddenapi_package_properties(self):
        trie = signature_trie()
        # Populate the trie with the classes that are provided by the
        # bootclasspath_fragment tagging them to make it clear where they
        # are from.
        sorted_classes = sorted(self.classes)
        for class_name in sorted_classes:
            trie.add(class_name + self.FAKE_MEMBER, self.BCPF)

        monolithic_classes = set()
        abs_flags_file = os.path.join(self.top_dir, _FLAGS_FILE)
        with open(abs_flags_file, "r", encoding="utf8") as f:
            for line in iter(f.readline, ""):
                signature = self.line_to_signature(line)
                class_name = self.signature_to_class(signature)
                if (class_name not in monolithic_classes and
                        class_name not in self.classes):
                    trie.add(
                        class_name + self.FAKE_MEMBER,
                        self.OTHER,
                        only_if_matches=True)
                    monolithic_classes.add(class_name)

        split_packages = []
        single_packages = []
        package_prefixes = []
        self.recurse_hiddenapi_packages_trie(trie, split_packages,
                                             single_packages, package_prefixes)
        return split_packages, single_packages, package_prefixes

    def recurse_hiddenapi_packages_trie(self, node, split_packages,
                                        single_packages, package_prefixes):
        nodes = node.child_nodes()
        if nodes:
            for child in nodes:
                # Ignore any non-package nodes.
                if child.type != "package":
                    continue

                package = child.selector.replace("/", ".")

                providers = set(child.get_matching_rows("**"))
                if not providers:
                    # The package and all its sub packages contain no
                    # classes. This should never happen.
                    pass
                elif providers == {self.BCPF}:
                    # The package and all its sub packages only contain
                    # classes provided by the bootclasspath_fragment.
                    self.log(f"Package '{package}.**' is not split")
                    package_prefixes.append(package)
                    # There is no point traversing into the sub packages.
                    continue
                elif providers == {self.OTHER}:
                    # The package and all its sub packages contain no
                    # classes provided by the bootclasspath_fragment.
                    # There is no point traversing into the sub packages.
                    self.log(f"Package '{package}.**' contains no classes from "
                             f"{self.bcpf}")
                    continue
                elif self.BCPF in providers:
                    # The package and all its sub packages contain classes
                    # provided by the bootclasspath_fragment and other
                    # sources.
                    self.log(f"Package '{package}.**' contains classes from "
                             f"{self.bcpf} and other sources")

                providers = set(child.get_matching_rows("*"))
                if not providers:
                    # The package contains no classes.
                    self.log(f"Package: {package} contains no classes")
                elif providers == {self.BCPF}:
                    # The package only contains classes provided by the
                    # bootclasspath_fragment.
                    self.log(f"Package '{package}.*' is not split")
                    single_packages.append(package)
                elif providers == {self.OTHER}:
                    # The package contains no classes provided by the
                    # bootclasspath_fragment. Child nodes make contain such
                    # classes.
                    self.log(f"Package '{package}.*' contains no classes from "
                             f"{self.bcpf}")
                elif self.BCPF in providers:
                    # The package contains classes provided by both the
                    # bootclasspath_fragment and some other source.
                    self.log(f"Package '{package}.*' is split")
                    split_packages.append(package)

                self.recurse_hiddenapi_packages_trie(child, split_packages,
                                                     single_packages,
                                                     package_prefixes)


def main(argv):
    args_parser = argparse.ArgumentParser(
        description="Analyze a bootclasspath_fragment module.")
    args_parser.add_argument(
        "--bcpf",
        help="The bootclasspath_fragment module to analyze",
        required=True,
    )
    args_parser.add_argument(
        "--apex",
        help="The apex module to which the bootclasspath_fragment belongs. It "
        "is not strictly necessary at the moment but providing it will "
        "allow this script to give more useful messages and it may be"
        "required in future.",
        default="SPECIFY-APEX-OPTION")
    args_parser.add_argument(
        "--sdk",
        help="The sdk module to which the bootclasspath_fragment belongs. It "
        "is not strictly necessary at the moment but providing it will "
        "allow this script to give more useful messages and it may be"
        "required in future.",
        default="SPECIFY-SDK-OPTION")
    args = args_parser.parse_args(argv[1:])
    top_dir = os.environ["ANDROID_BUILD_TOP"] + "/"
    out_dir = os.environ.get("OUT_DIR", os.path.join(top_dir, "out"))
    product_out_dir = os.environ.get("ANDROID_PRODUCT_OUT", top_dir)
    # Make product_out_dir relative to the top so it can be used as part of a
    # build target.
    product_out_dir = product_out_dir.removeprefix(top_dir)
    log_fd, abs_log_file = tempfile.mkstemp(
        suffix="_analyze_bcpf.log", text=True)
    with os.fdopen(log_fd, "w") as log_file:
        print(f"Writing log to {abs_log_file}")
        analyzer = BcpfAnalyzer(
            top_dir=top_dir,
            out_dir=out_dir,
            product_out_dir=product_out_dir,
            bcpf=args.bcpf,
            apex=args.apex,
            sdk=args.sdk,
            log_file=log_file,
        )
        analyzer.analyze()
        print(f"Log written to {abs_log_file}")


if __name__ == "__main__":
    main(sys.argv)
