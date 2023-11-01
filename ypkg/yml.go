// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package ypkg

import (
	"gopkg.in/yaml.v3"
	"os"
)

type PackageYML struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Release     int      `yaml:"release"`
	RunDeps     []string `yaml:"rundeps"`
	BuildDeps   []string `yaml:"builddeps"`
	CheckDeps   []string `yaml:"checkdeps"`
	Environment string   `yaml:"environment"`
	Setup       string   `yaml:"setup"`
	Build       string   `yaml:"build"`
	Install     string   `yaml:"install"`
}

func Load(path string) (pkg PackageYML, err error) {
	raw, err := os.Open(path)
	if err != nil {
		return
	}
	defer raw.Close()
	dec := yaml.NewDecoder(raw)
	err = dec.Decode(&pkg)
	return
}
