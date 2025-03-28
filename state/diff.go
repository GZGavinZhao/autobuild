// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"

	"github.com/jwalton/gchalk"
)

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

// Note that this is with colors!
func (d Diff) Show(color bool) string {
	if !color {
		panic("not implemented!")
	}

	if d.OldRelNum == 0 {
		return fmt.Sprintf("%s-%s", gchalk.Green(d.Ver), gchalk.Green(fmt.Sprint(d.RelNum)))
	}

	oldVerColor := gchalk.Reset
	oldRelColor := gchalk.Reset
	newVerColor := gchalk.Reset
	newRelColor := gchalk.Reset
	if d.OldRelNum < d.RelNum {
		oldRelColor = gchalk.Yellow
		newRelColor = gchalk.Green

		if d.OldVer != d.Ver {
			oldVerColor = gchalk.Yellow
			newVerColor = gchalk.Green
		}
	} else if d.OldRelNum == d.RelNum {
		if d.OldVer != d.Ver {
			oldVerColor = gchalk.Red
			newVerColor = gchalk.Red
		}
	} else if d.OldRelNum > d.RelNum {
		oldRelColor = gchalk.Yellow
		newRelColor = gchalk.Red

		if d.OldVer != d.Ver {
			oldVerColor = gchalk.Yellow
			newVerColor = gchalk.Red
		}
	}

	return fmt.Sprintf("%s-%s -> %s-%s", oldVerColor(d.OldVer), oldRelColor(fmt.Sprint(d.OldRelNum)), newVerColor(d.Ver), newRelColor(fmt.Sprint(d.RelNum)))
}
