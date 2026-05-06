//! A typed task scheduler with compile‑time lifecycle enforcement.
//!
//! The scheduler processes `WorkItem<R>` values submitted through `submit()`.
//! A separate `Task<State, R>` typestate machine demonstrates that
//! invalid state transitions (e.g., running a completed task) are
//! prevented at compile time.

use std::marker::PhantomData;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex, mpsc};
use std::thread;

// ---------------------------------------------------------------------------
// State markers for the typestate pattern (zero‑sized types)
// ---------------------------------------------------------------------------

/// The task has been created but not yet dispatched to a worker.
pub struct Pending;
/// The task is currently being executed by a worker.
pub struct Running;
/// The task completed successfully.
pub struct Completed;
/// The task failed with an error.
pub struct Failed;

// ---------------------------------------------------------------------------
// TaskId – a newtype for type safety
// ---------------------------------------------------------------------------

/// Unique identifier for a task.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct TaskId(pub u64);

// ---------------------------------------------------------------------------
// Typestate Task
// ---------------------------------------------------------------------------

/// A task whose lifecycle is tracked at the type level.
///
/// The generic parameter `State` is one of `Pending`, `Running`, `Completed`,
/// or `Failed`. `R` is the type of the successful result.
pub struct Task<State, R> {
    id: TaskId,
    name: String,
    _state: PhantomData<State>,
    _result: PhantomData<R>,
}

// Transition implementations – each method consumes `self` and returns
// a new `Task` with the target state.  No other transitions are possible,
// so trying to e.g. `start()` a `Completed` task is a compile‑time error.

impl<R: Send + 'static> Task<Pending, R> {
    /// Move from `Pending` to `Running`.
    pub fn start(self) -> Task<Running, R> {
        Task {
            id: self.id,
            name: self.name,
            _state: PhantomData,
            _result: PhantomData,
        }
    }

    /// Convenience: create a `WorkItem` directly from a pending task.
    ///
    /// This ties the compile‑time typestate to the runtime execution.
    pub fn into_work_item(
        self,
        work: impl FnOnce() -> Result<R, String> + Send + 'static,
    ) -> WorkItem<R> {
        WorkItem {
            id: self.id,
            name: self.name,
            work: Box::new(work),
        }
    }
}

impl<R> Task<Running, R> {
    /// Transition to `Completed` with a successful result.
    pub fn complete(self, _result: R) -> Task<Completed, R> {
        Task {
            id: self.id,
            name: self.name,
            _state: PhantomData,
            _result: PhantomData,
        }
    }

    /// Transition to `Failed` with an error message.
    pub fn fail(self, _error: String) -> Task<Failed, R> {
        Task {
            id: self.id,
            name: self.name,
            _state: PhantomData,
            _result: PhantomData,
        }
    }
}

// Optional: retrieving results once the task is in a final state.
impl<R> Task<Completed, R> {
    /// The value is only accessible when the state is `Completed`.
    // (In a real application you might store the value inside the struct.)
    pub fn result(&self) -> &PhantomData<R> {
        &PhantomData
    }
}

impl<R> Task<Failed, R> {
    /// The error is only accessible when the state is `Failed`.
    pub fn error(&self) -> &PhantomData<String> {
        &PhantomData
    }
}

// ---------------------------------------------------------------------------
// Global task ID generator
// ---------------------------------------------------------------------------

static NEXT_ID: AtomicU64 = AtomicU64::new(0);

fn next_id() -> TaskId {
    TaskId(NEXT_ID.fetch_add(1, Ordering::SeqCst))
}

// ---------------------------------------------------------------------------
// WorkItem – the unit of work dispatched to workers
// ---------------------------------------------------------------------------

/// A boxed closure that will produce a `Result<R, String>` and is `Send`.
pub struct WorkItem<R: Send + 'static> {
    id: TaskId,
    name: String,
    work: Box<dyn FnOnce() -> Result<R, String> + Send>,
}

impl<R: Send + 'static> WorkItem<R> {
    /// Create a new `WorkItem` with a unique ID.
    ///
    /// The closure will be executed exactly once by a worker.
    pub fn new(name: String, work: impl FnOnce() -> Result<R, String> + Send + 'static) -> Self {
        WorkItem {
            id: next_id(),
            name,
            work: Box::new(work),
        }
    }
}

