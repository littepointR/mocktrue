//go:build !ios

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS is a no-op on non-iOS platforms.
//
//nolint:unused // Reserved as the non-iOS counterpart to the iOS build hook.
func modifyOptionsForIOS(_ *application.Options) {
	// No modifications needed for non-iOS platforms
}
