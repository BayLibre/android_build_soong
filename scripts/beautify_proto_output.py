#!/usr/bin/env python3
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
"""A tool to print human-readable information taken from proto parsed output."""

import os
import sys
from .compat import json
#import soong_build_metrics.metrics_pb2.py


def main():
  json_file = sys.argv[1] if len(sys.argv) > 1 else '../../../out/beautify_proto.txt'
  with open(json_file) as f:
    proto = json.load(f)
    print(proto)


if __name__ == '__main__':
  main()

