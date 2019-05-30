#!/usr/bin/env python
#
# Copyright (C) 2018 The Android Open Source Project
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
"""Decision logic on the runtime hidden API enforcement policy of
   a given build module. This is intended for use by dexpreopt when
   pre-compiling bytecode in the system image on host, so that the
   enforcement policy matches the one later selected at runtime.

   This must be kept in sync with ApplicationInfo.isAllowedToUseHiddenApis(),
   otherwise the pre-compiled oat files will be rejected at runtime."""

from __future__ import print_function

import argparse
import sys
import subprocess
import zipfile

def parse_args():
  """Parse commandline arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--aapt', required=True, action='store')
  parser.add_argument('--dex-location', required=True, action='store')
  parser.add_argument('--app', dest='is_app', default=False, action='store_true')
  parser.add_argument('--platform-signed',
                      dest='is_platform_signed',
                      default=False,
                      action='store_true')
  parser.add_argument('--uses-non-sdk-apis',
                      default=False,
                      action='store_true')
  parser.add_argument('input_apk', help='input APK file')
  return parser.parse_args()

def is_system_app(dex_location):
  """Returns true if `dex_location` corresponds to location of system apps.
     Keep in sync with assignment of PackageManagerService.SCAN_AS_SYSTEM."""
  return dex_location.startswith('/system/app/') or \
         dex_location.startswith('/system/priv-app/')

def uses_non_sdk_apis(input_apk, aapt, cmdline_uses_non_sdk_apis):
  """Return true if this APK uses non-SDK APIs. This can come from two sources:
     (a) build module configuration (LOCAL_PRIVATE_PLATFORM_APIS in Makefile or
         `platform_apis` in Soong), passed in as a command-line flag, or
     (b) from a `usesNonSdkApi` attribute on the <application> tag in the manifest.
     The runtime only looks at (b) but we need (a) because dexpreopt runs on an
     intermediate APK with no resources. Conversely, (a) is not sufficient here
     because prebuilts may have (b) without declaring it in the build module."""
  if cmdline_uses_non_sdk_apis:
    return True
  elif "AndroidManifest.xml" in zipfile.ZipFile(input_apk).namelist():
    aapt_output = subprocess.check_output(
        [ aapt, 'd', 'xmltree', input_apk, 'AndroidManifest.xml' ])
    aapt_output = list(map(str.strip, aapt_output.split('\n')))
    # TODO: check that this is inside <application>
    return 'A: android:usesNonSdkApi(0x0101058e)=(type 0x12)0xffffffff' in aapt_output
  else:
    return False

def get_hiddenapi_enforcement_policy(args):
  """Return the hidden API enforcement policy at runtime."""
  if args.is_app:
    if args.is_platform_signed:
      return "disabled"
    elif is_system_app(args.dex_location) and \
        uses_non_sdk_apis(args.input_apk, args.aapt, args.uses_non_sdk_apis):
      return "disabled"
  return "enabled"

def main(args):
  """Program entry point."""
  policy = get_hiddenapi_enforcement_policy(args)
  print(policy)

if __name__ == '__main__':
  main(parse_args())
