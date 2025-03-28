// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"github.com/yourbasic/graph"
	"path/filepath"
)

type BinaryState struct {
	packages    []Package
	pvdToPkgIdx map[string]int
	srcToPkgIds map[string][]int
	depGraph    *graph.Immutable
	isGit       bool
}

func (s *BinaryState) Packages() []Package {
	return s.packages
}

func (s *BinaryState) SrcToPkgIds() map[string][]int {
	return s.srcToPkgIds
}

func (s *BinaryState) PvdToPkgIdx() map[string]int {
	return s.pvdToPkgIdx
}

func (s *BinaryState) DepGraph() *graph.Immutable {
	return s.depGraph
}

func LoadBinary(path string) (st *BinaryState, err error) {
	ext := filepath.Ext(path)

	if ext == ".xml" {
		st, err = loadEopkgIndex(path)
	} else if ext == ".stone" {
		err = errors.New("Not implemented")
	} else {
		err = errors.ErrUnsupported
	}

	return
}

func LoadRepo(repoName string) (st *BinaryState, err error) {
	st, err = loadEopkgRepo(repoName)
	return
}

// func LoadBinary(path string) (state *BinaryState, err error) {
// 	eopkgIndex, err := index.Load(path)
// 	if err != nil {
// 		return
// 	}
//
// 	state, err = LoadEopkgIndex(eopkgIndex)
// 	return
// }
//
