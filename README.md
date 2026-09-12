# mor10z

**Terminal sound system.** En neonfarget musikkspiller og liten synth-stasjon, skrevet i Go.

- MP3, WAV, FLAC, OGG, M4A og Opus gjennom en privat mpv-prosess.
- Play/pause, stop, neste/forrige, spoling, volum, mute, shuffle og repeat.
- Søkbart bibliotek og rekursiv mappeovervåking hvert tredje sekund.
- Navngitte spillelister med automatisk lagring og M3U/M3U8-import, M3U-eksport.
- 16-stegs synthesizer: saw, square, sine, triangle, suboscillator, envelope,
  low-pass-filter og delay. Juster pitch, tempo og gates; eksporter til WAV.
- Responsivt TUI med bibliotek, spillelister og synth-lab. Ingen nettjenester.

## Start

Last ned Arch/Omarchy-pakken fra [utgivelsene](https://github.com/mbekkelund/mor10z/releases).
Installer den nedlastede pakken med `sudo pacman -U ./mor10z-0.1.0-1-x86_64.pkg.tar.zst`.
Åpne deretter **mor10z** fra appmenyen eller terminalen:

```bash
mor10z
mor10z ~/Music "/sti/til/flere låter"
mor10z /sti/til/spilleliste.m3u
```

Start uten argumenter for å åpne forrige bibliotek. Første gang brukes `~/Music`
hvis mappen finnes. Trykk `f` for å legge til en mappe fra spilleren, eller `3` og
`Enter` for å høre den innebygde synth-sekvensen med en gang.

Krever **mpv** på PATH og Linux/macOS med en UTF-8-terminal på minst 48 × 18.
110 × 36 eller større anbefales for hele synth-panelet og sidekolonnen.
Lydmotoren ignorerer din vanlige
mpv-konfigurasjon og bruker en egen IPC-socket i en privat midlertidig mappe.

## Tastatur

| Tast | Handling |
| --- | --- |
| `Tab`, `Shift+Tab`, `1`–`4` | Bytt panel |
| `Enter` | Spill valgt låt og bruk den viste listen som avspillingskø |
| `Space` | Play/pause i bibliotek og spillelister |
| `x` | Stop |
| `n` / `b` | Neste / forrige (eller start låten på nytt etter 3 sekunder) |
| `←` / `→` | Spol 5 sekunder i bibliotek/spillelister |
| `v` / `V` | Volum opp / ned |
| `m` | Mute |
| `s` | Shuffle av/på |
| `r` | Repeat av → låt → alle |
| `j` / `k`, `↓` / `↑` | Velg låt |
| `PgUp` / `PgDn`, `Home` / `End` | Bla i biblioteket |
| `/` | Søk mens du skriver; Enter avslutter søkefeltet, Esc i listen nullstiller |
| `f` | Legg til mappe eller lydfil |
| `[` / `]` | Velg målspilleliste / bytt spilleliste |
| `a` | Legg valgt låt i målspillelisten |
| `P` | Opprett spilleliste |
| `d` | Fjern valgt låt fra spillelisten (lydfilen beholdes) |
| `I` | Importer lokal M3U/M3U8 |
| `e` | Eksporter målspillelisten til M3U |
| `?` | Hjelp |
| `q`, `Ctrl+C` | Lagre og avslutt |

Store og små bokstaver er ulike: `P`, `I` og `V` betyr Shift+tasten.

## Synth-lab

Trykk `3`, deretter `Enter`. Synthesizeren lager sin egen lyd i Go og spiller
sekvensen i loop. Synth erstatter musikkavspillingen; `1` og Enter på en låt
bytter tilbake til musikk. Dette er en step-sequencer, ikke et MIDI-instrument.

| Tast | Synth-handling |
| --- | --- |
| `←` / `→`, `h` / `l` | Velg steg |
| `↑` / `↓`, `k` / `j` | Endre tone med en halvtone |
| `Space` | Slå steg av/på |
| `Enter` | Bruk endringene og spill sekvensen i loop |
| `w` | Bytt oscillator |
| `+` / `-` | Tempo, 40–240 BPM |
| `,` / `.` | Senk / øk filterfrekvensen |
| `d` | Delay av/på |
| `[` / `]` | Flytt hele sekvensen en oktav |
| `e` | Eksporter én loop som 44,1 kHz / 16-bit mono WAV |
| `0` | Tilbakestill mønster |
| `x` | Stopp lyd |

Saw og square bruker polyBLEP for å redusere aliasing. Filteret er et enkelt
énpolet lavpassfilter. Delay varmes opp over flere runder før loopen eksporteres.
Oscillatortegningen viser valgt bølgeform; den er ikke en frekvensanalyse av musikken.
Synth-mønster lagres ved Enter og avslutning.

Du kan også generere demoen uten mpv eller terminalgrensesnitt:

```bash
./bin/mor10z --demo /tmp/mor10z-demo.wav
```

## Lagring og mappeovervåking

Standard: `$XDG_CONFIG_HOME/mor10z`, ellers `~/.config/mor10z`.
Bruk `--data-dir /annen/mappe` for en separat profil.

- `state.json`: mapper, spillelister og volum; skrives atomisk.
- `synth.json`: siste synth-mønster.
- `playlist-N.m3u`: eksportert spilleliste. Ny eksport av samme liste erstatter filen.
- `mor10z-synth-*.wav`: unikt navngitte synth-eksporter.

Mappeskanningen finner nye og slettede lydfiler, også i undermapper. Skjulte
undermapper hoppes over, og symlink-mapper følges ikke. Biblioteket sorteres etter
filnavn; spillertittelen kommer fra mpv. Store bibliotek kan gjøre skanning tregere,
men skanningen kjøres utenfor UI-tråden. En startet avspillingskø er et øyeblikksbilde;
nyoppdagede filer blir med når du starter en ny liste med Enter. Spillelister beholder
referanser til filer som flyttes eller slettes, og mpv melder feil ved avspilling.
M3U-import støtter lokale filstier, ikke strømmelenker.

## Bygg og test

For en installerbar Arch/Omarchy-pakke, se [pakkeveiledningen](packaging/README.md).

Go 1.25+ og mpv. Makefile finner også Go-kompilatoren som er hentet til
`../.tools/go` på denne maskinen.

```bash
make build
make test
make check  # vet + race + ekte MP3/WAV-integrasjon med mpv --ao=null; krever ffmpeg
```

Integrasjonstesten bruker lydløs utgang og sjekker lasting av MP3/WAV, pause,
spoling, volum, stopp og end-of-file. Enhetstestene dekker filendringer, spillelister,
M3U, tilstand, synth-lyd/WAV og TUI-navigasjon/layout.

Bygget på [Bubble Tea](https://pkg.go.dev/github.com/charmbracelet/bubbletea),
[Lip Gloss](https://github.com/charmbracelet/lipgloss) og
[mpvs JSON IPC](https://mpv.io/manual/stable/#json-ipc).

## Visualizer

Trykk `4` mens en låt spiller. `z` bytter mellom neonfarget frekvensspektrum,
oscilloskop og rullende spektrogram. Space pauser, piltastene spoler.
Synth-panelet har også bølgeformvisning. Start synth med `3`, Enter, og trykk
`4` for stor visualizer.

Analysen bruker ffmpeg til å lese korte utsnitt av
lydfilen og beregner FFT i Go. Visningen følger mpv sin avspillingsposisjon,
inkludert spoling og looping. Den viser signalet i kildefilen før volum/mute;
den tar ikke opp mikrofonen eller annen systemlyd. Manglende ffmpeg påvirker
ikke musikkavspillingen.

## Lisens

[MIT](LICENSE). Copyright © 2026 Morten Bekkelund.
