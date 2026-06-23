// Package app wires the PortWeave application shell: it builds the Wails v3
// *application.App from a module registry, bridges the EventBus to the
// frontend, and drives module InitAll/StartAll from the ApplicationStarted
// event. This package is the only place that imports Wails; business modules
// stay Wails-agnostic.
package app
