// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/GZGavinZhao/autobuild/common"
	"github.com/yourbasic/graph"
	"github.com/getsolus/libeopkg/index"
	"github.com/ulikunitz/xz"
)

type BinaryState struct {
	packages     []common.Package
	nameToSrcIdx map[string]int
	depGraph     *graph.Immutable
	isGit        bool
}

func (s *BinaryState) Packages() []common.Package {
	return s.packages
}

func (s *BinaryState) NameToSrcIdx() map[string]int {
	return s.nameToSrcIdx
}

func (s *BinaryState) DepGraph() *graph.Immutable {
	return s.depGraph
}

func (s *BinaryState) BuildGraph() {
	panic("Not Implmeneted!")
}

func LoadEopkgIndex(i *index.Index) (state *BinaryState, err error) {
	state = &BinaryState{}
	state.nameToSrcIdx = make(map[string]int)
	// Iterate through the eopkg index and check if there are version/release
	// discrepancies between the source repository and the binary index.
	for _, ipkg := range i.Packages {
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

func LoadBinary(path string) (state *BinaryState, err error) {
	eopkgIndex, err := index.Load(path)
	if err != nil {
		return
	}

	state, err = LoadEopkgIndex(eopkgIndex)
	return
}

func LoadEopkgRepo(name string) (state *BinaryState, err error) {
	indexUrl := fmt.Sprintf("https://packages.getsol.us/%s/eopkg-index.xml.xz", name)
	resp, err := http.Get(indexUrl)
	if err != nil {
		err = fmt.Errorf("Failed to fetch binary index from url %s: %w", indexUrl, err)
		return
	}

	r, err := xz.NewReader(resp.Body)
	if err != nil {
		err = fmt.Errorf("Failed to create XZ reader with binary index from url %s: %w", indexUrl, err)
		return
	}

	dec := xml.NewDecoder(r)
	var i index.Index
	err = dec.Decode(&i)
	if err != nil {
		err = fmt.Errorf("Failed to decode binary index from url %s: %w", indexUrl, err)
		return
	}

	state, err = LoadEopkgIndex(&i)
	return
}
