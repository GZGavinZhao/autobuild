## Usage

To compile `autobuild`:
```bash
go build -buildvcs .
```

Or compile and run directly:
```bash
go run -buildvcs . <args-to-autobuild>
```

### Configuration file

The configuration file is named either `autobuild.yaml` or `autobuild.yml`. When
both are found, the former takes precendence over the latter.
It should be place on the same
level as the package recipe file (e.g. `package.yml` or `stone.yaml`).

Currently it's still very simple, with the specification as follows:
```yml
# `true` means skip any package in the current directory and any subdirectories
ignore: false

solver:
  ignore:
    - <regex-of-dependencies-to-ignore>
```

Note that _currently_,
the regex in `ignore` has to match the entire package name.
For example, the following config file

```yml
solver:
  ignore:
    - haskell
```

would not ignore dependencies such as `haskell-cabal-install` and 
`haskell-hashable`, but if it's `haskell.*`, then every package that starts with
`haskell` would be ignored.

### TPath

TPath (typed path) is a way to specify different kinds of files that provide
information on packages. Currently, there are three supported types:

1. Binary, in the form of `bin:<path-to-binary-index>`. Example: 
   `bin:/var/lib/eopkg/index/Unstable/eopkg-index.xml`. Note that this must be
   an XML index file, not an xz-compressed XML index file (`eopkg-index.xml.xz`).
2. Source, in the form of `src:<path-to-source-index>`. The path should point to
   a directory containing YPKG source definitions. Usually this path points to
   the [Solus repository](https://github.com/getsolus/packages).
   Example: `src:$HOME/solus/package`.
3. Remote binary index, in the form of `repo:<name>`. This will fetch the index
   file from the url `https://packages.getsol.us/<name>/eopkg-index.xml.xz` and
   load it in the same way it would load a binary index. Example:
   `repo:unstable`.
   TODO(GZGavinZhao): add a progress bar to show the fetching progress.

### Query

Query the build order for a list of packages. Even though you can pass any tpath
to it, generally you are always passing a source tpath because it contains the
most information regarding build dependencies.

May fail or output an incorrect order if the dependency graph between the list 
of packages given has cycles.

```bash
autobuild query <tpath> <list-of-packages>
```

Example:
```bash
autobuild query src:../packages rocblas hipblas rocsolver hipsolver rocfft hipfft
```

### Diff

Outputs the changes between two different TPaths.

```bash
autobuild diff <old-tpath> <new-tpath>
```

Example: what packages have I updated locally?
```bash
autobuild diff repo:unstable src:../packages
```

### Push

Push all changes to the build server, in the correct build order.

```bash
autobuild push <old-tpath> <new-tpath>
```

May fail or output an incorrect order if the dependency graph between the list 
of packages given has cycles.

Note: you must already have permissions to push to the build server. By default,
it does a dry-run and you can inspect whether it will be pushing the packages
that you want to push. After you think everything looks fine, you can run the
command with `--dry-run=false` to actually push to the build server.

TODO(GZGavinZhao): add a yes/no dialogue even if `--dry-run=false`.

Example: push my ROCm stack
```bash
autobuild push repo:unstable src:$HOME/solus/work/rocm-6
```
