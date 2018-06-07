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
"""A tool for inserting values from the build system into a manifest."""

from __future__ import print_function
import argparse
from xml.dom import minidom


android_ns = 'http://schemas.android.com/apk/res/android'


def get_children_with_tag(parent, tag_name):
  children = []
  for child in  parent.childNodes:
    if child.nodeType == minidom.Node.ELEMENT_NODE and \
       child.tagName == tag_name:
      children.append(child)
  return children


def parse_args():
  """Parse commandline arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--minSdkVersion', default='', dest='min_sdk_version',
                      help='specify minSdkVersion used by the build system')
  parser.add_argument('input', help='input AndroidManifest.xml file')
  parser.add_argument('output', help='input AndroidManifest.xml file')
  return parser.parse_args()


def parse_manifest(doc):
  """Get the manifest element."""

  manifest = doc.documentElement
  if manifest.tagName != 'manifest':
    raise RuntimeError('expected manifest tag at root')
  return manifest


def as_int(s):
  try:
    i = int(s)
  except ValueError:
    return -1, False
  return i, True


def raise_min_sdk_version(doc, manifest, requested):
  """Ensure the manifest contains a <uses-sdk> tag with a minSdkVersion.

  Args:
    doc: The XML document.
    manifest: The <manifest> root tag.
    requested: The requested minSdkVersion attribute.
  Raises:
    RuntimeError: invalid manifest
  """

  # Get or insert the uses-sdk element
  uses_sdk = get_children_with_tag(manifest, 'uses-sdk')
  if len(uses_sdk) > 1:
    raise RuntimeError('found multiple uses-sdk elements')
  elif len(uses_sdk) == 1:
    element = uses_sdk[0]
  else:
    element = doc.createElement('uses-sdk')
    manifest.insertBefore(element, manifest.firstChild)
    # TODO(ccross): determine the indent?
    manifest.insertBefore(doc.createTextNode('\n    '), manifest.firstChild)

  # Get or insert the minSdkVersion attribute
  if element.hasAttributeNS(android_ns, 'minSdkVersion'):
    attr = element.getAttributeNodeNS(android_ns, 'minSdkVersion')
  else:
    attr = doc.createAttributeNS(android_ns, 'minSdkVersion')
    attr.value = '1'
    element.setAttributeNode(attr)

  manifest_int, manifest_is_int = as_int(attr.value)
  requested_int, requested_is_int = as_int(requested)

  # Update the value of the minSdkVersion attribute if necessary
  if manifest_is_int and requested_is_int:
    # Both manifest and requested values are codenames like "P":
    if requested.upper() > attr.value.upper():
      attr.value = requested.upper()
  elif not manifest_is_int and not requested_is_int:
    # Both manfiest and requested values are integer versions like "28":
    if requested_int > manifest_int:
      attr.value = str(requested_int)
  elif manifest_is_int and not requested_is_int:
    # Requested value is a codename, manifest is a number
    attr.value = requested.upper()
  else:
    # Requested value is a number, manifest is a codename
    pass


def main():
  """Program entry point."""
  args = parse_args()

  doc = minidom.parse(args.input)
  manifest = parse_manifest(doc)

  if args.min_sdk_version:
    raise_min_sdk_version(doc, manifest, args.min_sdk_version)

  with open(args.output, 'wb') as f:
    f.write('<?xml version="1.0" encoding="utf-8"?>\n')
    for node in doc.childNodes:
      f.write(node.toxml(encoding='utf-8') + '\n')


if __name__ == '__main__':
  main()
