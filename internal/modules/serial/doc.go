// Package serial implements the serial-port debugging module.
//
// Stage 0 ships a placeholder: it implements module.Module with no-op
// lifecycle and a minimal Service (Ping) to verify the binding channel.
// Real port management, buffering, recording and protocol parsing arrive in
// later stages.
package serial
