// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sync"

	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/config"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/charlievieth/fastwalk"
	"github.com/dominikbraun/graph"
)

var (
	badPackages = [...]string{"haskell-http-client-tls"}
)

type SourceState struct {
	packages     []common.Package
	nameToSrcIdx map[string]int
	depGraph     *graph.Graph[int, int]
	isGit        bool
}

func (s *SourceState) Packages() []common.Package {
	return s.packages
}

func (s *SourceState) NameToSrcIdx() map[string]int {
	return s.nameToSrcIdx
}

func (s *SourceState) DepGraph() *graph.Graph[int, int] {
	return s.depGraph
}

func (s *SourceState) IsGit() bool {
	return s.isGit
}

func (s *SourceState) BuildGraph() {
	panic("Not Implmeneted!")
}

func LoadSource(path string) (state *SourceState, err error) {
	state = &SourceState{}
	state.nameToSrcIdx = make(map[string]int)

	if utils.PathExists(filepath.Join(path, ".git")) {
		state.isGit = true
	}

	walkConf := fastwalk.Config{
		Follow: false,
	}
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
				return errors.New(fmt.Sprintf("LoadSource: failed to load autobuild config file: %s", err))
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

		pkg, err := common.ParsePackage(path)
		if err != nil {
			return err
		}

		mutex.Lock()
		state.packages = append(state.packages, pkg)
		mutex.Unlock()

		return filepath.SkipDir
	})

	if err != nil {
		return
	}

	for idx, pkg := range state.packages {
		state.nameToSrcIdx[pkg.Name] = idx
		for _, name := range pkg.Provides {
			state.nameToSrcIdx[name] = idx
		}
	}

	for idx := range state.packages {
		state.packages[idx].Resolve(state.nameToSrcIdx)
	}

	// fmt.Println("result:", state)
	return
}
