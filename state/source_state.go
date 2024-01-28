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
	"github.com/GZGavinZhao/autobuild/stone"
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
	if s.depGraph == nil {
		s.buildGraph()
	}
	return s.depGraph
}

func (s *SourceState) IsGit() bool {
	return s.isGit
}

func (s *SourceState) buildGraph() {
	g := graph.New(graph.IntHash, graph.Directed(), graph.Acyclic())

	for pkgIdx, pkg := range s.packages {
		attrsFunc := func(p *graph.VertexProperties) {
			p.Attributes["label"] = fmt.Sprintf("%s %d", pkg.Name, pkgIdx)
			p.Attributes["color"] = "2"
			p.Attributes["fillcolor"] = "1"
			if pkg.Synced {
				// if !pkg.Resolved {
				// 	waterlog.Fatalf("Package %s is synced but not all of its dependencies are solved!", pkg.Name)
				// }
				p.Attributes["colorscheme"] = "greens3"
				p.Attributes["style"] = "filled"
			} else if !pkg.Resolved {
				p.Attributes["colorscheme"] = "reds3"
				p.Attributes["style"] = "filled"
			} else if !pkg.Synced {
				p.Attributes["colorscheme"] = "ylorbr3"
				p.Attributes["style"] = "filled"
			}
		}

		g.AddVertex(pkgIdx, attrsFunc)
	}

	for pkgIdx, pkg := range s.packages {
		for _, dep := range pkg.BuildDeps {
			depIdx, depFound := s.nameToSrcIdx[dep]
			if !depFound {
				// waterlog.Fatalf("Dependency %s of package %s is not found!\n", dep, pkg.Name)
			} else if pkgIdx != depIdx {
				err := g.AddEdge(depIdx, pkgIdx)
				if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
					panic(errors.New(fmt.Sprintf("Failed to create edge from %s to %s: %s\n", dep, pkg.Name, err)))
				}
			}
		}
	}

	s.depGraph = &g
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

	err = fastwalk.Walk(&walkConf, path, func(pkgpath string, d fs.DirEntry, err error) error {
		// err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		// Some hard-coded problematic packages
		if slices.Contains(badPackages[:], filepath.Base(pkgpath)) {
			return nil
		}

		cfgFile := filepath.Join(pkgpath, "autobuild.yml")
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
		var pkg common.Package

		ypkgFile := filepath.Join(pkgpath, "package.yml")
		stoneFile := filepath.Join(pkgpath, "stone.yml")

		if utils.PathExists(ypkgFile) {
			pkg, err = common.ParsePackage(pkgpath)
			if err != nil {
				return err
			}
		} else if utils.PathExists(stoneFile) {
			pkg, err = stone.ParsePackage(pkgpath)
			if err != nil {
				return fmt.Errorf("Failed to parse %s: %w", pkgpath, err)
			}
		} else {
			return nil
		}

		pkg.Root = path

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
