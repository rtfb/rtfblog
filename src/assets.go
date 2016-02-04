// +build !go_get

package main

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"

	// This is a generated package that's being put under $GOPATH by Makefile
	"generated_res_dir.com/rtfb/rtfblog_resources"
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
	usr, err := user.Current()
	if err != nil {
		panic("Failed to get user.Current()")
	}
	path := filepath.Join(usr.HomeDir, ".rtfblog")
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
