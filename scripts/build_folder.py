#!/usr/bin/env python3
import argparse
import shutil
import subprocess
import sys
import os

def main():
    parser = argparse.ArgumentParser(fromfile_prefix_chars='@')
    # argparse normally only accepts 1 argument per line, but ninja can't write multiple lines
    # to the rspfile, so split the lines by whitespace. This means you can't use filepaths with
    # spaces.
    parser.convert_arg_line_to_args = lambda line: line.split()
    parser.add_argument("--stamp-file")
    parser.add_argument("dst_dir")
    parser.add_argument("--src_files", nargs="+", default=[])
    parser.add_argument("--dst_paths", nargs="+", default=[])
    args = parser.parse_args()

    if len(args.src_files) != len(args.dst_paths):
        sys.exit("Must provide a dst location for every src file.")

    if os.path.exists(args.dst_dir):
        shutil.rmtree(args.dst_dir)

    for src, dst in zip(args.src_files, args.dst_paths):
        if os.path.normpath(dst) != dst:
            sys.exit("destination paths must be normalized: " + dst)
        if os.path.relpath(dst, args.dst_dir).startswith("../"):
            sys.exit("destination paths must be under dst_dir")

        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.copy2(src, dst, follow_symlinks=False)

    if args.stamp_file:
        subprocess.check_call(["touch", args.stamp_file])

if __name__ == "__main__":
    main()
