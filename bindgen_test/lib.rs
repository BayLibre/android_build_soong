pub fn mul_three_numbers_in_c(a: u32, b: u32, c: u32) -> u32 {
    // SAFETY: shut up -D clippy::undocumented-unsafe-blocks
    unsafe { bindgen_test_bindings::mul_three_numbers(a, b, c) }
}
