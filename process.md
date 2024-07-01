autobuild commit: c41630ce9ff6fe516f6e09be055b7ab01d0795d2

getsolus/packages commit: d10fd470f2bdf13ff938fb934c85e711c8fac678

In general, a dependency can be ignored when solving build order if any of the
following satisfies:
- It's only for testing.
- It's only for generating documentation.
- It's only a rundep. This is a little special, because in general we want to
  keep the rundep information as much as possible, so we should only ignore a
  rundep when really necessary. 

  To see why, consider the case where B is a rundep of A, and C depends on A. If
  we ignore B, then when C is built, B may not be rebuilt, yet it is possible
  that we require a rebuilt for B to be able to _run_ A. A future autobuild
  version may be smarter and selectively take `rundeps` into account, but for
  now `rundeps` is always considered in resolving build order and we manually
  ignore rundeps if needed.

## Run 1:

```
> go run . query src:../packages samba ldb tdb talloc tevent
 ✗  Duplicate provider for pkgconfig(gmock) from gtest, currently antlr4-cpp-runtime
 ✗  Duplicate provider for pkgconfig(gmock_main) from gtest, currently antlr4-cpp-runtime
 ✗  Duplicate provider for pkgconfig(gtest) from gtest, currently antlr4-cpp-runtime
 ✗  Duplicate provider for pkgconfig(gtest_main) from gtest, currently antlr4-cpp-runtime
 ✗  Duplicate provider for rocm-hip from rocm-hip, currently rocm-clr
 ✗  Duplicate provider for rocm-opencl from rocm-opencl, currently rocm-clr
 🗸  Successfully parsed state!
 🗸  Found all requested packages in state!
 🗸  Successfully built dependency graph!
 🗲  Cycle 13: cifs-utils ldb tdb ffmpeg samba gvfs
 🗲  Dependency chain that led to this cycle:
cifs-utils -> gvfs -> gnome-session -> gnome-shell -> gdm -> xorg-xwayland -> xorg-server -> linux-driver-management -> mesalib -> qt5-base -> doxygen -> llvm -> libeconf -> util-linux -
> e2fsprogs -> libtirpc -> tdb -> ldb -> cifs-utils
 🕱  Failed to get topological sort order: topological sort cannot be computed on graph with cycles
```

Note the edge `doxygen -> llvm`. The arrow means that when `doxygen` is rebuild,
`llvm` should be rebuilt as well. Clearly, this is unnecessary. Therefore, we
should add the following to `package/l/llvm/autobuild.yml`:
```yml
solver:
  ignore:
    - doxygen
```

## Run 2

```
ffmpeg -> pipewire -> openal-soft -> qt5-multimedia -> python3-qt5 -> pulseaudio -> qt6-base -> gpgme -> samba -> ffmpeg
```

Determine how to break this cycle is a bit hard.
There aren't any dependency relations that look sus at
first glance. However, notice that `python3-qt5 -> pulseaudio -> qt6-base`. This
means that a rebuild in `python3-qt5` will eventually trigger a rebuild of
`qt6-base`! This doesn't look right. `qt6-base` shouldn't depend on anything
related to Qt 5. We check `pulseaudio`, and indeed, upon inspecting the recipe
of `pulseaudio`, we see
that `python3-qt5` is pulled in only because it is
part of the rundep of its subpackage
`pulseaudio-equalizer`. Clearly, we can safely ignore `python3-qt5` when
building `pulseaduio`, so we put the following in
`package/p/pulseaudio/autobuild.yml`:
```yml
solver:
  ignore:
    - python3-qt5
```

## Run 3

```
cifs-utils -> gvfs -> gnome-control-center -> gnome-shell -> gdm -> xorg-xwayland -> xorg-server -> linux-driver-management -> mesalib -> qt5-base -> doxygen -> tpm2-tss -> systemd -> ut
il-linux -> e2fsprogs -> libtirpc -> tdb -> ldb -> cifs-utils
```

We have our friend `doxygen` again! So just like for `llvm`, we put the
following in `packages/t/tpm2-tss/autobuild.yml`:
```yml
solver:
  ignore:
    - doxygen
```

## Run 4

```
gnome-control-center -> gnome-shell -> gdm -> xorg-xwayland -> xorg-server -> linux-driver-management -> mesalib -> qt5-base -> graphviz -> vala -> libsecret -> git -> llvm-bolt -> zlib 
-> openssl -> python3 -> libcap-ng -> cifs-utils -> gnome-control-center
```

The first sus relation we notice is `git -> llvm-bolt`. A rebuild in `git`
shouldn't really trigger a rebuild of `llvm-bolt`, and upon inspection of
`llvm-bolt`'s `package.yml`, we see that the only reason `git` is in `builddeps`
is probably because of build system requirements, so we can safely this
dependency as well. Again, we write the following at
`packages/l/llvm-bolt/autobuild.yml`:
```yml
solver:
  ignore:
    - git
```

