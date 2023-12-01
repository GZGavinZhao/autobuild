// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package push

import (
	"time"
)

type Job struct {
	ID       int        `json:"id"`
	Pkg      string     `json:"pkg"`
	Tag      string     `json:"tag"`
	Status   string     `json:"status"`
	Builder  string     `json:"builder"`
	Finished *time.Time `json:"finished,omitempty"`
	Path     *string    `json:"path,omitempty"`
	Ref      *string    `json:"ref,omitempty"`
}
