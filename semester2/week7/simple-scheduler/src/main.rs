use simple_scheduler::*;

fn main() {
    let scheduler = Scheduler::<String>::new(4);

    // Submit 20 tasks; every seventh task deliberately fails.
    for i in 0..20 {
        let item = WorkItem::new(format!("compute-{i}"), move || {
            // Simulate some work.
            std::thread::sleep(std::time::Duration::from_millis(10));
            if i % 7 == 0 {
                Err(format!("task {} hit a simulated error", i))
            } else {
                Ok(format!("task {} completed with value {}", i, i * i))
            }
        });
        if let Err(e) = scheduler.submit(item) {
            eprintln!("Error occurrs: {}", e)
        }
    }

    println!("All tasks submitted. Shutting down...");
    let results = scheduler.shutdown();

    let (ok, err): (Vec<_>, Vec<_>) = results.iter().partition(|r| r.outcome.is_ok());

    println!("\n✅ Succeeded: {}", ok.len());
    for r in &ok {
        println!("  {} → {}", r.name, r.outcome.as_ref().unwrap());
    }

    println!("\n❌ Failed: {}", err.len());
    for r in &err {
        println!("  {} → {}", r.name, r.outcome.as_ref().unwrap_err());
    }
}
