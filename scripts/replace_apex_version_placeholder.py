#!/usr/bin/env python3
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

import argparse
import collections
import sys

def main():
  parser = argparse.ArgumentParser()
  parser.add_argument('manifest',
                      help='Path to the apex_manifest.json file, to pull the apex version from.')
  parser.add_argument('input',
                      help='Input file to copy to the output file, while replacing the placeholder text')
  parser.add_argument('output',
                      help='Output file')
  parser.add_argument('--placeholder-text',
                      default='__APEX_VERSION_PLACEHOLDER__',
                      help='The string to replace in the input file with the apex version.')
  args = parser.parse_args()

  with open(args.manifest, 'r') as f:
    obj = json.load(f, object_pairs_hook=collections.OrderedDict)

  if 'version' not in obj:
    sys.exit(f'"version" attribute not found in {args.manifest}')

  with open(args.input, 'r') as i:
    with open(args.output, 'w') as o:
      o.write(i.read().replace(args.placeholder_text, obj['version']))


if __name__ == '__main__':
  main()
