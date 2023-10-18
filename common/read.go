package common

import (
	"io/fs"
	"path/filepath"
	"slices"
	"sync"

	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/charlievieth/fastwalk"
)

var (
	badPackages = [...]string{"haskell-http-client-tls"}
)

func ReadSrcPkgs(path string) (pkgs []Package, err error) {
	walkConf := fastwalk.Config{
		Follow: false,
	}
	// ch := make(chan int)
	var mutex sync.Mutex

	err = fastwalk.Walk(&walkConf, path, func(path string, d fs.DirEntry, err error) error {
		// err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		// Some hard-coded problematic packages
		if slices.Contains(badPackages[:], filepath.Base(path)) {
			return nil
		}

		// TODO: handle legacy XML packages too
		pkgFile := filepath.Join(path, "package.yml")
		if !utils.FileExists(pkgFile) {
			return nil
		}

		pkg, err := ParsePackage(path)
		if err != nil {
			return err
		}

		// ch <- pkg
		mutex.Lock()
		pkgs = append(pkgs, pkg)
		mutex.Unlock()

		return filepath.SkipDir
	})
	if err != nil {
		return
	}

	return
}
