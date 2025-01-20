use bindgen_test_rust::mul_three_numbers_in_c;

fn main() {
    println!("{} * {} * {} = {}", 3, 7, 11, mul_three_numbers_in_c(3, 7, 11));
}
