// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/dominikbraun/graph"
)

type State interface {
	Packages() []common.Package
	NameToSrcIdx() map[string]int
	DepGraph() *graph.Graph[int, int]
}

func Changed(old *State, cur *State) (res []Diff) {
	for idx, pkg := range (*cur).Packages() {
		oldIdx, found := (*old).NameToSrcIdx()[pkg.Name]

		if !found {
			res = append(res, Diff{
				Idx:    idx,
				RelNum: pkg.Release,
				Ver:    pkg.Version,
			})
			continue
		}

		oldPkg := (*old).Packages()[oldIdx]
		if oldPkg.Release != pkg.Release || oldPkg.Version != pkg.Version {
			res = append(res, Diff{
				Idx:       idx,
				OldIdx:    oldIdx,
				RelNum:    pkg.Release,
				OldRelNum: oldPkg.Release,
				Ver:       pkg.Version,
				OldVer:    oldPkg.Version,
			})
		}
	}

	return
}
