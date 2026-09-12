package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"mor10z/internal/synth"
	"mor10z/internal/visual"
)

type visualMsg struct {
	clip visual.Clip
	err  error
}
type visualState struct {
	clip            visual.Clip
	frame           visual.Frame
	peaks           [visual.Bands]float64
	history         [64][visual.Bands]float64
	historyAt       int
	mode            int
	busy            bool
	source, problem string
	retryAt         time.Time
}

func (m *Model) updateVisual() tea.Cmd {
	if m.tab != 3 && m.tab != 2 {
		return nil
	}
	snap := m.player.Snapshot()
	if snap.Path != m.viz.source {
		m.viz.source = snap.Path
		m.viz.frame = visual.Frame{}
		m.viz.peaks = [visual.Bands]float64{}
		m.viz.history = [64][visual.Bands]float64{}
		m.viz.problem = ""
	}
	if snap.Paused {
		return nil
	}
	target, valid := m.viz.clip.At(snap.Path, snap.Position)
	if snap.Idle {
		target = visual.Frame{}
		valid = false
	}
	for i, value := range target.Spectrum {
		m.viz.frame.Spectrum[i] = math.Max(value, m.viz.frame.Spectrum[i]*.86)
		m.viz.peaks[i] = math.Max(m.viz.frame.Spectrum[i], m.viz.peaks[i]-.012)
	}
	m.viz.frame.Wave = target.Wave
	m.viz.frame.RMS = target.RMS
	m.viz.frame.Peak = target.Peak
	m.viz.history[m.viz.historyAt] = m.viz.frame.Spectrum
	m.viz.historyAt = (m.viz.historyAt + 1) % len(m.viz.history)
	if snap.Idle || snap.Path == "" || m.viz.busy || time.Now().Before(m.viz.retryAt) {
		return nil
	}
	// Keep a full short track cached across synth/repeat loops; prefetch long tracks.
	end := m.viz.clip.Start + float64(len(m.viz.clip.Frames))/visual.FPS
	complete := m.viz.clip.Start == 0 && snap.Duration > 0 && end >= snap.Duration-.05
	if !valid || (!complete && snap.Position > end-1) {
		m.viz.busy = true
		path, pos := snap.Path, math.Max(0, snap.Position-.1)
		if snap.Duration > 0 && snap.Duration <= visual.Seconds {
			pos = 0
		}
		return func() tea.Msg { clip, err := visual.Decode(path, pos); return visualMsg{clip, err} }
	}
	return nil
}
func (m *Model) acceptVisual(msg visualMsg) {
	m.viz.busy = false
	if msg.clip.Path != m.player.Snapshot().Path {
		return
	}
	if msg.err != nil {
		m.viz.problem = "Audio analysis unavailable · check ffmpeg"
		m.viz.retryAt = time.Now().Add(5 * time.Second)
		return
	}
	m.viz.clip = msg.clip
	m.viz.problem = ""
	m.viz.retryAt = time.Time{}
}

var spectrumColors = []lipgloss.Color{"#45F3CE", "#50EBCF", "#5DDCD8", "#70C4EA", "#95A3FC", "#BB87EF", "#E574CA", "#FF659A"}

