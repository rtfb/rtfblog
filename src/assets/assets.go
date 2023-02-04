package assets

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rtfb/cachedir"
	"github.com/rtfb/rtfblog/src/rtfblog_resources"
)

type AssetBin struct {
	Root   string // root path of physical assets in filesystem
	fsOnly bool   // only consider FS files, don't fallback to baked
}

func NewAssetBin(binaryDir string) *AssetBin {
	return &AssetBin{
		Root: binaryDir,
	}
}

func (a *AssetBin) FSOnly() *AssetBin {
	return &AssetBin{
		Root:   a.Root,
		fsOnly: true,
	}
}

func (a *AssetBin) Load(path string) ([]byte, error) {
	fullPath := path
	if fullPath[0] != '/' {
		fullPath = filepath.Join(a.Root, path)
	}
	exists, err := FileExists(fullPath)
	if err != nil {
		return nil, err
	}
	// Physical file takes precedence
	if exists || a.fsOnly {
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
	d := http.Dir(a.Root)
	f, err := d.Open(name)
	if err == nil {
		return f, err
	}
	if name[0] == '/' {
		name = name[1:]
	}
	data, err := rtfblog_resources.Asset(name)
	return &AssetFile{
		name: name,
		data: data,
	}, err
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

func len64(b []byte) int64 {
	return int64(len(b))
}

func (af *AssetFile) Read(p []byte) (n int, err error) {
	if af.data == nil {
		af.data, err = rtfblog_resources.Asset(af.name)
	}
	n64 := min(len64(p), len64(af.data)-af.seekIndex)
	copy(p, af.data[af.seekIndex:af.seekIndex+n64])
	return int(n64), err
}

func (af *AssetFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case os.SEEK_SET:
		af.seekIndex = offset
	case os.SEEK_CUR:
		af.seekIndex += offset
	case os.SEEK_END:
		af.seekIndex = len64(af.data) - offset
	default:
		return 0, fmt.Errorf("Unknown seek mode %d", whence)
	}
	af.seekIndex = min(af.seekIndex, len64(af.data))
	if af.seekIndex < 0 {
		af.seekIndex = 0
	}
	return af.seekIndex, nil
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

func FileExistsNoErr(path string) bool {
	exists, err := FileExists(path)
	if err != nil {
		return false
	}
	return exists
}

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
