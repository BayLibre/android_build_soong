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
"""Generates source for stub shared libraries for the NDK."""
import argparse
import json
import os

from xml.etree.ElementTree import Element, SubElement, tostring
from gen_stub_libs import ParseError, get_tags, current_line_validation, decode_api_level_tags


class ApiCoverageSymbolFileParser(object):
    """Parses NDK symbol files for NDK api code coverage use."""

    def __init__(self, input_file, api_map):
        self.input_file = input_file
        self.current_line = None
        self.api_map = api_map

    def parse(self):
        """Parses the symbol file, convert to xml object then write to output_file."""
        root = Element('ndk-library')
        while self.next_line() != '':
            if '{' in self.current_line:
                self.parse_version(root)
            else:
                raise ParseError(
                    'Unexpected contents at top level: ' + self.current_line)
        return root

    def parse_version(self, root):
        """Parses a single version section and returns a xml object."""
        version_name = self.current_line.split('{')[0].strip()
        version_attributes = {}
        tags = get_tags(self.current_line)
        tags = decode_api_level_tags(tags, self.api_map)
        global_scope = parse_tags(tags, version_attributes)

        while self.next_line() != '':
            if '}' in self.current_line:
                # Ignoring base here, as it doesn't have much influence to api code coverage result.
                if not self.current_line.strip().endswith(';'):
                    raise ParseError(
                        'Unterminated version/export "C++" block (expected ;).: ' + self.current_line)
                return
            if ':' in self.current_line:
                visibility = self.current_line.split(':')[0].strip()
                # Ignoring all local functions.
                if visibility == 'local':
                    global_scope = False
                elif visibility == 'global':
                    continue
                else:
                    raise ParseError('Unknown visibility label: ' + visibility)
            elif global_scope:
                _, _, postfix = version_name.partition('_')
                is_platform = postfix == 'PRIVATE' or postfix == 'PLATFORM'
                is_deprecated = postfix == 'DEPRECATED'
                version_attributes.update(is_platform=str(is_platform))
                version_attributes.update(is_deprecated=str(is_deprecated))
                self.parse_api(root, version_attributes)

    def parse_api(self, root, version_attributes):
        """Parses a single symbol line and extend parsed xml object to root."""
        current_line_validation(self.current_line)
        name, _, _ = self.current_line.strip().partition(';')
        tags = get_tags(self.current_line)
        tags = decode_api_level_tags(tags, self.api_map)

        attributes = {'name': name}
        attributes.update(version_attributes)
        # If same version tags already exist, it will be overwrite here.
        if parse_tags(tags, attributes):
            SubElement(root, 'symbol', attributes)

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


def parse_tags(tags, attributes):
    """Parses tags and save needed tags in attributes.

    Return False if 'var' presents. Otherwise return True.
    """
    if 'var' in tags:
        return False

    arch = []
    for tag in tags:
        if tag.startswith('introduced=') or tag.startswith('introduced-'):
            key, _, value = tag.partition('=')
            attributes.update({key: value})
        elif tag.startswith('arm') or tag.startswith('x86'):
            arch.append(tag)
        elif tag == 'vndk' or tag == 'apex' or tag == 'llndk':
            attributes.update({tag: 'True'})
    attributes.update({'arch': ','.join(arch)})
    return True


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
        print("********symbol_file name: ", symbol_file.name)
        xml = ApiCoverageSymbolFileParser(symbol_file, api_map).parse()
        write_xml_to_file(xml, args.output_file)


if __name__ == '__main__':
    main()