// ---------------------------------------------------------------------------
// TaskResult – the outcome of a completed or failed task
// ---------------------------------------------------------------------------

/// The result produced by a worker after executing a `WorkItem`.
pub struct TaskResult<R> {
    pub id: TaskId,
    pub name: String,
    pub outcome: Result<R, String>,
}

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

use thiserror::Error;

/// Errors that can occur when interacting with the scheduler.
#[derive(Error, Debug)]
pub enum SchedulerError {
    #[error("scheduler is shut down")]
    ShutDown,

    #[error("task {0:?} failed: {1}")]
    TaskFailed(TaskId, String),

    /// Wraps an `mpsc::SendError` when the result channel is closed.
    #[error("channel send error")]
    ChannelError(#[from] std::sync::mpsc::SendError<()>),

    #[error("worker panicked")]
    WorkerPanic,
}

// ---------------------------------------------------------------------------
// Scheduler
// ---------------------------------------------------------------------------

/// Coordinates task submission, worker execution, and result collection.
///
/// Generic over the successful result type `R`.  `R` must be `Send + 'static`
/// so it can be moved across threads.
pub struct Scheduler<R: Send + 'static> {
    /// The sending half of the work channel – `None` after `shutdown()`.
    sender: Option<mpsc::Sender<WorkItem<R>>>,
    /// Receiving half of the result channel.
    results: mpsc::Receiver<TaskResult<R>>,
    /// Number of worker threads we spawned.
    num_workers: usize,
    /// We keep the worker thread handles so we can join them on shutdown.
    /// (Alternatively, we can use scoped threads, as hinted in Ch 6.)
    workers: Vec<thread::JoinHandle<()>>,
}

/// The worker loop – runs in each worker thread.
fn worker_loop<R: Send + 'static>(
    receiver: Arc<Mutex<mpsc::Receiver<WorkItem<R>>>>,
    result_tx: mpsc::Sender<TaskResult<R>>,
    _worker_id: usize,
) {
    loop {
        // Lock the shared receiver and wait for the next work item.
        let item = {
            let rx = receiver.lock().unwrap();
            rx.recv() // blocks until a WorkItem is available or the sender is dropped
        };

        match item {
            Ok(work_item) => {
                // Execute the work closure.
                let outcome = (work_item.work)();
                // Send the result back – ignore error if the scheduler already shut down.
                let _ = result_tx.send(TaskResult {
                    id: work_item.id,
                    name: work_item.name,
                    outcome,
                });
            }
            Err(_) => {
                // The sender has been dropped => shutdown time.
                break;
            }
        }
    }
}

impl<R: Send + 'static> Scheduler<R> {
    /// Create a new scheduler with `num_workers` worker threads.
    ///
    /// The scheduler immediately spawns the workers, which will block until
    /// tasks are submitted.
    pub fn new(num_workers: usize) -> Self {
        assert!(num_workers > 0, "need at least one worker");

        // Channel for work items – multiple producers, single consumer per worker.
        // We'll use one receiver shared across all workers via Arc<Mutex<>>.
        let (work_tx, work_rx) = mpsc::channel::<WorkItem<R>>();
        // Channel for results – single producer per worker, single consumer (the scheduler).
        let (result_tx, result_rx) = mpsc::channel::<TaskResult<R>>();

        let shared_rx = Arc::new(Mutex::new(work_rx));
        let mut workers = Vec::with_capacity(num_workers);

        for i in 0..num_workers {
            let rx_clone = Arc::clone(&shared_rx);
            let tx_clone = result_tx.clone();
            workers.push(thread::spawn(move || {
                worker_loop(rx_clone, tx_clone, i);
            }));
        }

        // The initial result_tx is no longer needed; drop it so the channel
        // closes only after all worker clones are dropped.
        drop(result_tx);

        Scheduler {
            sender: Some(work_tx),
            results: result_rx,
            num_workers,
            workers,
        }
    }

    /// Submit a `WorkItem` to be executed by one of the workers.
    ///
    /// Returns the assigned `TaskId` so the caller can track the task.
    pub fn submit(&self, item: WorkItem<R>) -> Result<TaskId, SchedulerError> {
        let id = item.id;
        self.sender
            .as_ref()
            .ok_or(SchedulerError::ShutDown)?
            .send(item)
            .map_err(|_| SchedulerError::ShutDown)?;
        Ok(id)
    }

    /// Shut down the scheduler: stop accepting new tasks, wait for all
    /// workers to finish pending work, and collect all results.
    ///
    /// This consumes the scheduler so no further submissions are possible.
    pub fn shutdown(mut self) -> Vec<TaskResult<R>> {
        // Drop the work sender – this signals workers to exit after they
        // drain any remaining items.
        drop(self.sender.take());

        // Wait for all workers to finish.
        for handle in self.workers.drain(..) {
            // If a worker panicked, we propagate the panic as an error.
            // Here we simply ignore panics for brevity; a production version
            // would join and handle results.
            let _ = handle.join();
        }

        // Drain the result channel.  After all workers have stopped, the
        // channel is closed and `iter()` will stop.
        self.results.into_iter().collect()
    }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------
