#!/usr/bin/env python3
#
# Copyright (C) 2024 The Android Open Source Project
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
#

"""
This script is used as a replacement for the Rust linker to allow fine-grained
control over the what gets emitted to the linker.
"""

import os
import shutil
import subprocess
import sys
from get_clang_version import get_clang_prebuilts_version

ANDROID_BUILD_TOP = os.environ.get("ANDROID_BUILD_TOP", ".")

replacementVersionScript = None

clang_args = sys.argv
clang_args[0] = ANDROID_BUILD_TOP + "/prebuilts/clang/host/linux-x86/" + get_clang_prebuilts_version(None) + "/bin/clang++"

for i, arg in enumerate(clang_args):
   if arg.startswith('-Wl,--android-version-script='):
      replacementVersionScript = arg.split("=")[1]
      del clang_args[i]
      break

if replacementVersionScript:
   versionScriptFound = False
   for i, arg in enumerate(clang_args):
      if arg.startswith('-Wl,--version-script='):
         clang_args[i] ='-Wl,--version-script=' + replacementVersionScript
         versionScriptFound = True
         break

   if not versionScriptFound:
       # If rustc did not emit a version script, just append the arg
       clang_args.append('-Wl,--version-script=' + replacementVersionScript)
try:
   subprocess.run(clang_args, encoding='utf-8', check=True)
except subprocess.CalledProcessError as e:
   sys.exit(-1)

