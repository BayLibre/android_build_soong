#!/usr/bin/env python
#
# Copyright 2021 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

""" `fsverity_sign` signs a file to be consumed by authfs

This actually is a simple wrapper around the `fsverity` program. A file is
signed by the program which produces the PKCS#7 signature file, merkle tree file
, and the fsverity_descriptor file. Then the files are packed into a single
output file so that the information about the signing stays together.

Currently, the output of this script is used by `fd_server` which is the host-
side backend of an authfs filesystem. `fd_server` uses this file in case when
the underlying filesystem (ext4, etc.) on the device doesn't support the
fsverity feature natively in which case the information is read directly from
the filesystem using ioctl.
"""

import argparse
import os
import re
import shutil
import subprocess
import sys
import tempfile
from struct import *

def parse_args(argv):
  p = argparse.ArgumentParser()
  p.add_argument(
      '--output',
      help='output file. If omitted, print to <INPUT>.fsverity_metadata',
      metavar='output')
  p.add_argument(
      'input',
      help='input file to be signed')
  p.add_argument(
      '--key',
      help='PKCS#8 private key file in PEM format',
      required=True)
  p.add_argument(
      '--cert',
      help='x509 certificate file in PEM format',
      required=True)
  p.add_argument(
      '--hash-alg',
      help='hash algorithm to use to build the merkle tree',
      choices=['sha256', 'sha512'],
      default='sha256')
  p.add_argument(
      '--block-size',
      help='merkle tree block size (in bytes) to use',
      type=int,
      default=4096)
  p.add_argument(
      '--raw_signature',
      help='raw signature is stored instead of PKCS#7 format',
      action='store_true')
  p.add_argument(
      '--fsverity-path',
      help='path to the fsverity program',
      required=True)
  return p.parse_args(argv)


def do_sign(args, work_dir):
  input_file = args.input
  output_file = (
      input_file + '.fsverity_metadata' if args.output is None
      else args.output)

  # temporary files
  desc_file = os.path.join(work_dir, 'desc')
  merkletree_file = os.path.join(work_dir, 'merkletree')
  sig_file = os.path.join(work_dir, 'signature')

  # run the fsverity util to create the temporary files
  cmd = [args.fsverity_path, 'sign']
  cmd.append(args.input)
  cmd.append(sig_file)
  cmd.extend(['--key', args.key])
  cmd.extend(['--cert', args.cert])
  cmd.extend(['--hash-alg', args.hash_alg])
  cmd.extend(['--block-size', str(args.block_size)])
  cmd.extend(['--out-merkle-tree', merkletree_file])
  cmd.extend(['--out-descriptor', desc_file])
  subprocess.run(cmd).check_returncode()

  with open(output_file, 'wb') as out:
    # 1. version
    out.write(pack('<I', 1))

    # 2. fsverity_descriptor
    with open(desc_file, 'rb') as f:
      out.write(f.read())

    # 3. signature
    SIG_TYPE_PKCS7 = 1
    SIG_TYPE_RAW = 2
    if args.raw_signature:
      out.write(pack('<I', SIG_TYPE_RAW))
      sig = raw_signature(sig_file)
      out.write(pack('<I', len(sig)))
      out.write(sig)
    else:
      with open(sig_file, 'rb') as f:
        out.write(pack('<I', SIG_TYPE_PKCS7))
        sig = f.read()
        out.write(pack('<I', len(sig)))
        out.write(sig)

    # 4. merkle tree
    with open(merkletree_file, 'rb') as f:
      # merkle tree is placed at the next nearest page boundary to make
      # mmapping possible
      out.seek(next_page(out.tell()))
      out.write(f.read())


def next_page(n):
  """ Returns the next nearest page boundary from `n` """
  PAGE_SIZE = 4096
  return int((n + PAGE_SIZE - 1) / PAGE_SIZE) * PAGE_SIZE


def raw_signature(pkcs7_sig_file):
  """ Extracts raw signature from DER formatted PKCS#7 detached signature file

  Do that by parsing the ASN.1 tree to get the location of the signature
  in the file and then read the portion.
  """

  # Note: there seems to be no public python API (even in 3p modules) that
  # provides direct access to the raw signature at this moment. So, `openssl
  # asn1parse` commandline tool is used instead.
  cmd = ['openssl', 'asn1parse']
  cmd.extend(['-inform', 'DER'])
  cmd.extend(['-in', pkcs7_sig_file])
  out = subprocess.run(cmd, capture_output=True, check=True).stdout

  # The signature is the last element in the tree
  last_line = out.decode('utf-8').splitlines()[-1]
  m = re.search('(\d+):.*hl=\s*(\d+)\s*l=\s*(\d+)\s*.*OCTET STRING', last_line)
  offset = int(m.group(1))
  header_len = int(m.group(2))
  size = int(m.group(3))
  with open(pkcs7_sig_file, 'rb') as f:
    f.seek(offset + header_len)
    return f.read(size)


class TempDirectory(object):

  def __enter__(self):
    self.name = tempfile.mkdtemp()
    return self.name

  def __exit__(self, *unused):
    shutil.rmtree(self.name)


if __name__ == '__main__':
  with TempDirectory() as work_dir:
    args = parse_args(sys.argv[1:])
    do_sign(args, work_dir)
