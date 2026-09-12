package synth

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderWavesAndWAV(t *testing.T) {
	for wave := range Waves {
		p := Default()
		p.Wave = wave
		samples := p.Render()
		want := int(float64(Rate)*60/float64(p.BPM)/4) * 16
		if len(samples) != want {
			t.Fatal("wrong duration")
		}
		sum := 0.0
		for _, v := range samples {
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1 {
				t.Fatal("invalid sample")
			}
			sum += v * v
		}
		if sum/float64(len(samples)) < .001 {
			t.Fatal("silent synth")
		}
		if math.Abs(samples[0]-samples[len(samples)-1]) > .02 {
			t.Fatalf("loop discontinuity: %f", samples[0]-samples[len(samples)-1])
		}
		path := filepath.Join(t.TempDir(), "synth.wav")
		if e := WriteWAV(path, samples); e != nil {
			t.Fatal(e)
		}
		data, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" || len(data) != 44+len(samples)*2 || binary.LittleEndian.Uint32(data[24:]) != Rate {
			t.Fatal("invalid WAV")
		}
	}
}
func TestAllGatesOffIsSilent(t *testing.T) {
	p := Default()
	p.Enabled = [16]bool{}
	for _, v := range p.Render() {
		if v != 0 {
			t.Fatal("disabled pattern produced sound")
		}
	}
}
