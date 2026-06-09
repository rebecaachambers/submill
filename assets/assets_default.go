//go:build !darwin && !linux && !windows
// +build !darwin,!linux,!windows

package assets

import (
	_ "embed"
)

// ????????
// ??????NODEBIN_PATH
var EmbeddedNode []byte
