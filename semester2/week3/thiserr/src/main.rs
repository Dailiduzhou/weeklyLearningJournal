use std::{fs, io};
use thiserror::Error;

// Simulated parse error type imported from another part
#[derive(Debug, Clone, Error)]
#[error("parse data error")]
pub struct ParseDataError;

// Definition of a custom error type
#[derive(Error, Debug)]
pub enum MyError {
    // Description for IoError, containing a nested io::Error
    #[error("I/O error occurred")]
    IoError(#[from] io::Error),
    // Description for ParseError, containing a nested ParseDataError
    #[error("failed to parse data")]
    ParseError(#[from] ParseDataError),
}

// Read a file and attempt to parse its contents
fn read_and_parse(filename: &str) -> Result<String, MyError> {
    // Read file contents, may throw an I/O error
    let content = fs::read_to_string(filename)?;
    // Attempt to parse contents, may throw a parse error
    parse_data(&content).map_err(MyError::from)
}

// Simulated data parsing function, which always returns an error here
fn parse_data(_content: &str) -> Result<String, ParseDataError> {
    Err(ParseDataError)
}

// Main function demonstrating how to use the above error handling logic
fn main() {
    match read_and_parse("data.txt") {
        Ok(data) => println!("Data: {}", data),
        Err(e) => eprintln!("Error: {}", e),
    }
}
