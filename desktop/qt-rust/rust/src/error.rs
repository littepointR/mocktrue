#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ErrorKind {
    Invalid = 0,
    NotFound = 1,
    Conflict = 2,
    Permission = 3,
    Unsupported = 4,
    Io = 5,
    Timeout = 6,
    Internal = 7,
}

pub const ERROR_KIND_COUNT: i32 = 8;

pub fn error_kind_count() -> i32 {
    ERROR_KIND_COUNT
}

pub fn classify(kind: i32) -> ErrorKind {
    match kind {
        0 => ErrorKind::Invalid,
        1 => ErrorKind::NotFound,
        2 => ErrorKind::Conflict,
        3 => ErrorKind::Permission,
        4 => ErrorKind::Unsupported,
        5 => ErrorKind::Io,
        6 => ErrorKind::Timeout,
        _ => ErrorKind::Internal,
    }
}

pub fn label(kind: ErrorKind) -> &'static str {
    match kind {
        ErrorKind::Invalid => "invalid",
        ErrorKind::NotFound => "not_found",
        ErrorKind::Conflict => "conflict",
        ErrorKind::Permission => "permission",
        ErrorKind::Unsupported => "unsupported",
        ErrorKind::Io => "io",
        ErrorKind::Timeout => "timeout",
        ErrorKind::Internal => "internal",
    }
}

pub fn label_bytes(kind: i32) -> &'static [u8] {
    match classify(kind) {
        ErrorKind::Invalid => b"invalid\0",
        ErrorKind::NotFound => b"not_found\0",
        ErrorKind::Conflict => b"conflict\0",
        ErrorKind::Permission => b"permission\0",
        ErrorKind::Unsupported => b"unsupported\0",
        ErrorKind::Io => b"io\0",
        ErrorKind::Timeout => b"timeout\0",
        ErrorKind::Internal => b"internal\0",
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn classification_labels_cover_all_contract_kinds() {
        let labels: Vec<&str> = (0..error_kind_count())
            .map(|kind| label(classify(kind)))
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
