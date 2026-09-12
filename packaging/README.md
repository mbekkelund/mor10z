# Arch / Omarchy

Create a source archive with Go dependencies included and a package recipe
with a SHA-256 checksum:

```bash
make dist
cd dist
makepkg -s
sudo pacman -U ./mor10z-0.1.0-1-*.pkg.tar.zst
```

The package installs `mor10z` in `/usr/bin` and the desktop entry in
`/usr/share/applications`. It opens in the default terminal from the application launcher.
Playback requires `mpv`. Install `ffmpeg` for the visualizer.
Building requires Go 1.25 or newer; the tests also require ffmpeg.
The package build uses the bundled Go dependencies without network access.

## Publishing

Source code: https://github.com/mbekkelund/mor10z — MIT license.

`make dist` creates a source archive with vendored Go dependencies and a
`dist/PKGBUILD` with the correct checksum and public download URL.
Upload this archive as a release asset; GitHub's automatic source archives
do not include the vendored dependencies.

For AUR: generate `.SRCINFO` by running `makepkg --printsrcinfo > .SRCINFO`
from `dist`, then publish only `PKGBUILD` and `.SRCINFO` to the AUR repository.
AUR requires a separate account and a registered SSH key. The package is not
available through AUR until this submission is complete.

The package has been built and tested on x86_64. aarch64 is listed as a build
target but has not been tested. A clean Arch build environment and namcap
have not been verified locally.

Guide: https://wiki.archlinux.org/title/AUR_submission_guidelines
