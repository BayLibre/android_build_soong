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

import classpaths_pb2

import google.protobuf.json_format as json_format


def json2binary(args):
    pb = classpaths_pb2.ExportedClasspathsJars()
    json_format.Parse(args.input.read(), pb)
    args.output.write(pb.SerializeToString())
    args.input.close()
    args.output.close()


def binary2json(args):
    pb = classpaths_pb2.ExportedClasspathsJars()
    pb.ParseFromString(args.input.read())
    print(json_format.MessageToJson(pb))
    args.input.close()


def main():
    parser = argparse.ArgumentParser("Utility to convert classpaths.proto messages between binary "
                                     "and JSON formats.")
    subparsers = parser.add_subparsers()

    parser_json2binary = subparsers.add_parser('proto',
                                               help='convert classpaths protobuf message from '
                                                    'JSON to binary format')
    parser_json2binary.add_argument('input',
                                    help='classpaths protobuf message in JSON format',
                                    nargs='?', type=argparse.FileType('r'), default=sys.stdin)
    parser_json2binary.add_argument('output',
                                    help='output file to write converted message in binary format',
                                    nargs='?', type=argparse.FileType('wb'),
                                    default=sys.stdout.buffer)
    parser_json2binary.set_defaults(func=json2binary)

    parser_binary2json = subparsers.add_parser('print',
                                               help='print classpaths config in JSON format')
    parser_binary2json.add_argument('input',
                                    help='classpaths config file in protobuf binary format',
                                    nargs='?', type=argparse.FileType('rb'), default=sys.stdin)
    parser_binary2json.set_defaults(func=binary2json)

    args = parser.parse_args()
    args.func(args)


if __name__ == '__main__':
    main()
