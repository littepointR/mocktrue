//go:build windows && integration

package integration

// This file keeps the automation integration package buildable on Windows.
// POSIX socat-backed automation tests are compiled only on darwin/linux, while
// Windows real-COM coverage lives in tests/go/integration.
