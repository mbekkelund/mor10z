package library

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Track struct {
	Path   string
	Title  string
	Folder string
	Format string
}
type Playlist struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}
type State struct {
	Roots     []string   `json:"roots"`
	Playlists []Playlist `json:"playlists"`
	Volume    int        `json:"volume"`
}

func Describe(path string) Track {
	return Track{path, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), filepath.Base(filepath.Dir(path)), strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))}
}
func Supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3", ".wav", ".flac", ".ogg", ".m4a", ".opus":
		return true
	}
	return false
}
func Expand(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return filepath.Abs(path)
}
func Scan(roots []string) ([]Track, error) {
	seen := map[string]bool{}
	tracks := []Track{}
	var errs []error
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				errs = append(errs, err)
				return nil
			}
			if d.IsDir() {
				if path != root && strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if Supported(path) && !seen[path] {
				seen[path] = true
				tracks = append(tracks, Describe(path))
			}
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	sort.Slice(tracks, func(i, j int) bool {
		a, b := strings.ToLower(tracks[i].Title), strings.ToLower(tracks[j].Title)
		if a == b {
			return tracks[i].Path < tracks[j].Path
		}
		return a < b
	})
	return tracks, errors.Join(errs...)
}
func Load(dir string) (State, error) {
	s := State{Volume: 70, Playlists: []Playlist{{Name: "Favorites", Paths: []string{}}}}
	b, e := os.ReadFile(filepath.Join(dir, "state.json"))
	if os.IsNotExist(e) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	if e = json.Unmarshal(b, &s); e != nil {
		return s, fmt.Errorf("read state: %w", e)
	}
	if s.Volume < 0 {
		s.Volume = 0
	}
	if s.Volume > 100 {
		s.Volume = 100
	}
	if len(s.Playlists) == 0 {
		s.Playlists = []Playlist{{Name: "Favorites"}}
	}
	return s, nil
}
func Save(dir string, s State) error {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".state-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), filepath.Join(dir, "state.json"))
}
func ReadM3U(path string) (Playlist, error) {
	p := Playlist{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))}
	f, e := os.Open(path)
	if e != nil {
		return p, e
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(s.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(filepath.Dir(path), line)
		}
		if Supported(line) {
			p.Paths = append(p.Paths, filepath.Clean(line))
		}
	}
	return p, s.Err()
}
func WriteM3U(path string, p Playlist) error {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for _, path := range p.Paths {
		if strings.ContainsAny(path, "\r\n") {
			return fmt.Errorf("filename contains newline")
		}
		b.WriteString(path + "\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0600)
}
func AddRoot(s *State, path string) error {
	p, e := Expand(strings.TrimSpace(path))
	if e != nil {
		return e
	}
	st, e := os.Stat(p)
	if e != nil {
		return e
	}
	if !st.IsDir() && !Supported(p) {
		return fmt.Errorf("unsupported file: %s", p)
	}
	for _, r := range s.Roots {
		if r == p {
			return nil
		}
	}
	s.Roots = append(s.Roots, p)
	return nil
}
