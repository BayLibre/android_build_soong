#!/usr/bin/env python
#
# Copyright (C) 2016 The Android Open Source Project
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
"""Generates xml of NDK libraries used for API coverage analysis."""
import argparse
import json
import os

from xml.etree.ElementTree import Element, SubElement, tostring
from gen_stub_libs import ALL_ARCHITECTURES, ParseError, current_line_validation, decode_api_level_tags, get_tags


ROOT_ELEMENT_TAG = 'ndk-library'
SYMBOL_ELEMENT_TAG = 'symbol'
ARCHITECTURE_ATTRIBUTE_KEY = 'arch'
DEPRECATED_ATTRIBUTE_KEY = 'is_deprecated'
PLATFORM_ATTRIBUTE_KEY = 'is_platform'
NAME_ATTRIBUTE_KEY = 'name'
VARIABLE_ATTRIBUTE_KEY = 'is_var'
EXPOSED_TARGET_TAGS = (
    'vndk',
    'apex',
    'llndk',
)
API_LEVEL_TAG_PREFIXES = (
    'introduced=',
    'introduced-',
)


class ApiCoverageSymbolFileParser(object):
    """Parses NDK symbol files for NDK api code coverage use."""

    def __init__(self, input_file, api_map):
        self.input_file = input_file
        self.current_line = None
        self.api_map = api_map

    def parse(self):
        """Parses the symbol file, convert to xml object and return."""
        root = Element(ROOT_ELEMENT_TAG)
        while self.next_line() != '':
            if '{' in self.current_line:
                self.parse_version(root)
            else:
                raise ParseError(
                    'Expected "{" at top level, but got: ' + self.current_line)
        return root

    def parse_version(self, root):
        """Parses a single version section and returns a xml object."""
        version_name = self.current_line.split('{')[0].strip()
        tags = decode_api_level_tags(get_tags(self.current_line), self.api_map)
        version_attributes = parse_tags(tags)
        # Parse version_name and update version_attributes if keywords appear.
        _, _, postfix = version_name.partition('_')
        is_platform = postfix == 'PRIVATE' or postfix == 'PLATFORM'
        is_deprecated = postfix == 'DEPRECATED'
        version_attributes.update({PLATFORM_ATTRIBUTE_KEY: str(is_platform)})
        version_attributes.update({DEPRECATED_ATTRIBUTE_KEY: str(is_deprecated)})

        global_scope = True
        cpp_symbols = False
        is_var_block = VARIABLE_ATTRIBUTE_KEY in version_attributes

        while self.next_line() != '':
            if '}' in self.current_line:
                # Ignoring base here, as it doesn't have much influence to api code coverage result.
                if not self.current_line.split('#')[0].strip().endswith(';'):
                    raise ParseError(
                        'Unterminated version/export "C++" block (expected ;).: ' + self.current_line)
                if cpp_symbols:
                    cpp_symbols = False
                else:
                    return
            if 'extern "C++" {' in self.current_line:
                cpp_symbols = True
            elif not cpp_symbols and ':' in self.current_line:
                visibility = self.current_line.split(':')[0].strip()
                # Ignoring all local functions.
                if visibility == 'local':
                    global_scope = False
                elif visibility == 'global':
                    continue
                else:
                    raise ParseError('Unknown visibility label: ' + visibility)
            elif global_scope and not cpp_symbols and not is_var_block:
                self.parse_api(root, version_attributes)
            else:
                # We're in 'extern "C++"' block. Ignore everything.
                pass
        raise ParseError('Unexpected EOF in version block. Expecting to meet "}" and return.')

    def parse_api(self, root, version_attributes):
        """Parses a single symbol line and extend parsed xml object to root."""
        current_line_validation(self.current_line)
        name, _, _ = self.current_line.strip().partition(';')
        tags = decode_api_level_tags(get_tags(self.current_line), self.api_map)
        attributes = {NAME_ATTRIBUTE_KEY: name}
        attributes.update(version_attributes)
        # If same version tags already exist, it will be overwrite here.
        tmp_attributes = parse_tags(tags)
        if VARIABLE_ATTRIBUTE_KEY not in tmp_attributes:
            attributes.update(tmp_attributes)
            SubElement(root, SYMBOL_ELEMENT_TAG, attributes)

    def next_line(self):
        """Returns the next non-empty non-comment line.

        A return value of '' indicates EOF.
        """
        line = self.input_file.readline()
        while line.strip() == '' or line.strip().startswith('#'):
            line = self.input_file.readline()
            if line == '':
                break
        self.current_line = line
        return self.current_line


def parse_tags(tags):
    """Parses tags and save needed tags in the created attributes.

    Return attributes dictionary.
    """
    attributes = {}
    if 'var' in tags:
        attributes.update({VARIABLE_ATTRIBUTE_KEY: 'True'})

    arch = []
    for tag in tags:
        if tag.startswith(tuple(API_LEVEL_TAG_PREFIXES)):
            key, _, value = tag.partition('=')
            attributes.update({key: value})
        elif tag in ALL_ARCHITECTURES:
            arch.append(tag)
        elif tag in EXPOSED_TARGET_TAGS:
            attributes.update({tag: 'True'})
    attributes.update({ARCHITECTURE_ATTRIBUTE_KEY: ','.join(arch)})
    return attributes


def parse_args():
    """Parses and returns command line arguments."""
    parser = argparse.ArgumentParser()

    parser.add_argument('symbol_file', type=os.path.realpath, help='Path to symbol file.')
    parser.add_argument(
        'output_file', type=os.path.realpath,
        help='The output parsed api coverage file.')
    parser.add_argument(
        '--api-map', type=os.path.realpath, required=True,
        help='Path to the API level map JSON file.')
    return parser.parse_args()


def write_xml_to_file(root, output_file):
    """Write xml element root to output_file."""
    parsed_data = tostring(root)
    output_file = open(output_file, "w")
    output_file.write(parsed_data)


def main():
    """Program entry point."""
    args = parse_args()

    with open(args.api_map) as map_file:
        api_map = json.load(map_file)

    with open(args.symbol_file) as symbol_file:
        xml = ApiCoverageSymbolFileParser(symbol_file, api_map).parse()
        write_xml_to_file(xml, args.output_file)


if __name__ == '__main__':
    main()
