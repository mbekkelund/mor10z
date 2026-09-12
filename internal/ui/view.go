package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"mor10z/internal/synth"
)

var (
	cyan   = lipgloss.Color("#49F2D2")
	pink   = lipgloss.Color("#FF5DA2")
	purple = lipgloss.Color("#A99BFF")
	dim    = lipgloss.Color("#777F9C")
	white  = lipgloss.Color("#E9ECFF")
	bg     = lipgloss.Color("#111525")
	accent = lipgloss.NewStyle().Foreground(cyan)
	muted  = lipgloss.NewStyle().Foreground(dim)
	bright = lipgloss.NewStyle().Foreground(white)
	rose   = lipgloss.NewStyle().Foreground(pink)
)

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, s)
}
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(clean(s))
}
func clock(v float64) string { n := max(0, int(v)); return fmt.Sprintf("%02d:%02d", n/60, n%60) }
func (m Model) View() string {
	if m.width < 48 || m.height < 18 {
		return accent.Render("mor10z") + "\nGive me at least 48 × 18 terminal cells.\nResize your terminal · q quits\n"
	}
	w := min(m.width-4, 124)
	bodyH := max(5, m.height-16)
	logo := lipgloss.NewStyle().Bold(true).Foreground(cyan).Render("m o r 1 0 z")
	header := logo + muted.Render("  /  TERMINAL SOUND SYSTEM")
	right := rose.Render("● LIVE") + muted.Render(fmt.Sprintf("  %d TRACKS · %d WATCHED", len(m.tracks), len(m.state.Roots)))
	if w > 80 {
		header += strings.Repeat(" ", max(1, w-lipgloss.Width(header)-lipgloss.Width(right))) + right
	}
	var tabs []string
	labels := []string{"1 LIBRARY", "2 PLAYLISTS", "3 SYNTH LAB", "4 VISUALS"}
	pad := 1
	if w < 64 {
		labels = []string{"1 MUSIC", "2 LISTS", "3 SYNTH", "4 VISUALS"}
		pad = 0
	}
	for i, label := range labels {
		st := lipgloss.NewStyle().Padding(0, pad).Foreground(dim)
		if i == m.tab {
			st = st.Background(cyan).Foreground(bg).Bold(true)
		}
		tabs = append(tabs, st.Render(label))
	}
	navigation := strings.Join(tabs, " ")
	var body string
	if m.help {
		body = m.helpView(w, bodyH)
	} else if m.tab == 3 {
		body = m.visualView(w, bodyH)
	} else if m.tab == 2 {
		body = m.synthView(w, bodyH)
	} else {
		body = m.libraryView(w, bodyH)
	}
	snap := m.player.Snapshot()
	state := "■ STOPPED"
	if !snap.Idle {
		state = "▶ PLAYING"
		if snap.Paused {
			state = "Ⅱ PAUSED"
		}
	}
	title := snap.Title
	if m.synthPlaying {
		title = "MOR10Z / " + synth.Waves[m.pattern.Wave] + " SESSION"
	}
	if title == "" {
		title = "Pick a track. Find your frequency."
	}
	transport := accent.Render(state) + "  " + bright.Bold(true).Render(fit(title, w-15))
	progressW := max(8, w-18)
	ratio := 0.0
	if snap.Duration > 0 {
		ratio = math.Max(0, math.Min(1, snap.Position/snap.Duration))
	}
	filled := int(ratio * float64(progressW))
	progress := accent.Render(strings.Repeat("━", filled)) + muted.Render(strings.Repeat("─", progressW-filled)) + muted.Render("  "+clock(snap.Position)+" / "+clock(snap.Duration))
	mode := []string{"OFF", "TRACK", "ALL"}[m.repeat]
	shuf := "OFF"
	if m.shuffle {
		shuf = "ON"
	}
	info := muted.Render(fmt.Sprintf("VOL %3d%%   SHUFFLE %s   REPEAT %s   QUEUE %d/%d", m.state.Volume, shuf, mode, max(0, m.queueAt+1), len(m.queue)))
	status := muted.Render(fit(m.status, w))
	if m.prompt != "" {
		status = accent.Render(m.prompt+" ❯ ") + bright.Render(fit(m.input+"█", w-len(m.prompt)-3))
	}
	keys := "space play/pause  n/b next/back  x stop  v/V volume  / search  ? help  q quit"
	if m.tab == 2 {
		keys = "←→ step  ↑↓ pitch  space gate  enter play  w wave  +/- bpm  e export  ? help"
	}
	if m.tab == 3 {
		keys = "z visual style  space pause  ←→ seek  n/b next/back  v/V volume  q quit"
	}
	lines := []string{header, "", navigation, "", body, "", transport, progress, info, "", status, muted.Render(fit(keys, w))}
	for i := range lines {
		lines[i] = lipgloss.NewStyle().MaxWidth(w).Render(lines[i])
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(lines, "\n"))
}
func (m Model) libraryView(w, h int) string {
	listW := w
	sidebar := w >= 90
	if sidebar {
		listW = w - 29
	}
	title := "YOUR COLLECTION"
	if m.tab == 1 {
		title = "PLAYLIST / " + m.state.Playlists[m.playlist].Name
	}
	if m.search != "" {
		title += "  / " + m.search
	}
	lines := []string{accent.Bold(true).Render(fit(title, listW)), muted.Render(strings.Repeat("─", listW))}
	tracks := m.visible()
	rows := max(1, h-3)
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	if len(tracks) == 0 {
		empty := []string{"", "    Nothing here. Yet.", "", "    f  Add a music folder", "    I  Import an M3U playlist", "    3  Make noise in the synth lab"}
		if m.search != "" {
			empty = []string{"", "    No matching tracks. Esc clears search."}
		}
		for _, s := range empty {
			if len(lines) < h-1 {
				lines = append(lines, muted.Render(fit(s, listW)))
			}
		}
	}
	for i := start; i < len(tracks) && i < start+rows; i++ {
		t := tracks[i]
		marker := " "
		if t.Path == m.current && !m.player.Snapshot().Idle {
			marker = "▶"
		}
		label := fmt.Sprintf(" %s %03d  %s", marker, i+1, t.Title)
		suffix := t.Format
		avail := listW - len(suffix) - 2
		label = fit(label, avail)
		label += strings.Repeat(" ", max(1, listW-lipgloss.Width(label)-len(suffix)-1)) + suffix + " "
		st := bright
		if i == m.cursor {
			st = lipgloss.NewStyle().Background(lipgloss.Color("#233647")).Foreground(cyan).Bold(true)
		} else if marker == "▶" {
			st = accent
		}
		lines = append(lines, st.Render(label))
	}
	for len(lines) < h-1 {
		lines = append(lines, strings.Repeat(" ", listW))
	}
	lines = append(lines, muted.Render(fit(fmt.Sprintf("%d tracks   ·   target: %s   [ / ] switch", len(tracks), m.state.Playlists[m.playlist].Name), listW)))
	left := lipgloss.NewStyle().Width(listW).Render(strings.Join(lines, "\n"))
	if !sidebar {
		return left
	}
	side := []string{rose.Bold(true).Render("ON YOUR RADAR"), muted.Render(strings.Repeat("─", 25)), "", bright.Render("PLAYLISTS")}
	for i, p := range m.state.Playlists {
		if len(side) > h-9 {
			break
		}
		mark := "  "
		if i == m.playlist {
			mark = "› "
		}
		side = append(side, accent.Render(fit(fmt.Sprintf("%s%s (%d)", mark, p.Name, len(p.Paths)), 25)))
	}
	side = append(side, "", bright.Render("WATCH FOLDERS"))
	for _, r := range m.state.Roots {
		if len(side) > h-4 {
			break
		}
		side = append(side, muted.Render(fit("↳ "+filepath.Base(r), 25)))
	}
	if len(m.state.Roots) == 0 {
		side = append(side, muted.Render("f to connect a folder"))
	}
	side = append(side, "", muted.Render("a add track · P new list"), muted.Render("e export · I import"))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "    ", lipgloss.NewStyle().Width(25).Height(h).MaxHeight(h).Render(strings.Join(side, "\n")))
}
func (m Model) synthView(w, h int) string {
	p := m.pattern
	delay := "DRY"
	if p.Delay {
		delay = "DOTTED DELAY"
	}
	if h < 14 {
		lines := []string{
			rose.Bold(true).Render(fmt.Sprintf("SYNTH LAB / STEP %02d", m.step+1)),
			accent.Render(fmt.Sprintf("%s · %d BPM · %.0f Hz · %s", synth.Waves[p.Wave], p.BPM, p.Cutoff, delay)),
		}
		for row := 0; row < 2; row++ {
			line := muted.Render(fmt.Sprintf("%02d–%02d  ", row*8+1, row*8+8))
			for col := 0; col < 8; col++ {
				i := row*8 + col
				label := "·"
				if p.Enabled[i] {
					label = synth.Note(p.Notes[i])
				}
				st := lipgloss.NewStyle().Width(4).Foreground(cyan)
				if i == m.step {
					st = st.Background(purple).Foreground(bg).Bold(true)
				}
				line += st.Render(label)
			}
			lines = append(lines, line)
		}
		if h >= 7 {
			lines = append(lines, m.synthScope(w, h-5)...)
		}
		lines = append(lines, muted.Render("Enter applies edits · 4 visuals · e exports WAV"))
		for len(lines) < h {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}
	lines := []string{rose.Bold(true).Render("SYNTH LAB") + muted.Render("  /  16-STEP MONOPHONIC SEQUENCER"), "", accent.Render(fmt.Sprintf("%s   %d BPM   LPF %.0f Hz   %s", synth.Waves[p.Wave], p.BPM, p.Cutoff, delay)), ""}
	cols := 16
	if w < 100 {
		cols = 8
	}
	cellW := min(7, w/cols)
	playing := -1
	if m.synthPlaying && !m.player.Snapshot().Paused && !m.player.Snapshot().Idle {
		playing = int(m.player.Snapshot().Position*float64(p.BPM)/60*4) % 16
	}
	for row := 0; row < 16/cols; row++ {
		var nums, notes, gates []string
		for c := 0; c < cols; c++ {
			i := row*cols + c
			st := lipgloss.NewStyle().Width(cellW - 1).Align(lipgloss.Center).Foreground(dim)
			if i == m.step {
				st = st.Background(purple).Foreground(bg).Bold(true)
			}
			nums = append(nums, st.Render(fmt.Sprintf("%02d", i+1)))
			note := " · "
			if p.Enabled[i] {
				note = synth.Note(p.Notes[i])
			}
			ns := lipgloss.NewStyle().Width(cellW - 1).Align(lipgloss.Center).Foreground(cyan)
			if i == playing {
				ns = ns.Background(cyan).Foreground(bg)
			}
			notes = append(notes, ns.Render(note))
			bar := "░░░"
			if p.Enabled[i] {
				bar = "━━━"
			}
			gates = append(gates, ns.Render(bar))
		}
		lines = append(lines, strings.Join(nums, " "), strings.Join(notes, " "), strings.Join(gates, " "), "")
	}
	if h >= 18 {
		label := "OSCILLATOR PREVIEW"
		if m.synthPlaying {
			label = "LIVE SYNTH WAVEFORM · 4 for full visualizer"
		}
		lines = append(lines, muted.Render(label))
		lines = append(lines, m.synthScope(w, 4)...)
	}
	lines = append(lines, muted.Render("w waveform   ,/. filter   d delay   [/] octave   0 reset"))
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	for i, s := range lines {
		lines[i] = lipgloss.NewStyle().MaxWidth(w).Render(s)
	}
	return strings.Join(lines, "\n")
}
func (m Model) helpView(w, h int) string {
	lines := []string{accent.Bold(true).Render("THE CONTROLS  /  ? OR ESC TO CLOSE"), "", "GLOBAL     tab / 1–4 panels · q quit · f add watched folder", "TRANSPORT  space play/pause · x stop · n next · b back/restart", "           ←/→ seek 5s · v/V volume up/down · m mute", "           s shuffle · r repeat off/track/all", "LIBRARY    j/k or ↑/↓ select · enter play selection · / search", "           a add to target playlist · [/] change target", "PLAYLIST   P create · d remove track · I import M3U · e export", "VISUALS    4 open · z spectrum/scope/waterfall · space pause", "SYNTH      ←/→ select step · ↑/↓ pitch · space toggle note", "           enter render & loop · x stop · w waveform", "           +/- BPM · ,/. low-pass filter · d delay · [/] octave", "           e export WAV · 0 reset factory pattern", "", "Folders scan recursively every 3 seconds. Lists save automatically.", "Synth edits apply on Enter. Synth replaces music playback."}
	wrapped := strings.Split(lipgloss.NewStyle().Width(w).Render(strings.Join(lines[2:], "\n")), "\n")
	start := min(m.helpOffset, max(0, len(wrapped)-(h-1)))
	lines = append([]string{accent.Render("CONTROLS · ↑↓ scroll · Esc close")}, wrapped[start:]...)
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	for i := range lines {
		lines[i] = lipgloss.NewStyle().MaxWidth(w).Render(lines[i])
	}
	return strings.Join(lines, "\n")
}
