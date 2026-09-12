// Package synth is a deterministic, offline subtractive synthesizer. No samples or plugins.
package synth

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

const Rate = 44100

var Waves = []string{"SAW", "SQUARE", "SINE", "TRIANGLE"}

type Pattern struct {
	Notes   [16]int  `json:"notes"`
	Enabled [16]bool `json:"enabled"`
	BPM     int      `json:"bpm"`
	Wave    int      `json:"wave"`
	Cutoff  float64  `json:"cutoff"`
	Delay   bool     `json:"delay"`
}

func Default() Pattern {
	return Pattern{Notes: [16]int{45, 45, 57, 45, 48, 45, 55, 48, 43, 43, 55, 43, 50, 48, 43, 40}, Enabled: [16]bool{true, false, true, true, true, false, true, false, true, false, true, true, true, true, false, true}, BPM: 118, Wave: 0, Cutoff: 2200, Delay: true}
}
func Note(n int) string {
	names := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	if n < 0 {
		return "--"
	}
	return fmt.Sprintf("%s%d", names[n%12], n/12-1)
}
func polyBLEP(t, dt float64) float64 {
	if t < dt {
		t /= dt
		return t + t - t*t - 1
	}
	if t > 1-dt {
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}
func Osc(phase, dt float64, wave int) float64 {
	switch wave {
	case 0:
		return 2*phase - 1 - polyBLEP(phase, dt)
	case 1:
		v := -1.0
		if phase < 0.5 {
			v = 1
		}
		return v + polyBLEP(phase, dt) - polyBLEP(math.Mod(phase+0.5, 1), dt)
	case 2:
		return math.Sin(2 * math.Pi * phase)
	default:
		return 1 - 4*math.Abs(phase-0.5)
	}
}
func (p Pattern) Render() []float64 {
	bpm := max(40, min(240, p.BPM))
	step := int(float64(Rate) * 60 / float64(bpm) / 4)
	size := step * 16
	// Warm up three repetitions so the delay tail wraps seamlessly at the loop boundary.
	out := make([]float64, size*3)
	delay := make([]float64, step*3)
	di := 0
	phase, sub, filtered := 0.0, 0.0, 0.0
	cutoff := max(80.0, min(16000.0, p.Cutoff))
	alpha := 1 - math.Exp(-2*math.Pi*cutoff/Rate)
	for i := range out {
		idx := (i / step) % 16
		t := float64(i%step) / Rate
		env := 0.0
		if p.Enabled[idx] {
			dur := float64(step) / Rate
			env = math.Min(1, t/.006) * math.Exp(-t*9) * math.Min(1, (dur-t)/.025)
		}
		freq := 440 * math.Pow(2, float64(p.Notes[idx]-69)/12)
		dt := freq / Rate
		phase = math.Mod(phase+dt, 1)
		sub = math.Mod(sub+dt/2, 1)
		v := (Osc(phase, dt, p.Wave)*.65 + math.Sin(2*math.Pi*sub)*.35) * env
		filtered += alpha * (v - filtered)
		v = filtered
		if p.Delay {
			echo := delay[di]
			delay[di] = v + echo*.35
			v += echo * .3
			di = (di + 1) % len(delay)
		}
		out[i] = math.Tanh(v*1.4) * .65
	}
	return out[size*2:]
}
func WriteWAV(path string, samples []float64) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	size := uint32(len(samples) * 2)
	header := make([]byte, 44)
	copy(header, "RIFF")
	binary.LittleEndian.PutUint32(header[4:], 36+size)
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], Rate)
	binary.LittleEndian.PutUint32(header[28:], Rate*2)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], size)
	if _, e = f.Write(header); e != nil {
		return e
	}
	data := make([]byte, len(samples)*2)
	for i, v := range samples {
		v = max(-1, min(1, v))
		binary.LittleEndian.PutUint16(data[i*2:], uint16(int16(v*32767)))
	}
	if _, e = f.Write(data); e != nil {
		return e
	}
	return f.Close()
}
