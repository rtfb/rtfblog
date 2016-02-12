// +build !go_get

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	// This is a generated package that's being put under $GOPATH by Makefile
	"generated_res_dir.com/rtfb/rtfblog_resources"

	"github.com/rtfb/cachedir"
)

type AssetBin struct {
	root string // root path of physical assets in filesystem
}

func NewAssetBin(binaryDir string) *AssetBin {
	return &AssetBin{
		root: binaryDir,
	}
}

func (a *AssetBin) Load(path string) ([]byte, error) {
	fullPath := filepath.Join(a.root, path)
	exists, err := FileExists(fullPath)
	if err != nil {
		return nil, err
	}
	// Physical file takes precedence
	if exists {
		return ioutil.ReadFile(fullPath)
	}
	// Fall back to baked asset
	return rtfblog_resources.Asset(path)
}

func (a *AssetBin) MustLoad(path string) []byte {
	b, err := a.Load(path)
	if err != nil {
		panic("Failed to read asset '" + path + "'; " + err.Error())
	}
	return b
}

func MustExtractDBAsset(defaultDB string) string {
	path, err := cachedir.Get("rtfblog")
	if err != nil {
		panic("Failed to cachedir.Get()")
	}
	dbPath := filepath.Join(path, defaultDB)
	// Extract it only in case there isn't one already from the last time
	if !FileExistsNoErr(dbPath) {
		err = rtfblog_resources.RestoreAsset(path, defaultDB)
		if err != nil {
			panic(fmt.Sprintf("Failed to RestoreAsset(%q, %q)", path, defaultDB))
		}
	}
	return dbPath
}

func (a *AssetBin) Open(name string) (http.File, error) {
	d := http.Dir(a.root)
	f, err := d.Open(name)
	if err == nil {
		return f, err
	}
	if name[0] == '/' {
		name = name[1:]
	}
	return &AssetFile{name: name}, nil
}

// Implements http.File
type AssetFile struct {
	name      string
	seekIndex int64
	data      []byte // slice into an Asset
}

func (af *AssetFile) Close() error {
	af.seekIndex = 0
	af.data = nil
	return nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (af *AssetFile) Read(p []byte) (n int, err error) {
	if af.data == nil {
		af.data, err = rtfblog_resources.Asset(af.name)
	}
	n64 := min(int64(len(p)), int64(len(af.data))-af.seekIndex)
	copy(p, af.data[af.seekIndex:af.seekIndex+n64])
	return int(n64), err
}

func (af *AssetFile) Seek(offset int64, whence int) (int64, error) {
	// TODO: respect whence and check what should be returned
	af.seekIndex = offset
	return offset, nil
}

func (af *AssetFile) Readdir(n int) (fi []os.FileInfo, err error) {
	dir, err := rtfblog_resources.AssetDir(af.name)
	if err != nil {
		return nil, err
	}
	for i, d := range dir {
		if i == n {
			break
		}
		df := &AssetFile{name: d}
		stat, err2 := df.Stat()
		if err2 != nil {
			return nil, err
		}
		fi = append(fi, stat)
	}
	return
}

func (af *AssetFile) Stat() (os.FileInfo, error) {
	return rtfblog_resources.AssetInfo(af.name)
}
