package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"mor10z/internal/audio"
	"mor10z/internal/library"
	"mor10z/internal/synth"
	"mor10z/internal/ui"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "mor10z:", e)
		os.Exit(1)
	}
}
func run() error {
	config := flag.String("data-dir", "", "state directory (default: XDG_CONFIG_HOME/mor10z)")
	demo := flag.String("demo", "", "render the built-in synth pattern to a WAV file and exit")
	version := flag.Bool("version", false, "show version")
	nullAudio := flag.Bool("null-audio", false, "use mpv's silent audio output for testing")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "mor10z — terminal sound system\n\nUsage: mor10z [options] [music-folder | audio-file | playlist.m3u ...]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *version {
		fmt.Println("mor10z 0.1.0")
		return nil
	}
	if *demo != "" {
		if e := synth.WriteWAV(*demo, synth.Default().Render()); e != nil {
			return e
		}
		fmt.Println("Rendered", *demo)
		return nil
	}
	dir := *config
	if dir == "" {
		base, e := os.UserConfigDir()
		if e != nil {
			return e
		}
		dir = filepath.Join(base, "mor10z")
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	state, e := library.Load(dir)
	if e != nil {
		return e
	}
	for _, arg := range flag.Args() {
		ext := strings.ToLower(filepath.Ext(arg))
		if ext == ".m3u" || ext == ".m3u8" {
			path, e := library.Expand(arg)
			if e != nil {
				return e
			}
			p, e := library.ReadM3U(path)
			if e != nil {
				return e
			}
			state.Playlists = append(state.Playlists, p)
		} else if e := library.AddRoot(&state, arg); e != nil {
			return e
		}
	}
	if len(state.Roots) == 0 && len(flag.Args()) == 0 {
		home, _ := os.UserHomeDir()
		music := filepath.Join(home, "Music")
		if st, e := os.Stat(music); e == nil && st.IsDir() {
			state.Roots = append(state.Roots, music)
		}
	}
	if e = library.Save(dir, state); e != nil {
		return e
	}
	temp, e := os.MkdirTemp("", "mor10z-synth-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(temp)
	player, e := audio.New(*nullAudio)
	if e != nil {
		return e
	}
	defer player.Close()
	player.Command("set_property", "volume", state.Volume)
	_, e = tea.NewProgram(ui.New(player, state, dir, temp), tea.WithAltScreen()).Run()
	return e
}
