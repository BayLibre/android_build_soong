#!/usr/bin/env python
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
#
"""Unit tests for generate_empty_manifest.py."""

import sys
import unittest
import xml.etree.ElementTree as ET
import json

import generate_empty_manifest

sys.dont_write_bytecode = True


class EmptyManifestGenerationTest(unittest.TestCase):
  """Unit tests for generate_empty_manifest function."""

  def assert_xml_equal(self, output, expected):
    self.assertEqual(ET.canonicalize(output), ET.canonicalize(expected))

  input_json = {
      "name": "com.android.sdkext",
      "version": 2147483647
  }

  empty_manifest = (
      '<?xml version="1.0" encoding="utf-8"?>\n'
      '<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="%s">\n'
      '</manifest>\n')

  input_file_name = 'input.json'
  output_file_name = 'output.xml'

  def test_empty_manifest_generation(self):
    input_file = open(self.input_file_name, 'w')
    input_file.write(json.dumps(self.input_json))
    input_file.close()

    expected = self.empty_manifest % self.input_json['name']
    output_file = open(self.output_file_name, 'w')
    output_file.close()

    generate_empty_manifest.generate_empty_manifest(self.input_file_name,
                                                    self.output_file_name)
    output_file = open(self.output_file_name, 'r')
    output = output_file.read()
    output_file.close()

    self.assertEqual(ET.canonicalize(output), ET.canonicalize(expected))


if __name__ == '__main__':
  unittest.main(verbosity=2)
