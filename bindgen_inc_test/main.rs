//! Example crate for testing bindgen bindings

fn main() {
    let mut x = buzz_bindgen::foo { x: 2 };
    // SAFETY: not production
    unsafe { buzz_bindgen::fizz(1, &mut x as *mut buzz_bindgen::foo) }
}
