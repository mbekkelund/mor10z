package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"mor10z/internal/audio"
	"mor10z/internal/library"
	"strings"
	"testing"
	"time"
)

func testModel(t *testing.T) Model {
	t.Helper()
	s, e := library.Load(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return New(&audio.Player{Events: make(chan audio.Event, 8)}, s, t.TempDir(), t.TempDir())
}
func press(m Model, key string) Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	if key == "enter" {
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	}
	if key == "esc" {
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	}
	out, _ := m.Update(msg)
	return out.(Model)
}
func TestSynthEditsPersistInModel(t *testing.T) {
	m := testModel(t)
	m = press(m, "3")
	wave := m.pattern.Wave
	m = press(m, "w")
	if m.pattern.Wave == wave {
		t.Fatal("wave edit lost")
	}
	note := m.pattern.Notes[0]
	m = press(m, "k")
	if m.pattern.Notes[0] != note+1 {
		t.Fatal("pitch edit lost")
	}
	gate := m.pattern.Enabled[0]
	m = press(m, " ")
	if m.pattern.Enabled[0] == gate {
		t.Fatal("gate edit lost")
	}
}
func TestPlaylistsAndLiveSearch(t *testing.T) {
	m := testModel(t)
	m.tracks = []library.Track{library.Describe("/music/Alpha.mp3"), library.Describe("/music/Beta.wav")}
	m = press(m, "P")
	m = press(m, "Night drive")
	m = press(m, "enter")
	if len(m.state.Playlists) != 2 || m.state.Playlists[1].Name != "Night drive" {
		t.Fatal("create playlist")
	}
	m = press(m, "1")
	m = press(m, "/")
	m = press(m, "beta")
	m = press(m, "enter")
	if len(m.visible()) != 1 {
		t.Fatal("search")
	}
	m = press(m, "a")
	m = press(m, "a")
	if len(m.state.Playlists[1].Paths) != 1 {
		t.Fatal("dedup add")
	}
	m = press(m, "2")
	m = press(m, "d")
	if len(m.state.Playlists[1].Paths) != 0 {
		t.Fatal("remove")
	}
	s, e := library.Load(m.dir)
	if e != nil || len(s.Playlists[1].Paths) != 0 {
		t.Fatal("persist")
	}
}
func TestViewFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{48, 18}, {80, 24}, {110, 36}, {150, 50}} {
		for tab := 0; tab < 4; tab++ {
			m := testModel(t)
			m.width = size[0]
			m.height = size[1]
			m.tab = tab
			m.tracks = []library.Track{library.Describe("/a/夜の音楽 🎧.mp3")}
			view := m.View()
			if lipgloss.Height(view) > m.height {
				t.Fatalf("height %dx%d tab %d: %d", m.width, m.height, tab, lipgloss.Height(view))
			}
			for _, line := range strings.Split(view, "\n") {
				if lipgloss.Width(line) > m.width {
					t.Fatalf("overflow %dx%d tab %d: width %d", m.width, m.height, tab, lipgloss.Width(line))
				}
			}
		}
	}
}
func TestWatchSchedulesRescan(t *testing.T) {
	m := testModel(t)
	m.frame = 89
	next, cmd := m.Update(tickMsg(time.Now()))
	if !next.(Model).scanning || cmd == nil {
		t.Fatal("watch did not schedule")
	}
	next, _ = next.Update(scanMsg{tracks: []library.Track{library.Describe("/new.wav")}})
	if len(next.(Model).tracks) != 1 || next.(Model).scanning {
		t.Fatal("watch result not applied")
	}
}

func TestVisualModesAndSmallTerminal(t *testing.T) {
	m := testModel(t)
	m = press(m, "4")
	if m.tab != 3 {
		t.Fatal("visualizer shortcut")
	}
	for mode := 0; mode < 3; mode++ {
		if m.viz.mode != mode {
			t.Fatal("visual mode cycle")
		}
		for _, size := range [][2]int{{48, 18}, {80, 24}, {110, 36}} {
			m.width = size[0]
			m.height = size[1]
			for i := range m.viz.frame.Spectrum {
				m.viz.frame.Spectrum[i] = float64(i) / 48
			}
			view := m.View()
			if !strings.Contains(view, "4 VISUALS") {
				t.Fatal("visual tab hidden")
			}
			if lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
				t.Fatal("visualizer overflows terminal")
			}
		}
		m = press(m, "z")
	}
}
