#!/usr/bin/env python

# Copyright (C) 2025 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

""""
This is an intermediary between rust-analyzer and the soong/ninja build system.
This file should be called when a Rust source file is updated, it will then parse
the rust-target-mapping.json file to figure out what ninja target(s) need to be built,
build them, parse the outputted error/warning diagnostics, and echo the diagnostics to 
stdout for rust-analyzer to parse.
"""

import json
import os
import platform
import subprocess
import sys
from argparse import ArgumentParser
from pathlib import Path
from typing import Dict, List


def load_soong_env_variables(android_build_top: Path, file_name: str) -> Dict[str, str]:
    soong_env_file = android_build_top / "out" / "soong" / file_name
    try:
        with open(soong_env_file, "r") as file:
            entries = json.load(file)
        return {entry["Key"]: entry["Value"] for entry in entries}
    except FileNotFoundError:
        raise Exception("fCould not find {file_name} file")


def load_check_targets(android_build_top: Path, source_file: Path) -> List[str]:
    path = android_build_top / "out" / "soong" / "rust-target-mapping.json"
    try:
        with open(path, "r") as file:
            mapping_entries = json.load(file)
        return [
            entry["check_target"]
            for entry in mapping_entries
            if str(source_file).startswith(entry["source_dir"])
        ]
    except FileNotFoundError:
        raise Exception(f"Could not open {path}")


# Returns the file name of the error file for a given check target.
#
# This is outputted by the ".checkJson" static build rule in the soong build system.
def error_file(check_target: str) -> str:
    return f"{check_target}.error"


def execute_soong_ui(
    android_build_top: Path, check_targets: List[str], soong_env: Dict[str, str]
) -> subprocess.CompletedProcess:
    # Command to run soong_ui.bash.
    #   1) --config-only [0]:
    #     This will skip running soong to generate a ninja file.
    #   2) --skip-config [1]:
    #     This will skip the kati config step.
    #   3) --soong-only [2]:
    #     This will skip the kati, kati ninja and ninja build steps.
    #   4) --skip-soong-tests [3]:
    #     This will skip the soong tests step.
    # [0]: https://cs.android.com/android/platform/superproject/main/+/02e2220f048f80a4b19db908bf5c842cc7dba939:build/soong/ui/build/config.go;l=886
    # [1]: https://cs.android.com/android/platform/superproject/main/+/02e2220f048f80a4b19db908bf5c842cc7dba939:build/soong/ui/build/config.go;l=890
    # [2]: https://cs.android.com/android/platform/superproject/main/+/02e2220f048f80a4b19db908bf5c842cc7dba939:build/soong/ui/build/config.go;l=871
    # [3]: https://cs.android.com/android/platform/superproject/main/+/02e2220f048f80a4b19db908bf5c842cc7dba939:build/soong/ui/build/config.go;l=892
    soong_ui_path = android_build_top / "build" / "soong" / "soong_ui.bash"
    soong_ui_cmd = [
        str(soong_ui_path),
        "--make-mode",
        "--config-only",
        "--skip-config",
        "--soong-only",
        "--skip-soong-tests",
        *check_targets,
    ]

    # Include all environment variables from "soong.environment.used.build" file. Since
    # soong_ui.bash will rebuild the ninja file if the environment variables change, we need to ensure
    # that we pass in the same environment variables that were used to build the ninja file to not cause
    # a rebuild.
    return subprocess.run(
        soong_ui_cmd, env=soong_env, text=True, stdout=sys.stderr, stderr=sys.stderr
    )


def execute_ninja(
    android_build_top: Path, check_targets: List[str], soong_env: Dict[str, str]
) -> subprocess.CompletedProcess:
    target_product = soong_env["TARGET_PRODUCT"]
    system_name = platform.system()

    if system_name == "Darwin":
        host_arch = "darwin-x86"
    elif system_name == "Linux":
        host_arch = "linux-x86"
    else:
        raise Exception(f"Unsupported host platform: {system_name}")

    ninja_path = (
        android_build_top / "prebuilts" / "build-tools" / host_arch / "bin" / "ninja"
    )

    ninja_target_file = android_build_top / "out" / f"combined-{target_product}.ninja"

    ninja_command = [
        str(ninja_path),
        "-f",
        str(ninja_target_file),
        "-k",
        "0",
        *check_targets,
    ]

    return subprocess.run(
        ninja_command, env=os.environ, text=True, stdout=sys.stderr, stderr=sys.stderr
    )


def main():
    parser = ArgumentParser(
        description="Rust-analyzer integration for rustc/clippy-driver binary"
    )
    parser.add_argument(
        "source_file",
        type=Path,
        help="The absolute path of the Rust source file to run rust_check on.",
    )
    parser.add_argument(
        "--direct-ninja",
        action="store_true",
        help="Run ninja directly instead of using soong_ui",
    )
    args = parser.parse_args()

    exe_path = Path(__file__).resolve()
    android_build_top = exe_path.parents[3]
    soong_env = load_soong_env_variables(
        android_build_top, "soong.environment.available"
    )

    if not soong_env.get("TARGET_PRODUCT"):
        raise Exception(
            "Lunch variables were not set. Please run lunch <target> && m nothing"
        )
    source_file = args.source_file.relative_to(android_build_top)
    check_targets = load_check_targets(android_build_top, source_file)

    if args.direct_ninja:
        result = execute_ninja(android_build_top, check_targets, soong_env)
    else:
        # https://cs.android.com/android/platform/superproject/main/+/main:build/soong/ui/build/soong.go;l=565-568
        soong_used_env = load_soong_env_variables(
            android_build_top, "soong.environment.used.build"
        )
        result = execute_soong_ui(android_build_top, check_targets, soong_used_env)

    # A negative return code indicates that the command was terminated by a signal.
    if result.returncode < 0:
        raise Exception(f"Command failed. Killed by signal {result.returncode}")

    # Since soong builds from AOSP_BUILD_TOP, the DiagnosticSpan.file_name field in the error/warning JSON
    # will be reported as relative to AOSP_BUILD_TOP. Relative paths are interpreted as relative to the
    # "rust-project.json" file [0]. We expect the developer to set SOONG_LINK_RUST_PROJECT_TO=${AOSP_BUILD_TOP}
    # to ensure that all these relative paths are correct. Therefore we simply check for duplicate messages and
    # then echo them out to stdout for rust-analyzer to process.
    #
    # [0]: https://rust-analyzer.github.io/manual.html#non-cargo-based-projects
    diagnostics = set()
    for check_target in check_targets:
        check_output = android_build_top / error_file(check_target)
        try:
            with open(check_output, "r") as file:
                for line in file:
                    diagnostics.add(line.strip())
        except FileNotFoundError:
            raise Exception(f"Failed to open {check_output}")
    for diagnostic in diagnostics:
        print(diagnostic)


if __name__ == "__main__":
    main()
