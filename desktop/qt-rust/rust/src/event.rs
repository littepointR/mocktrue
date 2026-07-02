#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RustEventKind {
    StartupSmoke,
    LifecycleNote,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct RustEvent {
    pub kind: RustEventKind,
    pub message: &'static str,
}

pub const STARTUP_SMOKE_BYTES: &[u8] = b"PortWeave Rust core ready\0";

pub fn startup_event() -> RustEvent {
    RustEvent {
        kind: RustEventKind::StartupSmoke,
        message: "PortWeave Rust core ready",
    }
}

pub fn startup_smoke_message() -> &'static str {
    startup_event().message
}

pub fn startup_smoke_bytes() -> &'static [u8] {
    STARTUP_SMOKE_BYTES
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn startup_event_is_stable() {
        let event = startup_event();
        assert_eq!(event.kind, RustEventKind::StartupSmoke);
        assert_eq!(event.message, "PortWeave Rust core ready");
    }
}
