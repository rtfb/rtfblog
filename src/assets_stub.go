// +build go_get

// When assets.go is being excluded from the build, it's symbols are undefined
// and 'go get' gets upset. Add this stub with inversed build condition to
// compensate.

package main

import (
	"github.com/rtfb/cachedir"
)

type AssetBin struct {
	root string // root path of physical assets in filesystem
}

func NewAssetBin(binaryDir string) *AssetBin {
	return nil
}

func (a *AssetBin) Load(path string) ([]byte, error) {
	return nil, nil
}

func (a *AssetBin) MustLoad(path string) []byte {
	return nil
}

func MustExtractDBAsset(defaultDB string) string {
	return defaultDB
}