## Run 5

```
cifs-utils -> gvfs -> gnome-control-center -> gnome-shell -> gdm -> xorg-xwayland -> xorg-server -> linux-driver-management -> mesalib -> qt5-base -> graphviz -> vala -> libsecret -> sub
version -> py -> python-pytest -> python-jinja -> systemd -> util-linux -> e2fsprogs -> libtirpc -> tdb -> ldb -> cifs-utils
```

Here, we enter our classic `python-pytest -> python-X` issue. A rebuild in
`python-pytest` shouldn't trigger a rebuild of `python-jinja`. Indeed, upon
inspection of `python-jinja`'s `package.yml`, we see that 
```yml
builddeps  :
    - python-markupsafe # check
    - python-pytest     # check
```

Therefore, we can move `python-markupsafe` and `python-pytest` into
`packages/py/python-jinja/autobuild.yml`:
```yml
solver:
  ignore:
    - python-markupsafe
    - python-pytest
```

## Run 6

```
gnome-control-center -> gnome-shell -> gdm -> xorg-xwayland -> xorg-server -> linux-driver-management -> mesalib -> qt5-base -> graphviz -> vala -> libsecret -> subversion -> py -> pytho
n-pytest -> python-requests -> python-sphinx -> python-recommonmark -> python-docutils -> cifs-utils -> gnome-control-center
```

Again, we have `python-pytest -> python-requests`. Upon inspection of
`python-requests`'s `package.yml`, we are resonably confident that
`python-pytest` is only used for testing purposes, so we can put the following
in `packages/py/python-requests/autobuild.yml`:
```yml
solver:
  ignore:
    - python-pytest
```

## Run 7

```
cifs-utils -> gvfs -> gnome-control-center -> gnome-shell -> gdm -> xorg-xwayland -> xorg-server -> linux-driver-management -> mesalib -> qt5-base -> graphviz -> vala -> libsecret -> sub
version -> py -> python-pytest -> python-chardet -> python-requests -> python-sphinx -> python-recommonmark -> llvm -> libeconf -> util-linux -> e2fsprogs -> libtirpc -> tdb -> ldb -> ci
fs-utils
```

Same issue, `python-pytest -> python-chardet`. Put the following into
`packages/py/python-chardet/autobuild.yml`:
```yml
solver:
  ignore:
    - python-pytest
```

## Run 8

```yml
cifs-utils -> gvfs -> gnome-session -> gnome-shell -> gdm -> xorg-xwayland -> xorg-server -> linux-driver-management -> mesalib -> qt5-base -> graphviz -> vala -> libsecret -> subversion
 -> py -> python-pytest -> python-wheel -> python-importlib-metadata -> python-sphinx -> python-recommonmark -> llvm -> zstd -> gcc -> e2fsprogs -> libtirpc -> tdb -> ldb -> cifs-utils
```

Same issue, `python-pytest -> python-wheel`. Put the following into
`packages/py/python-wheel/autobuild.yml`:
```yml
solver:
  ignore:
    - python-pytest
```

## Run 9

```
ffmpeg -> openjdk-11 -> jtreg -> openjdk-17 -> fop -> asciidoc -> liblttng-ust -> qt6-base -> gpgme -> samba -> ffmpeg
```

There are several options here, `openjdk-11 -> jtreg -> openjdk-17` because it
doesn't make sense for a
rebuild of Java 11 to trigger a rebuild of Java 17, and `asciidoc
-> liblttng-ust`, because `asciidoc` looks like a dependency only used for
documentation generation. In this case, we choose `openjdk-11 -> jtreg ->
openjdk-17`, because if you check what package `jtreg` is, we see that it's 
`Test harness for testing the JDK`, which means it's clearly for testing, so we
can ignore this by putting the following in
`packages/o/openjdk-17/autobuild.yml`:
```yml
solver:
  ignore:
    - jtreg
```

## Run 10

```
ffmpeg -> pipewire -> gnome-shell -> gdm -> xorg-xwayland -> xorg-server -> linux-driver-management -> mesalib -> qt6-base -> gpgme -> samba -> ffmpeg
```

For this one, the sus relation is `linux-driver-management -> mesalib`. It
doesn't feel right for such a base-level package `mesalib` to be dependent on a
mere application, and indeed, `linux-driver-management` is only pulled in
because it's a rundep of `mesalib`, so we can safely ignore
`linux-driver-management` by putting the
following in `packages/m/mesalib/autobuild.yml`:
```yml
solver:
  ignore:
    - linux-driver-management
```

## Run 11

```
 🗸  Build order: talloc tdb tevent ldb samba 
```

YAY!
