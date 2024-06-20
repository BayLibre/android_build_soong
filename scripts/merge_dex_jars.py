#!/usr/bin/env python3
import argparse
import copy
import os
import subprocess
import sys
import zipfile

def main():
    parser = argparse.ArgumentParser(
        description="Given a list of jars this will merge them using the normal "
        "naming convention for classes*.dex files."
    )
    parser.add_argument(
        "--output",
        help="Path to output jar "
    )
    parser.add_argument(
        "--stripFile",
        action="append",
        help="Path to output jar "
    )
    parser.add_argument(
        "--original_jar",
        help="Path to the original jar "
    )

    parser.add_argument("input_jars", nargs='+')
    args = parser.parse_args()

    current_index = 1
    with zipfile.ZipFile(args.output, 'w') as out_file:
        for zip in args.input_jars:
            with zipfile.ZipFile(zip, 'r') as input_file:
                for zipinfo in input_file.infolist():
                    name = zipinfo.filename
                    if name.startswith("classes") and name.endswith(".dex"):
                        dest_name = "classes.dex" if current_index==1 else "classes%s.dex" % current_index
                        current_index = current_index+1
                        zipinfo_out = copy.copy(zipinfo)
                        zipinfo_out.filename = dest_name
                        out_file.writestr(zipinfo_out, input_file.read(zipinfo))
                        continue
        with zipfile.ZipFile(args.original_jar, 'r') as input_file:
            for zipinfo in input_file.infolist():
                if not zipinfo.filename.endswith(".class"):
                    out_file.writestr(zipinfo, input_file.read(zipinfo))


if __name__ == "__main__":
    main()
