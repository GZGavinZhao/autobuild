// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type AutobuildConfig struct {
	Ignore bool         `yaml:"ignore"`
	Solver SolverConfig `yaml:"solver"`
}

func Load(path string) (cfg AutobuildConfig, err error) {
	raw, err := os.Open(path)
	if err != nil {
		return
	}
	defer raw.Close()
	dec := yaml.NewDecoder(raw)
	err = dec.Decode(&cfg)
	return
}
