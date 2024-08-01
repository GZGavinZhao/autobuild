// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sync"

	"github.com/GZGavinZhao/autobuild/config"
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

		cfgFile := filepath.Join(path, "autobuild.yml")
		if utils.PathExists(cfgFile) {
			abConfig, err := config.Load(cfgFile)
			if err != nil {
				return errors.New(fmt.Sprintf("Fail to load autobuild config file: %s", err))
			}

			if abConfig.Ignore {
				return filepath.SkipDir
			}
		}

		// TODO: handle legacy XML packages too
		pkgFile := filepath.Join(path, "package.yml")
		if !utils.PathExists(pkgFile) {
			return nil
		}

		pkg, err := ParsePackage(path)
		if err != nil {
			return err
		}

		// ch <- pkg
		mutex.Lock()
		pkgs = append(pkgs, pkg...)
		mutex.Unlock()

		return filepath.SkipDir
	})
	if err != nil {
		return
	}

	return
}
