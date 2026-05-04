use std::fmt;
use std::str::FromStr;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Port(u16);

impl Port {
    pub fn get(&self) -> u16 {
        self.0
    }
}

// 1. Parse from strongly-typed raw data (u16 -> Port)
impl TryFrom<u16> for Port {
    type Error = PortError;

    fn try_from(value: u16) -> Result<Self, Self::Error> {
        if value == 0 {
            Err(PortError::Zero)
        } else {
            Ok(Port(value))
        }
    }
}

// 2. Parse from untyped external data (String -> Port)
impl FromStr for Port {
    type Err = PortError;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        let value: u16 = s.parse().map_err(|_| PortError::InvalidFormat)?;
        // Reuse TryFrom logic
        Port::try_from(value)
    }
}

#[derive(Debug, PartialEq, Eq)]
pub enum PortError {
    Zero,
    InvalidFormat,
}

impl fmt::Display for PortError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            PortError::Zero => write!(f, "port must be between 1 and 65535"), // for clarity
            PortError::InvalidFormat => write!(f, "port must be a valid 16-bit integer"),
        }
    }
}

impl std::error::Error for PortError {}

// Now the type system enforces validity!
fn start_server(port: Port) {
    // No validation needed inside this function.
    // If we have a `Port`, it is structurally guaranteed to be valid.
    println!("Listening on port {}", port.get());
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let port1 = Port::try_from(8080)?;
    start_server(port1);

    let port2 = "443".parse::<Port>()?;
    start_server(port2);

    assert_eq!(Port::try_from(0), Err(PortError::Zero));
    assert_eq!("not-a-port".parse::<Port>(), Err(PortError::InvalidFormat));

    println!(
        "{}",
        match "0".parse::<Port>() {
            Ok(_) => return Ok(()),
            Err(e) => e,
        }
    );

    Ok(())
}
