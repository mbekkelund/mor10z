package audio

import (
	"mor10z/internal/synth"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func await(t *testing.T, p *Player, predicate func(Snapshot) bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s := p.Snapshot()
		if predicate(s) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("audio state timed out: %+v", p.Snapshot())
}
func TestMPVTransport(t *testing.T) {
	if os.Getenv("MOR10Z_AUDIO_TEST") != "1" {
		t.Skip("set MOR10Z_AUDIO_TEST=1 for real mpv IPC tests")
	}
	dir := t.TempDir()
	wav := filepath.Join(dir, "test track.wav")
	if e := synth.WriteWAV(wav, synth.Default().Render()); e != nil {
		t.Fatal(e)
	}
	paths := []string{wav}
	if _, e := exec.LookPath("ffmpeg"); e == nil {
		mp3 := filepath.Join(dir, "test track.mp3")
		if out, e := exec.Command("ffmpeg", "-v", "error", "-i", wav, "-y", mp3).CombinedOutput(); e != nil {
			t.Fatalf("encode MP3: %s: %v", out, e)
		}
		paths = append(paths, mp3)
	} else {
		t.Fatal("ffmpeg required to test MP3")
	}
	p, e := New(true)
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	for _, path := range paths {
		if e = p.Load(path, false); e != nil {
			t.Fatal(e)
		}
		await(t, p, func(s Snapshot) bool { return s.Path == path && !s.Idle && s.Duration > 1 && s.Position > .05 })
		p.Command("set_property", "pause", true)
		await(t, p, func(s Snapshot) bool { return s.Paused })
		p.Command("seek", 1, "absolute")
		await(t, p, func(s Snapshot) bool { return s.Position >= .95 })
		p.Command("set_property", "volume", 35)
		await(t, p, func(s Snapshot) bool { return s.Volume == 35 })
		p.Command("set_property", "pause", false)
		await(t, p, func(s Snapshot) bool { return !s.Paused })
		p.Command("stop")
		await(t, p, func(s Snapshot) bool { return s.Idle })
	}
	short := filepath.Join(dir, "short.wav")
	synth.WriteWAV(short, make([]float64, synth.Rate/5))
	for len(p.Events) > 0 {
		<-p.Events
	}
	if e = p.Load(short, false); e != nil {
		t.Fatal(e)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case e := <-p.Events:
			if e.Err != nil {
				t.Fatal(e.Err)
			}
			if e.EOF {
				return
			}
		case <-deadline:
			t.Fatal("no EOF event")
		}
	}
}
