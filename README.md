## Usage

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

Note: you must already have permissions to push to the build server. It's
recommended that you run it with the dry-run flag `-n` to inspect the push order
before proceeding. TODO(GZGavinZhao): add a yes/no dialogue.

Example: push my ROCm stack
```bash
autobuild push repo:unstable src:$HOME/solus/work/rocm-6
```
