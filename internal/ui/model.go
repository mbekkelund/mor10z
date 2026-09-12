package ui

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"mor10z/internal/audio"
	"mor10z/internal/library"
	"mor10z/internal/synth"
)

type tickMsg time.Time
type scanMsg struct {
	tracks []library.Track
	err    error
}
type renderedMsg struct {
	path   string
	err    error
	export bool
}
type Model struct {
	viz                                              visualState
	player                                           *audio.Player
	state                                            library.State
	dir, temp                                        string
	tracks                                           []library.Track
	queue                                            []string
	queueAt                                          int
	current                                          string
	tab, cursor, playlist, width, height, frame      int
	helpOffset                                       int
	search, input, prompt, status                    string
	scanning, shuffle, help, synthPlaying, rendering bool
	repeat                                           int
	pattern                                          synth.Pattern
	step                                             int
}

func New(p *audio.Player, s library.State, dir, temp string) Model {
	m := Model{player: p, state: s, dir: dir, temp: temp, width: 100, height: 32, queueAt: -1, pattern: synth.Default(), status: "Ready. f adds a music folder · ? opens the manual"}
	if b, e := os.ReadFile(filepath.Join(dir, "synth.json")); e == nil {
		var pat synth.Pattern
		if json.Unmarshal(b, &pat) == nil && pat.BPM >= 40 && pat.BPM <= 240 && pat.Wave >= 0 && pat.Wave < len(synth.Waves) {
			valid := pat.Cutoff >= 80 && pat.Cutoff <= 16000
			for _, n := range pat.Notes {
				valid = valid && n >= 24 && n <= 96
			}
			if valid {
				m.pattern = pat
			}
		}
	}
	return m
}
func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tickMsg(t) })
}
func scan(roots []string) tea.Cmd {
	copyRoots := append([]string(nil), roots...)
	return func() tea.Msg { t, e := library.Scan(copyRoots); return scanMsg{t, e} }
}
func (m Model) Init() tea.Cmd { return tea.Batch(tick(), scan(m.state.Roots)) }
func (m *Model) save() {
	if e := library.Save(m.dir, m.state); e != nil {
		m.status = "Save failed: " + e.Error()
	}
}
func (m *Model) savePattern() {
	b, e := json.MarshalIndent(m.pattern, "", "  ")
	if e == nil {
		e = os.WriteFile(filepath.Join(m.dir, "synth.json"), b, 0600)
	}
	if e != nil {
		m.status = "Pattern save failed: " + e.Error()
	}
}
func (m Model) visible() []library.Track {
	var src []library.Track
	if m.tab == 1 {
		for _, p := range m.state.Playlists[m.playlist].Paths {
			src = append(src, library.Describe(p))
		}
	} else {
		src = m.tracks
	}
	if m.search == "" {
		return src
	}
	var out []library.Track
	for _, t := range src {
		if strings.Contains(strings.ToLower(t.Title+" "+t.Folder), strings.ToLower(m.search)) {
			out = append(out, t)
		}
	}
	return out
}
func (m *Model) command(args ...any) {
	if e := m.player.Command(args...); e != nil {
		m.status = e.Error()
	}
}
func (m *Model) play(path string) {
	if e := m.player.Load(path, false); e != nil {
		m.status = e.Error()
		return
	}
	m.current = path
	m.synthPlaying = false
	m.status = "Playing · " + library.Describe(path).Title
}
func (m *Model) advance(delta int, auto bool) {
	if len(m.queue) == 0 {
		return
	}
	if auto && m.repeat == 1 {
		m.play(m.queue[m.queueAt])
		return
	}
	n := m.queueAt + delta
	if m.shuffle && delta > 0 && len(m.queue) > 1 {
		n = (m.queueAt + 1 + rand.IntN(len(m.queue)-1)) % len(m.queue)
	}
	if n >= len(m.queue) {
		if m.repeat == 2 || !auto {
			n = 0
		} else {
			m.status = "Queue finished"
			return
		}
	}
	if n < 0 {
		n = len(m.queue) - 1
	}
	m.queueAt = n
	m.play(m.queue[n])
}
func (m *Model) render(export bool) tea.Cmd {
	if m.rendering {
		return nil
	}
	m.rendering = true
	pat := m.pattern
	dir := m.temp
	if export {
		dir = m.dir
	}
	return func() tea.Msg {
		f, e := os.CreateTemp(dir, "mor10z-synth-*.wav")
		if e != nil {
			return renderedMsg{err: e, export: export}
		}
		path := f.Name()
		f.Close()
		e = synth.WriteWAV(path, pat.Render())
		return renderedMsg{path, e, export}
	}
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case visualMsg:
		m.acceptVisual(msg)
	case scanMsg:
		m.scanning = false
		m.tracks = msg.tracks
		if msg.err != nil {
			m.status = "Watch: " + msg.err.Error()
		}
		m.cursor = min(m.cursor, max(0, len(m.visible())-1))
	case renderedMsg:
		m.rendering = false
		if msg.err != nil {
			m.status = msg.err.Error()
		} else if msg.export {
			m.status = "WAV exported · " + msg.path
		} else {
			if e := m.player.Load(msg.path, true); e != nil {
				m.status = e.Error()
			} else {
				m.synthPlaying = true
				m.current = msg.path
				m.status = "Synth running · changes apply with Enter"
			}
		}
	case tickMsg:
		m.frame++
		for {
			select {
			case e := <-m.player.Events:
				if e.Err != nil {
					m.status = e.Err.Error()
				}
				if e.EOF && !m.synthPlaying {
					m.advance(1, true)
				}
			default:
				goto drained
			}
		}
	drained:
		visualCmd := m.updateVisual()
		if m.frame%90 == 0 && !m.scanning {
			m.scanning = true
			return m, tea.Batch(tick(), scan(m.state.Roots), visualCmd)
		}
		return m, tea.Batch(tick(), visualCmd)
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			m.save()
			m.savePattern()
			return m, tea.Quit
		}
		if m.prompt != "" {
			switch key {
			case "esc":
				m.prompt = ""
				m.input = ""
			case "backspace":
				r := []rune(m.input)
				if len(r) > 0 {
					m.input = string(r[:len(r)-1])
				}
				if m.prompt == "Search" {
					m.search = m.input
					m.cursor = 0
				}
			case "enter":
				cmd := m.submit()
				return m, cmd
			default:
				if msg.Type == tea.KeyRunes {
					m.input += string(msg.Runes)
				} else if key == " " {
					m.input += " "
				}
				if m.prompt == "Search" {
					m.search = m.input
					m.cursor = 0
				}
			}
			return m, nil
		}
		if key == "?" {
			m.help = !m.help
			m.helpOffset = 0
			return m, nil
		}
		if m.help {
			if key == "j" || key == "down" {
				m.helpOffset++
			}
			if key == "k" || key == "up" {
				m.helpOffset = max(0, m.helpOffset-1)
			}
			if key == "esc" || key == "q" {
				m.help = false
			}
			return m, nil
		}
		switch key {
		case "q":
			m.save()
			m.savePattern()
			return m, tea.Quit
		case "tab":
			m.tab = (m.tab + 1) % 4
			m.cursor = 0
			m.search = ""
		case "shift+tab":
			m.tab = (m.tab + 3) % 4
			m.cursor = 0
			m.search = ""
		case "1", "2", "3", "4":
			m.tab = int(key[0] - '1')
			m.cursor = 0
			m.search = ""
		case "z":
			m.viz.mode = (m.viz.mode + 1) % 3
		case "f":
			m.prompt = "Folder"
			m.input = ""
		case "P":
			m.prompt = "New playlist"
			m.input = ""
		case "I":
			m.prompt = "Import M3U"
			m.input = ""
		case "n":
			m.advance(1, false)
		case "b":
			if m.player.Snapshot().Position > 3 {
				m.command("seek", 0, "absolute")
			} else {
				m.advance(-1, false)
			}
		case "x":
			m.command("stop")
			m.synthPlaying = false
			m.status = "Stopped"
		case "s":
			m.shuffle = !m.shuffle
		case "r":
			m.repeat = (m.repeat + 1) % 3
		case "v", "V":
			delta := 5
			if key == "V" {
				delta = -5
			}
			m.state.Volume = max(0, min(100, m.state.Volume+delta))
			m.command("set_property", "volume", m.state.Volume)
			m.save()
		case "m":
			m.command("cycle", "mute")
		default:
			if m.tab == 2 {
				cmd := m.synthKey(key)
				return m, cmd
			}
			if m.tab == 3 {
				switch key {
				case " ", "enter":
					m.command("cycle", "pause")
				case "left", "h":
					m.command("seek", -5, "relative")
				case "right", "l":
					m.command("seek", 5, "relative")
				}
				return m, nil
			}
			tracks := m.visible()
			switch key {
			case "j", "down":
				m.cursor = min(max(0, len(tracks)-1), m.cursor+1)
			case "k", "up":
				m.cursor = max(0, m.cursor-1)
			case "pgdown":
				m.cursor = min(max(0, len(tracks)-1), m.cursor+max(1, m.height-17))
			case "pgup":
				m.cursor = max(0, m.cursor-max(1, m.height-17))
			case "home", "g":
				m.cursor = 0
			case "end", "G":
				m.cursor = max(0, len(tracks)-1)
			case "/":
				m.prompt = "Search"
				m.input = m.search
			case "esc":
				m.search = ""
				m.cursor = 0
			case "[", "]":
				d := 1
				if key == "[" {
					d = -1
				}
				m.playlist = (m.playlist + d + len(m.state.Playlists)) % len(m.state.Playlists)
				m.cursor = 0
			case "enter":
				if len(tracks) > 0 {
					m.queue = nil
					for _, t := range tracks {
						m.queue = append(m.queue, t.Path)
					}
					m.queueAt = m.cursor
					m.play(tracks[m.cursor].Path)
				}
			case " ":
				if m.player.Snapshot().Idle {
					if len(tracks) > 0 {
						m.queue = nil
						for _, t := range tracks {
							m.queue = append(m.queue, t.Path)
						}
						m.queueAt = m.cursor
						m.play(tracks[m.cursor].Path)
					}
				} else {
					m.command("cycle", "pause")
				}
			case "right", "l":
				m.command("seek", 5, "relative")
			case "left", "h":
				m.command("seek", -5, "relative")
			case "a":
				if len(tracks) > 0 {
					p := &m.state.Playlists[m.playlist]
					path := tracks[m.cursor].Path
					found := false
					for _, v := range p.Paths {
						if v == path {
							found = true
						}
					}
					if !found {
						p.Paths = append(p.Paths, path)
						m.status = "Added to " + p.Name
						m.save()
					} else {
						m.status = "Already in " + p.Name
					}
				}
			case "d":
				if m.tab == 1 && len(tracks) > 0 {
					p := &m.state.Playlists[m.playlist]
					for i, v := range p.Paths {
						if v == tracks[m.cursor].Path {
							p.Paths = append(p.Paths[:i], p.Paths[i+1:]...)
							break
						}
					}
					m.cursor = min(m.cursor, max(0, len(m.visible())-1))
					m.save()
				}
			case "e":
				path := filepath.Join(m.dir, fmt.Sprintf("playlist-%d.m3u", m.playlist+1))
				if e := library.WriteM3U(path, m.state.Playlists[m.playlist]); e != nil {
					m.status = e.Error()
				} else {
					m.status = "Exported · " + path
				}
			}
		}
	}
	return m, nil
}
func (m *Model) submit() tea.Cmd {
	mode, input := m.prompt, strings.TrimSpace(m.input)
	m.prompt = ""
	m.input = ""
	switch mode {
	case "Search":
		return nil
	case "Folder":
		if input == "" {
			return nil
		}
		if e := library.AddRoot(&m.state, input); e != nil {
			m.status = e.Error()
		} else {
			m.save()
			m.status = "Watching · " + input
			m.scanning = true
			return scan(m.state.Roots)
		}
	case "New playlist":
		if input != "" {
			m.state.Playlists = append(m.state.Playlists, library.Playlist{Name: input, Paths: []string{}})
			m.playlist = len(m.state.Playlists) - 1
			m.tab = 1
			m.cursor = 0
			m.search = ""
			m.save()
			m.status = "Playlist created · " + input
		}
	case "Import M3U":
		path, e := library.Expand(input)
		if e != nil {
			m.status = e.Error()
			return nil
		}
		p, e := library.ReadM3U(path)
		if e != nil {
			m.status = e.Error()
		} else {
			m.state.Playlists = append(m.state.Playlists, p)
			m.playlist = len(m.state.Playlists) - 1
			m.tab = 1
			m.cursor = 0
			m.search = ""
			m.save()
			m.status = fmt.Sprintf("Imported %d tracks", len(p.Paths))
		}
	}
	return nil
}
func (m *Model) synthKey(key string) tea.Cmd {
	switch key {
	case "left", "h":
		m.step = (m.step + 15) % 16
	case "right", "l":
		m.step = (m.step + 1) % 16
	case "up", "k":
		m.pattern.Notes[m.step] = min(96, m.pattern.Notes[m.step]+1)
	case "down", "j":
		m.pattern.Notes[m.step] = max(24, m.pattern.Notes[m.step]-1)
	case " ":
		m.pattern.Enabled[m.step] = !m.pattern.Enabled[m.step]
	case "enter":
		m.savePattern()
		return m.render(false)
	case "w":
		m.pattern.Wave = (m.pattern.Wave + 1) % len(synth.Waves)
	case "+", "=":
		m.pattern.BPM = min(240, m.pattern.BPM+2)
	case "-":
		m.pattern.BPM = max(40, m.pattern.BPM-2)
	case ",":
		m.pattern.Cutoff = max(80, m.pattern.Cutoff/1.25)
	case ".":
		m.pattern.Cutoff = min(16000, m.pattern.Cutoff*1.25)
	case "d":
		m.pattern.Delay = !m.pattern.Delay
	case "[":
		for i, n := range m.pattern.Notes {
			m.pattern.Notes[i] = max(24, n-12)
		}
	case "]":
		for i, n := range m.pattern.Notes {
			m.pattern.Notes[i] = min(96, n+12)
		}
	case "e":
		return m.render(true)
	case "0":
		m.pattern = synth.Default()
	}
	return nil
}
