// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package ypkg

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	YmlFile   = "package.yml"
	PspecFile = "pspec_x86_64.xml"
)

type PackageYML struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	Release     int       `yaml:"release"`
	Component   yaml.Node `yaml:"component"`
	Patterns    yaml.Node `yaml:"patterns"`
	RunDeps     yaml.Node `yaml:"rundeps"`
	BuildDeps   []string  `yaml:"builddeps"`
	CheckDeps   []string  `yaml:"checkdeps"`
	Environment string    `yaml:"environment"`
	Setup       string    `yaml:"setup"`
	Build       string    `yaml:"build"`
	Install     string    `yaml:"install"`
	Networking  bool      `yaml:"networking"`
	Clang       bool      `yaml:"clang"`
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

func (p *PackageYML) ParseRunDeps() (res []string, err error) {
	if p.RunDeps.Kind == 0 {
		return
	} else if p.RunDeps.Kind == yaml.SequenceNode {
		for _, children := range p.RunDeps.Content {
			if children.Kind == yaml.ScalarNode {
				res = append(res, children.Value)
			} else if children.Kind == yaml.MappingNode {
				for _, subpkg := range children.Content {
					for _, rundep := range subpkg.Content {
						if rundep.Kind != yaml.ScalarNode {
							continue
						}

						res = append(res, rundep.Value)
					}
				}
			}
		}
	} else {
		err = fmt.Errorf("ParseRunDeps: invalid YAML node type %q, content %s", p.RunDeps.Kind, p.RunDeps.Value)
	}

	return
}
