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
	Changed(*State) []Diff
}

type Diff struct {
	Idx       int
	OldIdx    int
	RelNum    int
	OldRelNum int
	Ver       string
	OldVer    string
}
