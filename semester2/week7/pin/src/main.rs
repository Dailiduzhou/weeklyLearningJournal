use std::marker::PhantomPinned;
use std::pin::Pin;

// A self-referential struct (simplified):
struct SelfRef {
    data: String,
    ptr: *const String,  // Points to `data` above
    _pin: PhantomPinned, // Opts out of Unpin — can't be moved
}

impl SelfRef {
    fn new(s: &str) -> Pin<Box<Self>> {
        let val = SelfRef {
            data: s.to_string(),
            ptr: std::ptr::null(),
            _pin: PhantomPinned,
        };
        let mut boxed = Box::pin(val);

        // SAFETY: we don't move the data after setting the pointer
        let self_ptr: *const String = &boxed.data;
        unsafe {
            let mut_ref = Pin::as_mut(&mut boxed);
            Pin::get_unchecked_mut(mut_ref).ptr = self_ptr;
        }
        boxed
    }

    fn data(&self) -> &str {
        &self.data
    }

    fn ptr_data(&self) -> &str {
        // SAFETY: ptr was set to point to self.data while pinned
        unsafe { &*self.ptr }
    }
}

fn main() {
    let pinned = SelfRef::new("hello");
    assert_eq!(pinned.data(), pinned.ptr_data()); // Both "hello"
    // std::mem::swap would invalidate ptr — but Pin prevents it
}
