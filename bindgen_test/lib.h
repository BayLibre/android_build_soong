#ifndef BUILD_RUST_TESTS_BINDGEN_STATIC_FNS_TEST_LIB_H_
#define BUILD_RUST_TESTS_BINDGEN_STATIC_FNS_TEST_LIB_H_

#include <stdint.h>

uint32_t mul_two_numbers(uint32_t a, uint32_t b);

static inline uint32_t mul_three_numbers(uint32_t a,
                                                          uint32_t b,
                                                          uint32_t c) {
  return mul_two_numbers(mul_two_numbers(a, b), c);
}

#endif  //  BUILD_RUST_TESTS_BINDGEN_STATIC_FNS_TEST_LIB_H_