#[cfg(test)]
mod tests {
    use super::*;

    /// Happy path: submit 10 successful tasks, shut down, check all are Ok.
    #[test]
    fn happy_path() {
        let scheduler = Scheduler::<String>::new(4);

        for i in 0..10 {
            let item = WorkItem::new(format!("task-{i}"), move || Ok(format!("result-{i}")));
            scheduler.submit(item).unwrap();
        }

        let results = scheduler.shutdown();
        assert_eq!(results.len(), 10);
        for r in &results {
            assert!(r.outcome.is_ok(), "task {} should be Ok", r.name);
        }
    }

    /// Error handling: submit tasks, some of which fail.
    #[test]
    fn handles_failures() {
        let scheduler = Scheduler::<String>::new(2);

        scheduler
            .submit(WorkItem::new("good".into(), || Ok("ok".into())))
            .unwrap();
        scheduler
            .submit(WorkItem::new("bad".into(), || Err("boom".into())))
            .unwrap();

        let results = scheduler.shutdown();
        assert_eq!(results.len(), 2);

        let failures: Vec<_> = results.iter().filter(|r| r.outcome.is_err()).collect();
        assert_eq!(failures.len(), 1);
        assert_eq!(failures[0].outcome.as_ref().unwrap_err(), "boom");
    }

    /// Edge case: create and immediately shut down – no tasks, no panics.
    #[test]
    fn empty_scheduler() {
        let scheduler = Scheduler::<i32>::new(2);
        let results = scheduler.shutdown();
        assert!(results.is_empty());
    }

    /// Compile‑time test: the typestate prevents invalid transitions.
    ///
    /// Uncommenting any of the lines below will produce a compiler error.
    #[test]
    fn typestate_transitions_are_compile_time() {
        // Create a pending task (using a dummy work closure for the test).
        // Note: `Task` does not store the closure; we only use it for the
        // state demonstration.  We'll construct it manually.
        let pending = Task::<Pending, String> {
            id: TaskId(42),
            name: "demo".into(),
            _state: PhantomData,
            _result: PhantomData,
        };

        // Valid: Pending -> Running
        let running = pending.start();
        // Valid: Running -> Completed
        let completed = running.complete("success".into());
        // Valid: we can inspect the `Completed` task.
        let _ = completed.result();

        // Re-create another running task to test failure transition.
        let pending2 = Task::<Pending, String> {
            id: TaskId(43),
            name: "fail".into(),
            _state: PhantomData,
            _result: PhantomData,
        };
        let running2 = pending2.start();
        let failed = running2.fail("oops".into());
        let _ = failed.error();

        // The following lines would NOT compile – uncomment to see the errors:
        // let nope = pending.complete("bad".into());   // no `complete` on Pending
        // let _ = pending.fail("bad".into());          // no `fail` on Pending
        // let rerun = completed.start();               // no `start` on Completed
    }

    // Property‑based test (requires `proptest` in dev-dependencies).
    #[test]
    fn property_always_returns_exact_n_results() {
        use proptest::prelude::*;

        let strategy = 1..=100usize;
        proptest!(|(num_tasks in strategy)| {
            let scheduler = Scheduler::<String>::new(2);
            for i in 0..num_tasks {
                scheduler.submit(WorkItem::new(
                    format!("task-{}", i),
                    move || Ok(format!("result-{}", i)),
                )).unwrap();
            }
            let results = scheduler.shutdown();
            assert_eq!(results.len(), num_tasks);
        });
    }
}
