// Package v4app carries the self-contained Desktop V4 browser and provider assets.
package v4app

import "embed"

// Assets are embedded into the V4 executable so runtime does not depend on a repository checkout.
//
//go:embed web assets/media_bridge.py
var Assets embed.FS

func MediaBridgeBytes() ([]byte, error) {
	data, err := Assets.ReadFile("assets/media_bridge.py")
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), data...), nil
}
