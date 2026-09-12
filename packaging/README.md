# Arch / Omarchy

Lag et kildearkiv med Go-avhengighetene inkludert, og en lokal byggeoppskrift
med SHA-256-kontrollsum:

```bash
make dist
cd dist
makepkg -s
sudo pacman -U ./mor10z-0.1.0-1-*.pkg.tar.zst
```

Pakken installerer `mor10z` i `/usr/bin` og appstarteren i
`/usr/share/applications`. Den åpnes i standardterminalen fra appmenyen.
`mpv` kreves for avspilling. Installer `ffmpeg` for visualizeren.
Byggingen trenger Go 1.25 eller nyere; testene trenger også ffmpeg.
Selve pakkebyggingen bruker de medfølgende Go-avhengighetene uten nettverk.

## Publisering

Kildekode: https://github.com/mbekkelund/mor10z — MIT-lisens.

`make dist` lager et kildearkiv med vendorerte Go-avhengigheter og en
`dist/PKGBUILD` med korrekt kontrollsum og offentlig nedlastingsadresse.
Last opp dette arkivet som release-vedlegg; GitHubs automatiske kildearkiv
inneholder ikke de vendorerte avhengighetene.

For AUR: generer `.SRCINFO` med `makepkg --printsrcinfo > .SRCINFO` fra
`dist`, og publiser bare `PKGBUILD` og `.SRCINFO` i AUR-repoet.
AUR krever egen konto og registrert SSH-nøkkel. Pakken er ikke tilgjengelig
via AUR før denne innsendingen er fullført.

Pakken er bygget og testet på x86_64. aarch64 er oppført som byggemål,
men er ikke testet. Rent Arch-byggemiljø og namcap er ikke verifisert lokalt.

Veiledning: https://wiki.archlinux.org/title/AUR_submission_guidelines
