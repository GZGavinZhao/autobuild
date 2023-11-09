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