func spectrumStyle(i, n int) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(spectrumColors[min(len(spectrumColors)-1, i*len(spectrumColors)/max(1, n))])
}
func (m Model) visualView(w, h int) string {
	mode := []string{"SPECTRUM", "OSCILLOSCOPE", "WATERFALL"}[m.viz.mode]
	label := "AUDIO REACTIVE / " + mode
	if m.player.Snapshot().Paused {
		label += " · PAUSED"
	}
	lines := []string{accent.Bold(true).Render(fit(label, w))}
	if m.viz.problem != "" {
		lines[0] = rose.Render(fit(m.viz.problem, w))
	} else if m.player.Snapshot().Idle {
		lines[0] = accent.Render(fit("VISUALIZER · play a track, or 3 → Enter for synth", w))
	} else if m.viz.busy && m.viz.clip.Path != m.viz.source {
		lines[0] = accent.Render("Tuning into the audio…")
	}
	plotH := h - 2
	switch m.viz.mode {
	case 0:
		if plotH >= 12 {
			spectrumH := plotH - 5
			lines = append(lines, m.spectrum(w, spectrumH)...)
			lines = append(lines, muted.Render(fit("50 Hz          200          1k           5k       10k", w)))
			lines = append(lines, m.scope(w, 4)...)
		} else {
			lines = append(lines, m.spectrum(w, plotH)...)
		}
	case 1:
		lines = append(lines, m.scope(w, plotH)...)
	case 2:
		lines = append(lines, m.waterfall(w, plotH)...)
	}
	meter := int(math.Min(1, m.viz.frame.RMS*3) * 16)
	level := accent.Render(strings.Repeat("▰", meter)) + muted.Render(strings.Repeat("▱", 16-meter))
	lines = append(lines, level+muted.Render(fmt.Sprintf("  %5.1f dB  · z style", 20*math.Log10(math.Max(.00001, m.viz.frame.Peak)))))
	return strings.Join(lines, "\n")
}
func (m Model) spectrum(w, h int) []string {
	count := min(visual.Bands, w/3)
	cell := w / count
	barW := max(1, cell-1)
	lines := make([]string, h)
	partial := []rune(" ▁▂▃▄▅▆▇█")
	for row := 0; row < h; row++ {
		var b strings.Builder
		for col := 0; col < count; col++ {
			from, to := col*visual.Bands/count, (col+1)*visual.Bands/count
			value, peak := 0.0, 0.0
			for i := from; i < to; i++ {
				value = math.Max(value, m.viz.frame.Spectrum[i])
				peak = math.Max(peak, m.viz.peaks[i])
			}
			units := int(math.Max(0, math.Min(8, value*float64(h)*8-float64(h-1-row)*8)))
			mark := strings.Repeat(string(partial[units]), barW)
			if units == 0 && peak > .03 && h-1-int(peak*float64(h-1)) == row {
				mark = strings.Repeat("▔", barW)
			}
			b.WriteString(spectrumStyle(col, count).Render(mark))
			b.WriteString(" ")
		}
		lines[row] = b.String()
	}
	return lines
}
func (m Model) scope(w, h int) []string {
	// Unicode braille provides a 2×4 subpixel grid per terminal cell.
	pixels := make([][]bool, w*2)
	for x := range pixels {
		pixels[x] = make([]bool, h*4)
	}
	previous := h * 2
	gain := 1.0
	if m.viz.frame.Peak > .02 {
		gain = math.Min(4, .85/m.viz.frame.Peak)
	}
	for x := 0; x < w*2; x++ {
		pos := float64(x) * float64(visual.WavePoints-1) / float64(max(1, w*2-1))
		i := int(pos)
		frac := pos - float64(i)
		sample := m.viz.frame.Wave[i]*(1-frac) + m.viz.frame.Wave[min(i+1, visual.WavePoints-1)]*frac
		y := int((1 - math.Max(-1, math.Min(1, sample*gain))) * float64(h*4-1) / 2)
		if x == 0 {
			previous = y
		}
		for py := min(previous, y); py <= max(previous, y); py++ {
			pixels[x][py] = true
		}
		previous = y
	}
	dots := [2][4]uint{{0, 1, 2, 6}, {3, 4, 5, 7}}
	lines := make([]string, h)
	for row := 0; row < h; row++ {
		var b strings.Builder
		for col := 0; col < w; col++ {
			mask := 0
			for dx := 0; dx < 2; dx++ {
				for dy := 0; dy < 4; dy++ {
					if pixels[col*2+dx][row*4+dy] {
						mask |= 1 << dots[dx][dy]
					}
				}
			}
			r := ' '
			if mask != 0 {
				r = rune(0x2800 + mask)
			}
			b.WriteString(spectrumStyle(col, w).Render(string(r)))
		}
		lines[row] = b.String()
	}
	return lines
}
func (m Model) waterfall(w, h int) []string {
	shades := []rune(" ·░▒▓█")
	lines := make([]string, h)
	for row := 0; row < h; row++ {
		age := (h - 1 - row) * min(h, len(m.viz.history)) / max(1, h)
		index := (m.viz.historyAt - 1 - age + len(m.viz.history)) % len(m.viz.history)
		var b strings.Builder
		for x := 0; x < w; x++ {
			v := m.viz.history[index][x*visual.Bands/w]
			shade := min(len(shades)-1, int(v*float64(len(shades))))
			b.WriteString(spectrumStyle(x, w).Render(string(shades[shade])))
		}
		lines[row] = b.String()
	}
	return lines
}

func (m Model) synthScope(w, h int) []string {
	if !m.synthPlaying {
		for i := range m.viz.frame.Wave {
			phase := math.Mod(float64(i)/float64(visual.WavePoints)*3, 1)
			m.viz.frame.Wave[i] = synth.Osc(phase, .001, m.pattern.Wave)
		}
		m.viz.frame.Peak = 1
	}
	return m.scope(w, h)
}
