#include "symbol_inject.h"

#include <elf.h>
#include <err.h>
#include <fcntl.h>
#include <inttypes.h>
#include <stdio.h>

#include <deque>
#include <string_view>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include <android-base/unique_fd.h>

__attribute__((noreturn)) void usage(bool err) {
  FILE* output = err ? stderr : stdout;
  fprintf(output, "usage: symbol_inject inject <file> [-o <output>] [<symbol name> <value>]...\n");
  fprintf(output, "       symbol_inject find <file> [<symbol name>]...\n");
  exit(err);
}

std::unordered_map<std::string_view, SymbolLocation> find_symbols(
    File& file, const std::unordered_set<std::string_view>& symbol_names) {
  // TODO: Support Mach-O as well...
  return find_symbols_elf(file, symbol_names);
}

int main(int argc, const char** argv) {
  if (argc <= 1) {
    usage(true);
  }

  std::deque<std::string_view> args(argv + 1, argv + argc);
  if (args[0] == "help" || args[0] == "--help") {
    usage(false);
  }

  bool inject = args[0] == "inject";
  if (args[0] != "inject" && args[0] != "find") {
    fprintf(stderr, "symbol_inject: unknown command '%s'\n", args[0].data());
    usage(true);
  }
  args.pop_front();

  if (args.empty()) {
    usage(true);
  }

  std::string_view input_file = args.front();
  args.pop_front();

  std::unordered_set<std::string_view> symbol_names;
  std::unordered_map<std::string_view, std::string_view> symbol_replacements;
  std::string_view output = "a.out";

  while (!args.empty()) {
    std::string_view first = args.front();
    args.pop_front();

    if (inject) {
      if (args.empty()) {
        usage(true);
      }

      bool is_output = first == "-o";

      std::string_view second = args.front();
      args.pop_front();

      if (second.empty()) {
        if (is_output) {
          fprintf(stderr, "symbol_inject: invalid empty output file\n");
        } else {
          fprintf(stderr, "symbol_inject: invalid empty symbol name\n");
        }
        usage(true);
      }

      if (first == "-o") {
        output = second;
      } else {
        symbol_names.insert(first);
        symbol_replacements[first] = second;
      }
    } else {
      symbol_names.insert(first);
    }
  }

  if (symbol_names.empty()) {
    errx(1, "nothing to do");
  }

  android::base::unique_fd fd(open(input_file.data(), O_RDONLY));
  if (fd == -1) {
    err(1, "failed to open input file %s", input_file.data());
  }

  File file(std::move(fd));
  auto symbol_locations = find_symbols(file, symbol_names);
  if (symbol_locations.size() > symbol_names.size()) {
    errx(1, "found more symbols than requested?");
  } else if (symbol_locations.size() != symbol_names.size()) {
    fprintf(stderr, "symbol_inject: failed to find all requested symbols.\n");
    fprintf(stderr, "missing symbols:\n");
    for (const auto& symbol_name : symbol_names) {
      if (symbol_locations.count(symbol_name) != 1) {
        fprintf(stderr, "  %s\n", symbol_name.data());
      }
    }
    exit(1);
  }

  if (!inject) {
    for (const auto& [symbol_name, symbol_location] : symbol_locations) {
      printf("Symbol name\t\t\t\t\tFile offset\tSize\n");
      printf("%-47s %-11" PRIu64 "\t%" PRIu64 "\n", symbol_name.data(), symbol_location.offset,
             symbol_location.size);
    }
  } else {
    // Make sure that all of our replacements fit.
    for (const auto& [symbol_name, symbol_location] : symbol_locations) {
      auto it = symbol_replacements.find(symbol_name);
      if (it == symbol_replacements.end()) {
        errx(1, "failed to find symbol '%s'", symbol_name.data());
      }

      std::string_view replacement = it->second;

      // string_view::size doesn't include the null terminator.
      if (replacement.size() + 1 > symbol_location.size) {
        errx(1, "replacement for symbol '%s' too large (max len: %" PRIu64 " including terminator)",
             symbol_name.data(), symbol_location.size);
      }
    }

    android::base::unique_fd output_fd;
    if (output == "-") {
      output_fd.reset(dup(STDOUT_FILENO));
    } else {
      output_fd.reset(open(output.data(), O_RDWR | O_CREAT | O_TRUNC, 0777));
      if (output_fd == -1) {
        err(1, "failed to open output file '%s'", output.data());
      }
    }

    File output_file(std::move(output_fd));
    output_file.CopyFrom(file);
    for (const auto& [symbol_name, symbol_location] : symbol_locations) {
      std::string_view replacement = symbol_replacements[symbol_name];
      if (!output_file.ZeroAndWrite(replacement, symbol_location.offset, symbol_location.size)) {
        err(1, "failed to write to output file");
      }
    }
  }

  return 0;
}
