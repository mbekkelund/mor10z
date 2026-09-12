// Package visual analyzes short decoded windows of the current track. It never
// captures a microphone or system audio; frames are selected by mpv's playhead.
package visual

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/cmplx"
	"os/exec"
	"strconv"
	"time"
)

const (
	Rate       = 22050
	FPS        = 30
	Bands      = 48
	WavePoints = 160
	windowSize = 2048
	Seconds    = 6
)

type Frame struct {
	Spectrum  [Bands]float64
	Wave      [WavePoints]float64
	RMS, Peak float64
}
type Clip struct {
	Path   string
	Start  float64
	Frames []Frame
}

func (c Clip) At(path string, position float64) (Frame, bool) {
	index := int(math.Floor((position - c.Start) * FPS))
	if path != c.Path || position < c.Start || index < 0 || index >= len(c.Frames) {
		return Frame{}, false
	}
	return c.Frames[index], true
}

// Decode bounds work and memory to six seconds, even for a multi-hour recording.
func Decode(path string, start float64) (Clip, error) {
	c := Clip{Path: path, Start: math.Max(0, start)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-v", "error", "-ss", strconv.FormatFloat(c.Start, 'f', 4, 64), "-i", path, "-t", strconv.Itoa(Seconds), "-map", "0:a:0", "-vn", "-ac", "1", "-ar", strconv.Itoa(Rate), "-f", "f32le", "pipe:1")
	data, err := cmd.Output()
	if err != nil {
		return c, fmt.Errorf("visualizer: ffmpeg could not decode audio: %w", err)
	}
	samples := make([]float64, len(data)/4)
	for i := range samples {
		value := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:])))
		if !math.IsNaN(value) && !math.IsInf(value, 0) {
			samples[i] = value
		}
	}
	c.Frames = Analyze(samples)
	if len(c.Frames) == 0 {
		return c, fmt.Errorf("visualizer: no audio samples")
	}
	return c, nil
}

func Analyze(samples []float64) []Frame {
	hop := Rate / FPS
	frames := make([]Frame, 0, (len(samples)+hop-1)/hop)
	for offset := 0; offset < len(samples); offset += hop {
		var frame Frame
		var bins [windowSize]complex128
		count := min(windowSize, len(samples)-offset)
		energy := 0.0
		for i := 0; i < count; i++ {
			sample := samples[offset+i]
			energy += sample * sample
			frame.Peak = math.Max(frame.Peak, math.Abs(sample))
			bins[i] = complex(sample*(.5-.5*math.Cos(2*math.Pi*float64(i)/(windowSize-1))), 0)
		}
		frame.RMS = math.Sqrt(energy / float64(count))
		// A short, zero-crossing aligned oscilloscope trace stays legible on bass notes.
		crossing := 0
		for i := 1; i < min(count/2, 400); i++ {
			if samples[offset+i-1] <= 0 && samples[offset+i] > 0 {
				crossing = i
				break
			}
		}
		for i := range frame.Wave {
			index := offset + crossing + i*4
			if index < len(samples) {
				frame.Wave[i] = samples[index]
			}
		}
		fft(bins[:])
		for band := range frame.Spectrum {
			low := 50 * math.Pow(200, float64(band)/Bands)
			high := 50 * math.Pow(200, float64(band+1)/Bands)
			lo := max(1, int(low*windowSize/Rate))
			hi := min(windowSize/2, max(lo+1, int(high*windowSize/Rate)))
			peak := 0.0
			for bin := lo; bin < hi; bin++ {
				peak = math.Max(peak, cmplx.Abs(bins[bin])*4/windowSize)
			}
			db := 20 * math.Log10(math.Max(peak, 1e-6))
			frame.Spectrum[band] = math.Max(0, math.Min(1, (db+65)/65))
		}
		frames = append(frames, frame)
	}
	return frames
}
func fft(data []complex128) {
	n := len(data)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			data[i], data[j] = data[j], data[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		root := cmplx.Rect(1, -2*math.Pi/float64(size))
		for base := 0; base < n; base += size {
			factor := complex(1, 0)
			for j := 0; j < size/2; j++ {
				a, b := data[base+j], data[base+j+size/2]*factor
				data[base+j] = a + b
				data[base+j+size/2] = a - b
				factor *= root
			}
		}
	}
}
