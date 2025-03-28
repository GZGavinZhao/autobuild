package state

import (
	"fmt"
	"strings"

	"github.com/jwalton/gchalk"
)

type Package struct {
	Path      string
	Names     []string
	Source    string
	Version   string
	Root      string
	Release   int
	Provides  []string
	BuildDeps []string
	RunDeps   []string
	Ignores   []string
}

// // Merge the info from `other` to itself. Prefer `other` if different.
// func (p *Package) Merge(o Package) {
// 	if len(o.Name) > 0 {
// 		p.Name = o.Name
// 	}
// 	if len(o.Version) > 0 {
// 		p.Version = o.Version
// 	}
// 	if len()
// }

// Show is the toString method for a package.
//
// When `sub` is true, show the subpackages that this package (node) represents.
//
// When `color` is true, show the subpackages in gray color for easier viewing.
// Obviously this has no effects when `sub` is false.
func (p *Package) Show(sub bool, color bool) string {
	if !sub {
		return p.Source
	} else if color {
		return p.Source + gchalk.Gray(fmt.Sprintf("{%s}", strings.Join(p.Names, ", ")))
	} else {
		return fmt.Sprintf("%s{%s}", p.Source, strings.Join(p.Names, ", "))
	}
}
