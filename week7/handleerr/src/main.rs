use std::fs::File;

fn main() {
    let f: Result<File, std::io::Error> = File::open("hello.txt");

    let _f = match f {
        Ok(file) => file,
        Err(error) => {
            panic!("Error opening the file: {:?}", error)
        }
    };
}
