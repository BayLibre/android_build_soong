#!/usr/bin/env python
#
# Copyright (C) 2021 The Android Open Source Project
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
"""A tool to report the current clang version used during build"""

import os
import re


THIS_DIR = os.path.dirname(__file__)

def get_clang_prebuilts_version():
  # TODO(b/187231324): Get clang version from the json file once it is no longer
  # hard-coded in global.go
  with open(THIS_DIR + '/../cc/config/global.go') as infile:
    contents = infile.read()

  regex_rev = r'\tClangDefaultVersion\s+= "clang-(?P<rev>r\d+[a-z]?\d?)"'
  match_rev = re.search(regex_rev, contents)
  if match_rev is None:
    raise RuntimeError('Parsing clang info failed')
  return match_rev.group('rev')


def main():
  print(get_clang_prebuilts_version());


if __name__ == '__main__':
  main()
