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

func LoadBinary(path string) (state *BinaryState, err error) {
	state = &BinaryState{}
	state.nameToSrcIdx = make(map[string]int)

	eopkgIndex, err := index.Load(path)
	if err != nil {
		return
	}

	// Iterate through the eopkg index and check if there are version/release
	// discrepancies between the source repository and the binary index.
	for _, ipkg := range eopkgIndex.Packages {
		if _, ok := state.nameToSrcIdx[ipkg.Source.Name]; ok {
			continue
		}

		var pkg common.Package
		pkg, err = common.ParseIndexPackage(ipkg)
		if err != nil {
			return
		}

		// TODO: is this O(N^2)? Check how `len` is calculated.
		state.nameToSrcIdx[pkg.Name] = len(state.packages)
		state.packages = append(state.packages, pkg)
	}

	return
}
