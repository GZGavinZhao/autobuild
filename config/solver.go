// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package config

type SolverConfig struct {
	Ignore []string            `yaml:"ignore"`
	Split  []string            `yaml:"split"`
	Move   map[string][]string `yaml:"move"`
}
