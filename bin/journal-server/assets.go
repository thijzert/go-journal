package main

import (
	"embed"
	"io"
)

// embeddedAssets holds static web assets
//go:embed assets
var embeddedAssets embed.FS

func Asset(name string) ([]byte, error) {
	f, err := embeddedAssets.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}
