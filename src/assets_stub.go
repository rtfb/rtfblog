// +build go_get

// When assets.go is being excluded from the build, it's symbols are undefined
// and 'go get' gets upset. Add this stub with inversed build condition to
// compensate.

package main

import (
	"net/http"
	"os"

	"github.com/rtfb/cachedir"
)

type AssetBin struct {
	root string // root path of physical assets in filesystem
}

func init() {
	cachedir.Get("") // no-op to fix unused import
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

func (a *AssetBin) Open(name string) (http.File, error) {
	return nil, nil
}

// Implements http.File
type AssetFile struct {
}

func (af *AssetFile) Close() error {
	return nil
}

func (af *AssetFile) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (af *AssetFile) Seek(offset int64, whence int) (int64, error) {
	return offset, nil
}

func (af *AssetFile) Readdir(n int) (fi []os.FileInfo, err error) {
	return nil, nil
}

func (af *AssetFile) Stat() (os.FileInfo, error) {
	return nil, nil
}
