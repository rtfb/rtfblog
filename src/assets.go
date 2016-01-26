package main

import (
	"io/ioutil"
	"path/filepath"

	"generated_res_dir/rtfb/rtfblog_resources"
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
	exists, err := FileExists(filepath.Join(a.root, path))
	if err != nil {
		return nil, err
	}
	// Physical file takes precedence
	if exists {
		return ioutil.ReadFile(path)
	}
	// Fall back to baked asset
	return rtfblog_resources.Asset(path)
}
