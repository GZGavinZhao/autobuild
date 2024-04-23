// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sync"

	"github.com/DataDrake/waterlog"
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
				err := g.AddEdge(depIdx, pkgIdx, graph.EdgeWeight(1))
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
	_ = walkConf
	var mutex sync.Mutex

	// err = filepath.WalkDir(path, func(pkgpath string, d fs.DirEntry, err error) error {
	err = fastwalk.Walk(&walkConf, path, func(pkgpath string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		// Some hard-coded problematic packages
		if slices.Contains(badPackages[:], filepath.Base(pkgpath)) {
			return nil
		}

		for _, cfgFile := range []string{"autobuild.yaml", "autobuild.yml"} {
			cfgFile = filepath.Join(pkgpath, cfgFile)
			if utils.PathExists(cfgFile) {
				waterlog.Debugf("LoadSource: loading config file for %s at %s\n", filepath.Base(pkgpath), cfgFile)
				abConfig, err := config.Load(cfgFile)
				if err != nil {
					return fmt.Errorf("LoadSource: failed to load autobuild config file at %s: %w", cfgFile, err)
				}

				if abConfig.Ignore {
					return filepath.SkipDir
				}

				break
			}
		}

		// TODO: handle legacy XML packages too
		var pkg common.Package

		ypkgFile := filepath.Join(pkgpath, "package.yml")
		stoneFile := filepath.Join(pkgpath, "stone.yaml")

		if utils.PathExists(ypkgFile) {
			if pkg, err = common.ParsePackage(pkgpath); err != nil {
				return fmt.Errorf("Failed to parse %s: %w", ypkgFile, err)
			}
		} else if utils.PathExists(stoneFile) {
			if pkg, err = stone.ParsePackage(pkgpath); err != nil {
				return fmt.Errorf("Failed to parse %s: %w", stoneFile, err)
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

	slices.SortFunc(state.packages, func(a, b common.Package) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for idx, pkg := range state.packages {
		if nidx, ok := state.nameToSrcIdx[pkg.Name]; ok && nidx != idx {
			waterlog.Errorf("Duplicate provider for %s from %s, currently %s\n", pkg.Name, pkg.Name, state.packages[nidx].Name)
		}
		state.nameToSrcIdx[pkg.Name] = idx
		for _, name := range pkg.Provides {
			if nidx, ok := state.nameToSrcIdx[name]; ok && nidx != idx {
				waterlog.Errorf("Duplicate provider for %s from %s, currently %s\n", name, pkg.Name, state.packages[nidx].Name)
			}
			state.nameToSrcIdx[name] = idx
		}
	}

	for idx := range state.packages {
		state.packages[idx].Resolve(state.nameToSrcIdx)
	}

	// fmt.Println("result:", state)
	return
}
