"""Bloaty CSV Merger

Merges a list of .csv files from Bloaty into a protobuf.  It takes the list as
a first argument and the output as second. For instance:

    $ bloaty_merger binary_sizes.lst binary_sizes.pb

"""

import argparse
import csv

import file_sections_pb2

BLOATY_EXTENSION = ".bloaty.csv"

def parse_csv(path):
  """Parses a Bloaty-generated CSV file into a protobuf.

  Args:
    path: The filepath to the CSV file, relative to $ANDROID_TOP.

  Returns:
    A file_sections_pb2.File if the file was found; None otherwise.
  """
  file_proto = None
  with open(path, newline='') as csv_file:
    file_proto = file_sections_pb2.File()
    if path.endswith(BLOATY_EXTENSION):
      file_proto.path = path[:-len(BLOATY_EXTENSION)]
    section_reader = csv.DictReader(csv_file)
    for row in section_reader:
      section = file_proto.sections.add()
      section.name = row["sections"]
      section.vm_size = int(row["vmsize"])
      section.file_size = int(row["filesize"])
  return file_proto

def create_file_size_metrics(input_list, output_proto):
  """Creates a FileSizeMetrics proto from a list of CSV files.

  Args:
    input_list: The path to the file which contains the list of CSV files, one
        file per line.
    output_proto: The path for the output protobuf.
  """
  metrics = file_sections_pb2.FileSizeMetrics()
  with open(input_list) as inputs:
    for csv_path in inputs.readlines():
      file_proto = parse_csv(csv_path.strip())
      if file_proto:
        metrics.files.append(file_proto)
  with open(output_proto, "wb") as output:
    output.write(metrics.SerializeToString())

def main():
  parser = argparse.ArgumentParser()
  parser.add_argument("input_list_file", help="List of bloaty csv files, one filepath per line.")
  parser.add_argument("output_proto", help="Output proto.")
  args = parser.parse_args()
  create_file_size_metrics(args.input_list_file, args.output_proto)

if __name__ == '__main__':
  main()
