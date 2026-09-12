// Package audio controls a private, configuration-independent mpv process over JSON IPC.
package audio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Snapshot struct {
	Position, Duration, Volume float64
	Paused, Idle               bool
	Path, Title                string
}
type Event struct {
	EOF bool
	Err error
}
type Player struct {
	mu        sync.Mutex
	writeMu   sync.Mutex
	state     Snapshot
	conn      net.Conn
	cmd       *exec.Cmd
	dir       string
	done      chan struct{}
	Events    chan Event
	closeOnce sync.Once
}

func New(nullAudio bool) (*Player, error) {
	if _, e := exec.LookPath("mpv"); e != nil {
		return nil, fmt.Errorf("mpv is required: %w", e)
	}
	dir, e := os.MkdirTemp("", "mor10z-ipc-")
	if e != nil {
		return nil, e
	}
	p := &Player{dir: dir, done: make(chan struct{}), Events: make(chan Event, 64), state: Snapshot{Idle: true, Volume: 70}}
	args := []string{"--no-config", "--idle=yes", "--no-video", "--audio-display=no", "--no-terminal", "--really-quiet", "--volume=70", "--input-ipc-server=" + filepath.Join(dir, "socket")}
	if nullAudio {
		args = append(args, "--ao=null")
	}
	p.cmd = exec.Command("mpv", args...)
	if e = p.cmd.Start(); e != nil {
		os.RemoveAll(dir)
		return nil, e
	}
	go func() { p.cmd.Wait(); close(p.done) }()
	for i := 0; i < 100; i++ {
		p.conn, e = net.Dial("unix", filepath.Join(dir, "socket"))
		if e == nil {
			break
		}
		select {
		case <-p.done:
			os.RemoveAll(dir)
			return nil, fmt.Errorf("mpv exited during startup")
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	if e != nil {
		p.Close()
		return nil, fmt.Errorf("connect to mpv: %w", e)
	}
	go p.read()
	for i, name := range []string{"time-pos", "duration", "pause", "idle-active", "path", "media-title", "volume"} {
		if e = p.Command("observe_property", i+1, name); e != nil {
			p.Close()
			return nil, e
		}
	}
	return p, nil
}
func (p *Player) emit(e Event) {
	select {
	case p.Events <- e:
	default:
	}
}
func (p *Player) read() {
	s := bufio.NewScanner(p.conn)
	s.Buffer(make([]byte, 4096), 1024*1024)
	for s.Scan() {
		var v struct {
			Event  string          `json:"event"`
			Name   string          `json:"name"`
			Data   json.RawMessage `json:"data"`
			Reason string          `json:"reason"`
			Error  string          `json:"error"`
		}
		if json.Unmarshal(s.Bytes(), &v) != nil {
			continue
		}
		if v.Event == "end-file" {
			if v.Reason == "eof" {
				p.emit(Event{EOF: true})
			}
			if v.Reason == "error" {
				p.emit(Event{Err: fmt.Errorf("cannot play this file: %s", v.Error)})
			}
		}
		if v.Event == "" && v.Error != "" && v.Error != "success" {
			p.emit(Event{Err: fmt.Errorf("mpv: %s", v.Error)})
		}
		if v.Event != "property-change" {
			continue
		}
		p.mu.Lock()
		switch v.Name {
		case "time-pos":
			json.Unmarshal(v.Data, &p.state.Position)
		case "duration":
			json.Unmarshal(v.Data, &p.state.Duration)
		case "pause":
			json.Unmarshal(v.Data, &p.state.Paused)
		case "idle-active":
			json.Unmarshal(v.Data, &p.state.Idle)
		case "path":
			json.Unmarshal(v.Data, &p.state.Path)
		case "media-title":
			json.Unmarshal(v.Data, &p.state.Title)
		case "volume":
			json.Unmarshal(v.Data, &p.state.Volume)
		}
		p.mu.Unlock()
	}
	p.emit(Event{Err: fmt.Errorf("audio connection closed")})
}
func (p *Player) Command(args ...any) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	if p.conn == nil {
		return fmt.Errorf("audio unavailable")
	}
	p.conn.SetWriteDeadline(time.Now().Add(time.Second))
	return json.NewEncoder(p.conn).Encode(map[string]any{"command": args})
}
func (p *Player) Load(path string, loop bool) error {
	v := "no"
	if loop {
		v = "inf"
	}
	if e := p.Command("set_property", "loop-file", v); e != nil {
		return e
	}
	if e := p.Command("loadfile", path, "replace"); e != nil {
		return e
	}
	return p.Command("set_property", "pause", false)
}
func (p *Player) Snapshot() Snapshot { p.mu.Lock(); defer p.mu.Unlock(); return p.state }
func (p *Player) Close() {
	p.closeOnce.Do(func() {
		if p.conn != nil {
			p.Command("quit")
			p.conn.Close()
		}
		if p.cmd != nil && p.cmd.Process != nil {
			select {
			case <-p.done:
			case <-time.After(time.Second):
				p.cmd.Process.Kill()
				<-p.done
			}
		}
		os.RemoveAll(p.dir)
	})
}
