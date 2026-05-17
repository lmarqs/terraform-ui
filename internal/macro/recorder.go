package macro

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Manifest describes a recording session for GIF stitching.
type Manifest struct {
	Width  int         `json:"width"`
	Height int         `json:"height"`
	Frames []FrameMeta `json:"frames"`
}

// FrameMeta describes a single captured frame.
type FrameMeta struct {
	File    string `json:"file"`
	DelayMs int64  `json:"delay_ms"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// Recorder wraps a tea.Model and captures frames + tape during interaction.
type Recorder struct {
	inner    tea.Model
	dir      string
	width    int
	height   int
	frames   []FrameMeta
	tape     []string
	frameNum int
	lastTime time.Time
}

// NewRecorder creates a recording middleware that writes frames to dir.
func NewRecorder(inner tea.Model, dir string, width, height int) *Recorder {
	return &Recorder{
		inner:    inner,
		dir:      dir,
		width:    width,
		height:   height,
		lastTime: time.Now(),
	}
}

func (r *Recorder) Init() tea.Cmd {
	return r.inner.Init()
}

func (r *Recorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		r.recordKey(msg)
	case tea.WindowSizeMsg:
		r.recordResize(msg)
	}

	var cmd tea.Cmd
	r.inner, cmd = r.inner.Update(msg)
	r.captureFrame()
	return r, cmd
}

func (r *Recorder) View() string {
	return r.inner.View()
}

// Finalize writes manifest.json and recording.tape to the output dir.
func (r *Recorder) Finalize() error {
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return err
	}

	manifest := Manifest{
		Width:  r.width,
		Height: r.height,
		Frames: r.frames,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(r.dir, "manifest.json"), manifestData, 0644); err != nil {
		return err
	}

	tapeContent := strings.Join(r.tape, "\n") + "\n"
	return os.WriteFile(filepath.Join(r.dir, "recording.tape"), []byte(tapeContent), 0644)
}

func (r *Recorder) recordKey(msg tea.KeyMsg) {
	elapsed := time.Since(r.lastTime)
	if elapsed > 200*time.Millisecond && len(r.tape) > 0 {
		r.tape = append(r.tape, fmt.Sprintf("sleep %s", roundDuration(elapsed)))
	}

	key := KeyToString(msg)
	if key != "" {
		r.tape = append(r.tape, fmt.Sprintf("key %s", key))
	}
	r.lastTime = time.Now()
}

func (r *Recorder) recordResize(msg tea.WindowSizeMsg) {
	r.width = msg.Width
	r.height = msg.Height
	r.tape = append(r.tape, fmt.Sprintf("resize %d %d", msg.Width, msg.Height))
}

func (r *Recorder) captureFrame() {
	r.CaptureView(r.inner.View())
}

// CaptureView records a frame with the given rendered view content.
// Used by the Runner in headless mode where the Driver owns the model.
func (r *Recorder) CaptureView(view string) {
	r.frameNum++
	filename := fmt.Sprintf("frame_%04d.txt", r.frameNum)

	var delayMs int64
	if r.frameNum > 1 {
		delayMs = time.Since(r.lastTime).Milliseconds()
	}

	path := filepath.Join(r.dir, filename)
	_ = os.MkdirAll(r.dir, 0755)
	_ = os.WriteFile(path, []byte(view), 0644)

	r.frames = append(r.frames, FrameMeta{
		File:    filename,
		DelayMs: delayMs,
		Width:   r.width,
		Height:  r.height,
	})
}

func roundDuration(d time.Duration) string {
	if d < time.Second {
		ms := d.Truncate(100 * time.Millisecond)
		return ms.String()
	}
	s := d.Truncate(100 * time.Millisecond)
	return s.String()
}
