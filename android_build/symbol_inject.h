#pragma once

#include <err.h>
#include <stdint.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#include <string_view>
#include <type_traits>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include <android-base/file.h>
#include <android-base/unique_fd.h>

class File {
 public:
  File(android::base::unique_fd fd) : fd_(std::move(fd)) {}

  std::unique_ptr<char[]> ReadBytes(size_t len, uint64_t offset) {
    // TODO: Support mmap?
    auto result = std::make_unique<char[]>(len);
    if (!android::base::ReadFullyAtOffset(fd_.get(), result.get(), len, offset)) {
      return nullptr;
    }
    return result;
  }

  template <typename T>
  std::enable_if_t<std::is_pod_v<T>, std::unique_ptr<T[]>> Read(size_t count, uint64_t offset) {
    auto result = std::make_unique<T[]>(count);
    if (!android::base::ReadFullyAtOffset(fd_.get(), result.get(), count * sizeof(T), offset)) {
      return nullptr;
    }
    return result;
  }

  uint64_t Length() const {
    struct stat st;
    if (fstat(fd_.get(), &st) == -1) {
      err(1, "stat failed");
    }

    return st.st_size;
  }

  bool CopyFrom(File& other) {
    // TODO: copy_file_range?
    uint64_t length = other.Length();
    if (ftruncate(fd_.get(), length) == -1) {
      err(1, "failed to resize file");
    }

    constexpr size_t buffer_size = 16384;
    char buf[buffer_size];
    for (uint64_t offset = 0; offset < length; offset += buffer_size) {
      uint64_t bytes_left = length - offset;
      uint64_t len = std::min(bytes_left, buffer_size);
      if (!android::base::ReadFullyAtOffset(other.fd_.get(), buf, len, offset)) {
        return false;
      }
      if (!android::base::WriteFullyAtOffset(fd_.get(), buf, len, offset)) {
        return false;
      }
    }

    return true;
  }

  bool ZeroAndWrite(const std::string_view& data, uint64_t offset, size_t size) {
    if (size < data.size()) {
      return false;
    }

    if (!android::base::WriteFullyAtOffset(fd_.get(), data.begin(), data.size(), offset)) {
      return false;
    }

    char buf[4096] = {0};
    size_t bytes_left = size - data.size();
    while (bytes_left > 0) {
      uint64_t new_offset = offset + (size - bytes_left);
      size_t bytes_to_write = std::min(bytes_left, sizeof(buf));
      if (!android::base::WriteFullyAtOffset(fd_.get(), buf, bytes_to_write, new_offset)) {
        return false;
      }
      bytes_left -= bytes_to_write;
    }

    return true;
  }

 private:
  android::base::unique_fd fd_;
};

struct SymbolLocation {
  uint64_t offset;
  uint64_t size;
};

std::unordered_map<std::string_view, SymbolLocation> find_symbols_elf(
    File& file, const std::unordered_set<std::string_view>& symbol_names);
