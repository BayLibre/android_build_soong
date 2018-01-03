#include <elf.h>
#include <err.h>
#include <fcntl.h>
#include <inttypes.h>
#include <stdio.h>

#include <type_traits>
#include <unordered_map>
#include <vector>

#include <android-base/file.h>
#include <android-base/logging.h>
#include <android-base/unique_fd.h>

using android::base::unique_fd;

class File {
 public:
  File(unique_fd fd) : fd_(std::move(fd)) {}

  std::unique_ptr<char[]> ReadBytes(size_t len, uint64_t offset) {
    // TODO: Support mmap?
    auto result = std::make_unique<char[]>(len);
    if (!android::base::ReadFullyAtOffset(fd_, result.get(), len, offset)) {
      return nullptr;
    }
    return result;
  }

  template<typename T>
  std::enable_if_t<std::is_pod_v<T>, std::unique_ptr<T[]>> Read(size_t count, uint64_t offset) {
    auto result = std::make_unique<T[]>(count);
    if (!android::base::ReadFullyAtOffset(fd_, result.get(), count * sizeof(T), offset)) {
      return nullptr;
    }
    return result;
  }

 private:
  unique_fd fd_;
};

using Replacements = std::unordered_map<std::string, std::string>;

int inject_elf(File& file, const Replacements& replacements);

template <typename EhdrType, typename ShdrType, typename SymType>
int inject_elf_impl(File& file,
                    const Replacements& replacements);

const auto inject_elf32 = inject_elf_impl<Elf32_Ehdr, Elf32_Shdr, Elf32_Sym>;
const auto inject_elf64 = inject_elf_impl<Elf64_Ehdr, Elf64_Shdr, Elf64_Sym>;

int main(int argc, const char** argv) {
  if (argc < 4 || argc % 2 != 0) {
    errx(1, "usage: symbol_inject <object file> [<symbol name> <symbol value>]...");
  }

  Replacements replacements;
  for (int i = 2; i < argc - 1; i += 2) {
    if (strlen(argv[i]) == 0) {
      errx(1, "received empty symbol name");
    }

    replacements[argv[i]] = argv[i + 1];
  }

  unique_fd fd(open(argv[1], O_RDONLY));
  if (fd == -1) {
    err(1, "failed to open file");
  }

  File file(std::move(fd));
  return inject_elf(file, replacements);
}

static bool check_elf_magic(const char* e_ident) {
  char magic[4] = { 0x7f, 'E', 'L', 'F' };
  return memcmp(e_ident, magic, 4) == 0;
}

int inject_elf(File& file, const Replacements& replacements) {
  auto elf_ident = file.ReadBytes(EI_NIDENT, 0);
  if (!elf_ident) {
    errx(1, "failed to read ELF ident header");
  }

  if (!check_elf_magic(elf_ident.get())) {
    errx(1, "target file is not an ELF file");
  }

  CHECK_EQ(1, elf_ident[EI_DATA]);
  CHECK_EQ(1, elf_ident[EI_VERSION]);

  if (elf_ident[EI_CLASS] == 1) {
    printf("elf32\n");
    return inject_elf32(file, replacements);
  } else if (elf_ident[EI_CLASS] == 2) {
    printf("elf64\n");
    return inject_elf64(file, replacements);
  } else {
    errx(1, "unknown EI_CLASS: %d", elf_ident[EI_CLASS]);
  }
}

template <typename EhdrType, typename ShdrType, typename SymType>
int inject_elf_impl(File& file, const Replacements& replacements) {
  std::unique_ptr<EhdrType[]> ehdr_storage = file.Read<EhdrType>(1, 0);
  if (!ehdr_storage) {
    errx(1, "failed to read full ELF header");
  }

  EhdrType* ehdr = ehdr_storage.get();

  if (!check_elf_magic(reinterpret_cast<char*>(ehdr->e_ident))) {
    errx(1, "target file is not an ELF file");
  }

  uint64_t section_headers_length = ehdr->e_shentsize * ehdr->e_shnum;
  if (static_cast<uint64_t>(SIZE_MAX) < section_headers_length) {
    errx(1, "section header size overflow");
  }

  auto section_header_storage = file.ReadBytes(section_headers_length, ehdr->e_shoff);
  if (!section_header_storage) {
    errx(1, "failed to read section headers");
  }

  std::vector<ShdrType*> section_headers;
  ShdrType* symbol_table_header = nullptr;
  for (uint64_t i = 0; i < ehdr->e_shnum; ++i) {
    uint64_t offset = i * ehdr->e_shentsize;
    ShdrType* section_header = reinterpret_cast<ShdrType*>(section_header_storage.get() + offset);
    section_headers.push_back(section_header);

    if (section_header->sh_type == SHT_SYMTAB) {
      if (symbol_table_header) {
        errx(1, "ELF file has multiple symbol tables");
      }

      symbol_table_header = section_header;
    }
  }

  ShdrType* string_table_header = section_headers[ehdr->e_shstrndx];
  if (static_cast<uint64_t>(SIZE_MAX) < string_table_header->sh_size) {
    errx(1, "string table size overflow");
  }

  uint64_t string_table_len = string_table_header->sh_size;
  auto string_table = file.ReadBytes(string_table_len, string_table_header->sh_offset);
  if (!string_table) {
    errx(1, "failed to read string table");
  }

  CHECK_EQ(sizeof(SymType), symbol_table_header->sh_entsize);
  uint64_t symbol_count = symbol_table_header->sh_size / sizeof(SymType);
  CHECK_EQ(0ULL, symbol_table_header->sh_size % sizeof(SymType));
  auto symbol_table = file.Read<SymType>(symbol_count, symbol_table_header->sh_offset);
  for (uint64_t i = 0; i < symbol_count; ++i) {
    SymType* symbol = &symbol_table[i];
    if (symbol->st_name < 0 || symbol->st_name > string_table_len) {
      errx(1, "symbol name index overflows string table");
    }

    const char* symbol_name = &string_table[symbol->st_name];
    auto it = replacements.find(symbol_name);
    if (it != replacements.end()) {
      ShdrType* section = section_headers.at(symbol->st_shndx);
      uint64_t symbol_begin = section->sh_offset + symbol->st_value;
      uint64_t symbol_end = symbol_begin + symbol->st_size;

      printf("TODO: replace symbol %s at offsets [%" PRIu64 "-%" PRIu64 ") of file with '%s'\n",
             symbol_name, symbol_begin, symbol_end, it->second.c_str());
    }
  }

  return 0;
}
