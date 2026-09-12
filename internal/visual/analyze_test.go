package visual

import (
	"math"
	"mor10z/internal/synth"
	"os/exec"
	"path/filepath"
	"testing"
)

func tone(freq float64) []float64 {
	samples := make([]float64, Rate)
	for i := range samples {
		samples[i] = .5 * math.Sin(2*math.Pi*freq*float64(i)/Rate)
	}
	return samples
}
func TestSpectrumFindsTone(t *testing.T) {
	for _, frequency := range []float64{100, 440, 4000} {
		frames := Analyze(tone(frequency))
		if len(frames) != FPS {
			t.Fatalf("frames: %d", len(frames))
		}
		frame := frames[0]
		peak := 0
		for i, v := range frame.Spectrum {
			if v > frame.Spectrum[peak] {
				peak = i
			}
		}
		center := 50 * math.Pow(200, (float64(peak)+.5)/Bands)
		if math.Abs(math.Log(center/frequency)) > .16 {
			t.Fatalf("%g Hz tone plotted near %g Hz", frequency, center)
		}
		if math.Abs(frame.RMS-math.Sqrt(.125)) > .02 || math.Abs(frame.Peak-.5) > .01 {
			t.Fatalf("bad meters: %+v", frame)
		}
	}
}
func TestSilenceAndTimeline(t *testing.T) {
	clip := Clip{Path: "test.wav", Start: 5, Frames: Analyze(make([]float64, Rate))}
	for _, frame := range clip.Frames {
		if frame != (Frame{}) {
			t.Fatal("silence produced nonzero visualization")
		}
	}
	for _, p := range []float64{0, 4.99, 6, 20} {
		if _, ok := clip.At("test.wav", p); ok {
			t.Fatalf("out-of-window frame at %v", p)
		}
	}
	if _, ok := clip.At("other.wav", 5.5); ok {
		t.Fatal("stale track accepted")
	}
	if _, ok := clip.At("test.wav", 5.5); !ok {
		t.Fatal("missing frame")
	}
}
func TestDecodeWAVAndMP3(t *testing.T) {
	if _, e := exec.LookPath("ffmpeg"); e != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	wav := filepath.Join(dir, "a 'track.wav")
	mp3 := filepath.Join(dir, "a track.mp3")
	if err := synth.WriteWAV(wav, synth.Default().Render()); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("ffmpeg", "-v", "error", "-i", wav, mp3).CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", out, err)
	}
	for _, path := range []string{wav, mp3} {
		clip, err := Decode(path, .5)
		if err != nil {
			t.Fatal(err)
		}
		frame, ok := clip.At(path, .6)
		if !ok || frame.Peak == 0 {
			t.Fatal("decoded audio missing")
		}
		if len(clip.Frames) > Seconds*FPS {
			t.Fatal("unbounded decode")
		}
	}
	if _, err := Decode(filepath.Join(dir, "missing.wav"), 0); err == nil {
		t.Fatal("expected decoder error")
	}
}
