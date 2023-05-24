use std::env;

fn main() {
    println!("rustenv environment:");

    for (key, val) in env::vars() {
        println!(" {key}: {val}");
    }
}
