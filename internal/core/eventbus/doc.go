// Package eventbus provides an in-process pub/sub bus that optionally bridges
// to the Wails frontend: once BridgeToFrontend is called, every Publish also
// calls app.Event.Emit so the webview receives the event.
package eventbus
