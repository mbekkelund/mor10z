package library

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func put(t *testing.T, path string) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, []byte("fixture"), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestScanWatchChanges(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "album", "A.WAV")
	put(t, a)
	put(t, filepath.Join(dir, "z.mp3"))
	put(t, filepath.Join(dir, "cover.jpg"))
	put(t, filepath.Join(dir, ".hidden", "secret.mp3"))
	tracks, e := Scan([]string{dir, filepath.Dir(a)})
	if e != nil || len(tracks) != 2 || tracks[0].Path != a {
		t.Fatalf("scan: %v %v", tracks, e)
	}
	os.Remove(a)
	put(t, filepath.Join(dir, "new", "fresh.wav"))
	tracks, e = Scan([]string{dir})
	if e != nil || len(tracks) != 2 || tracks[0].Title != "fresh" {
		t.Fatalf("rescan: %v %v", tracks, e)
	}
}
func TestStateAndM3URoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, e := Load(dir)
	if e != nil {
		t.Fatal(e)
	}
	s.Volume = 35
	s.Roots = []string{dir}
	s.Playlists = append(s.Playlists, Playlist{Name: "夜 · beats", Paths: []string{filepath.Join(dir, "a b.wav"), filepath.Join(dir, "æ.mp3")}})
	if e = Save(dir, s); e != nil {
		t.Fatal(e)
	}
	got, e := Load(dir)
	if e != nil || !reflect.DeepEqual(s, got) {
		t.Fatalf("round trip: %#v %v", got, e)
	}
	path := filepath.Join(dir, "list.m3u")
	if e = WriteM3U(path, s.Playlists[1]); e != nil {
		t.Fatal(e)
	}
	p, e := ReadM3U(path)
	if e != nil || !reflect.DeepEqual(p.Paths, s.Playlists[1].Paths) {
		t.Fatalf("m3u: %v %v", p, e)
	}
	os.WriteFile(path, []byte("\ufeff#EXTM3U\n#EXTINF:1,test\na b.wav\næ.mp3\n"), 0600)
	p, e = ReadM3U(path)
	if e != nil || !reflect.DeepEqual(p.Paths, s.Playlists[1].Paths) {
		t.Fatalf("relative m3u: %v %v", p, e)
	}
}
func TestCorruptStateIsNotSilentlyReset(t *testing.T) {
	dir := t.TempDir()
	put(t, filepath.Join(dir, "state.json"))
	if _, e := Load(dir); e == nil {
		t.Fatal("expected corrupt state error")
	}
}
func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for input, want := range map[string]string{"~": home, "~/Music": filepath.Join(home, "Music"), "~/~": filepath.Join(home, "~")} {
		got, e := Expand(input)
		if e != nil || got != want {
			t.Fatalf("%q: %q %v", input, got, e)
		}
	}
}
