mod error;
mod event;

use std::os::raw::c_char;

#[no_mangle]
pub extern "C" fn portweave_rust_smoke_message() -> *const c_char {
    event::startup_smoke_bytes().as_ptr().cast()
}

#[no_mangle]
pub extern "C" fn portweave_rust_error_label(kind: i32) -> *const c_char {
    error::label_bytes(kind).as_ptr().cast()
}

#[no_mangle]
pub extern "C" fn portweave_rust_error_kind_count() -> i32 {
    error::error_kind_count()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ffi::CStr;

    #[test]
    fn smoke_message_matches_event_module() {
        let raw = portweave_rust_smoke_message();
        let rendered = unsafe { CStr::from_ptr(raw) }.to_str().unwrap();
        assert_eq!(rendered, event::startup_smoke_message());
    }

    #[test]
    fn error_labels_cover_every_classification_kind() {
        let count = portweave_rust_error_kind_count();
        assert_eq!(count, 8);
        let labels: Vec<String> = (0..count)
            .map(|kind| {
                let raw = portweave_rust_error_label(kind);
                unsafe { CStr::from_ptr(raw) }.to_str().unwrap().to_string()
            })
            .collect();
        assert_eq!(
            labels,
            vec![
                "invalid",
                "not_found",
                "conflict",
                "permission",
                "unsupported",
                "io",
                "timeout",
                "internal",
            ]
        );
    }
}
