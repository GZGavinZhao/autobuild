// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/getsolus/libeopkg/pspec"
)

const (
	ignoreFile = ".autobuild-ignore"
)

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

// Returns the mapping from a binary package to a source package
func getPackages(path string) (res []string, err error) {
	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		if fileExists(filepath.Join(path, ignoreFile)) {
			return nil
		}

		// Check if the given directory contains a package definition
		pkgFile := filepath.Join(path, "package.yml")
		pspecFile := filepath.Join(path, "pspec_x86_64.xml")
		// TODO: handle legacy XML packages too
		if !fileExists(pkgFile) || !fileExists(pspecFile) {
			return nil
		}
		// if !fileExists(pkgFile) {
		// 	pkgFile = filepath.Join(path, "pspec.xml")
		// 	if !fileExists(pkgFile) {
		// 		return nil
		// 	}
		// }
		// if !fileExists(pspecFile) {
		// 	return nil
		// }

		res = append(res, path)

		return nil
	})

	return
}

// Returns the mapping from a binary package to a source package
func getBinaryToSource(path string) (res map[string]string, err error) {
	res = make(map[string]string)
	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		if fileExists(filepath.Join(path, ignoreFile)) {
			return nil
		}

		// Check if the given directory contains a package definition
		pkgFile := filepath.Join(path, "package.yml")
		pspecFile := filepath.Join(path, "pspec_x86_64.xml")
		// TODO: handle legacy XML packages too
		if !fileExists(pkgFile) || !fileExists(pspecFile) {
			return nil
		}
		// if !fileExists(pkgFile) {
		// 	pkgFile = filepath.Join(path, "pspec.xml")
		// 	if !fileExists(pkgFile) {
		// 		return nil
		// 	}
		// }
		// if !fileExists(pspecFile) {
		// 	return nil
		// }

		pspecXml, err := pspec.Load(pspecFile)
		if err != nil {
			return err
		}
		for _, pkg := range pspecXml.Packages {
			res[pkg.Name] = path
		}

		return nil
	})

	return
}
