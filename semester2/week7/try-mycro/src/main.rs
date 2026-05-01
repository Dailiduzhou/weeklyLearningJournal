// In the application crate:
use my_derive::Describe;

#[derive(Describe, Debug)]
struct SensorReading {
    sensor_id: u16,
    value: f64,
    timestamp: u64,
}

fn main() {
    println!("{}", SensorReading::describe());
    // "SensorReading { sensor_id, value, timestamp }"
    let st = SensorReading {
        sensor_id: 1,
        value: 1.0,
        timestamp: 1,
    };

    println!("{:?}", st);
}
