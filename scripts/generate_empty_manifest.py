#  Copyright (C) 2021 The Android Open Source Project
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

import argparse
import json
import sys


# Fetches the package name from the json file.
def fetch_package_name(input_file):
  input = open(input_file, "r")
  json_data = json.loads(input.read())
  input.close()

  return json_data['name']


# Generates the empty manifest file with the package name fetched from the input file.
def generate_empty_manifest(input_file, output_file):
  empty_manifest = "<?xml version='1.0' encoding='utf-8'?>\n" + \
                   "<manifest xmlns:android='http://schemas.android.com/apk/res/android' package='" + \
                   fetch_package_name(input_file) + "'>\n" + "</manifest>\n"

  output = open(output_file, "w")
  output.write(empty_manifest)
  output.close()


def main():
  parser = argparse.ArgumentParser(
      'Generate empty android manifest file with package name fetched from input file.')
  parser.add_argument('-i', '--input',
                      nargs='?', type=argparse.FileType('rb'),
                      default=sys.stdin.buffer)
  parser.add_argument('-o', '--output',
                      nargs='?', type=argparse.FileType('wb'),
                      default=sys.stdout.buffer)

  args = parser.parse_args()
  generate_empty_manifest(args.input.name, args.output.name)


if __name__ == '__main__':
  main()
