package assets

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rtfb/bark"
	"github.com/rtfb/cachedir"
	"github.com/rtfb/rtfblog/src/rtfblog_resources"
)

// Bin wraps around all assets, both the baked-in, and on-disk.
type Bin struct {
	// root path of physical assets in filesystem. These files will only be read
	roDir string

	// writable path. This is where files will be written, as well
	// as read. Writable files will take precedence over read-only when looking
	// up files.
	wrDir string

	// only consider FS files, don't fallback to baked
	fsOnly bool
}

// NewBin creates a new Bin.
func NewBin(roDir, wrDir string, logger *bark.Logger) (*Bin, error) {
	logger.Printf("assets.NewBin: roDir=%s, wrDir=%s\n", roDir, wrDir)
	err := os.MkdirAll(wrDir, 0750)
	return &Bin{
		roDir: roDir,
		wrDir: wrDir,
	}, err
}

func (a *Bin) FSOnly() *Bin {
	return &Bin{
		roDir:  a.roDir,
		wrDir:  a.wrDir,
		fsOnly: true,
	}
}

func (a *Bin) WriteRoot() string {
	return a.wrDir
}

func (a *Bin) Load(path string) ([]byte, error) {
	fullPath, wrExists, err := fileExistsAt(path, a.wrDir)
	if err != nil {
		return nil, err
	}
	// Physical writable file takes highest precedence
	if wrExists {
		return ioutil.ReadFile(fullPath)
	}
	fullPath, roExists, err := fileExistsAt(path, a.roDir)
	if err != nil {
		return nil, err
	}
	// Physical read-only file takes lower precedence, and it's the last thing
	// to try if fsOnly is set to true
	if roExists || a.fsOnly {
		return ioutil.ReadFile(fullPath)
	}
	// Fall back to baked asset
	return rtfblog_resources.Asset(path)
}

func fileExistsAt(filePath, root string) (string, bool, error) {
	path := filePath
	if !strings.HasPrefix(path, "/") {
		path = filepath.Join(root, filePath)
	}
	ok, err := FileExists(path)
	return path, ok, err
}

func (a *Bin) MustLoad(path string) []byte {
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

func (a *Bin) Open(name string) (http.File, error) {
	f, err := http.Dir(a.wrDir).Open(name)
	if err == nil {
		return f, nil
	}
	f, err = http.Dir(a.roDir).Open(name)
	if err == nil {
		return f, nil
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
