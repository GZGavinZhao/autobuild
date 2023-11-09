// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/dominikbraun/graph"
	"github.com/getsolus/libeopkg/index"
)

type BinaryState struct {
	packages     []common.Package
	nameToSrcIdx map[string]int
	depGraph     *graph.Graph[int, int]
	isGit        bool
}

func (s *BinaryState) Packages() []common.Package {
	return s.packages
}

func (s *BinaryState) NameToSrcIdx() map[string]int {
	return s.nameToSrcIdx
}

func (s *BinaryState) DepGraph() *graph.Graph[int, int] {
	return s.depGraph
}

func (s *BinaryState) BuildGraph() {
	panic("Not Implmeneted!")
}

func LoadBinary(path string) (state BinaryState, err error) {
	eopkgIndex, err := index.Load(path)
	if err != nil {
		return
	}

	// Iterate through the eopkg index and check if there are version/release
	// discrepancies between the source repository and the binary index.
	for _, ipkg := range eopkgIndex.Packages {
		if ipkg.Name != ipkg.Source.Name {
			continue
		}

		var pkg common.Package
		pkg, err = common.ParseIndexPackage(ipkg)
		if err != nil {
			return
		}
		state.packages = append(state.packages, pkg)
	}

	for idx, pkg := range state.packages {
		state.nameToSrcIdx[pkg.Name] = idx
	}

	return
}
