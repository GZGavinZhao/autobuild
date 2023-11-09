// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

type Diff struct {
	Idx       int
	OldIdx    int
	RelNum    int
	OldRelNum int
	Ver       string
	OldVer    string
}

func (d Diff) IsSame() bool {
	return d.RelNum == d.OldRelNum && d.Ver == d.OldVer
}

func (d Diff) IsSameRel() bool {
	return d.RelNum == d.OldRelNum
}

func (d Diff) IsNewRel() bool {
	return d.RelNum > d.OldRelNum
}

func (d Diff) IsUpdate() bool {
	return d.IsNewRel() && d.RelNum != d.OldRelNum
}

func (d Diff) IsDowngrade() bool {
	return d.RelNum < d.OldRelNum
}
