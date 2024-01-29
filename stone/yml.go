// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package stone

import (
	"github.com/deckarep/golang-set/v2"
	"gopkg.in/yaml.v3"
	"os"
)

type SubPackage struct {
	Summary     string   `yaml:"summary"`
	Description string   `yaml:"description"`
	RunDeps     []string   `yaml:"rundeps"`
	Paths       []string `yaml:"paths"`
}

type StoneYML struct {
	Name        string                  `yaml:"name"`
	Version     string                  `yaml:"string"`
	Release     int                     `yaml:"release"`
	RunDeps     []string                `yaml:"rundeps"`
	BuildDeps   []string                `yaml:"builddeps"`
	CheckDeps   []string                `yaml:"checkdeps"`
	Toolchain   string                  `yaml:"toolchain"`
	SubPackages []map[string]SubPackage `yaml:"packages"`
}

func (s *StoneYML) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawStoneYML StoneYML
	raw := rawStoneYML{
		Toolchain: "clang",
	}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*s = StoneYML(raw)
	return nil
}

func (s *StoneYML) CollectRunDeps() (res []string) {
	set := mapset.NewSet[string](s.RunDeps...)

	for _, subpkg := range s.SubPackages {
		for _, subpkg := range subpkg {
			set.Append(subpkg.RunDeps...)
		}
	}

	return set.ToSlice()
}

func Load(path string) (pkg StoneYML, err error) {
	raw, err := os.Open(path)
	if err != nil {
		return
	}
	defer raw.Close()

	dec := yaml.NewDecoder(raw)
	err = dec.Decode(&pkg)
	return
}
