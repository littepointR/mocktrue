// Package module defines PortWeave's plugin/module system.
//
// A Module is a self-contained debugging feature (serial, tcp, can, ...).
// Modules declare dependencies via Manifest; ModuleRegistry starts them in
// topological order and rolls back on failure. Each module exposes
// ServicesWrapped (already application.NewService-wrapped) so the shell can
// aggregate them into Wails Options.Services without reflection.
package module
