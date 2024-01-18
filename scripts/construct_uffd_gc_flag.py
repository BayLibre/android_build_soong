#!/usr/bin/env python
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
"""A tool for constructing UFFD GC flag."""

from __future__ import print_function

import argparse

from uffd_gc_utils import should_enable_uffd_gc


def parse_args():
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--kernel-version-file',
      required=True,
      dest='kernel_version_file',
  )
  return parser.parse_args()

def main():
  args = parse_args()
  print("--runtime-arg -Xgc:CMC"
        if should_enable_uffd_gc(args.kernel_version_file)
        else "",
        end="")


if __name__ == '__main__':
  main()
