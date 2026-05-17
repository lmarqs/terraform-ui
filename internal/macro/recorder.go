package macro

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type Manifest struct {
	Width  int         `json:"width"`
	Height int         `json:"height"`
	Frames []FrameMeta `json:"frames"`
}

type FrameMeta struct {
	File    string `json:"file"`
	DelayMs int64  `json:"delay_ms"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

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

func NewRecorder(inner tea.Model, dir string, width, height int) *Recorder {
	return &Recorder{
		inner:    inner,
		dir:      dir,
		width:    width,
		height:   height,
		lastTime: time.Now(),
	}
}

// Inner returns the wrapped model (nil in headless mode).
func (r *Recorder) Inner() tea.Model {
	return r.inner
}

func (r *Recorder) Init() tea.Cmd {
	if r.inner == nil {
		return nil
	}
	return r.inner.Init()
}

func (r *Recorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		r.recordKey(msg)
	case tea.WindowSizeMsg:
		r.recordResize(msg)
	}

	if r.inner == nil {
		return r, nil
	}

	var cmd tea.Cmd
	r.inner, cmd = r.inner.Update(msg)
	r.CaptureView(r.inner.View())
	return r, cmd
}

func (r *Recorder) View() string {
	if r.inner == nil {
		return ""
	}
	return r.inner.View()
}

func (r *Recorder) Finalize() error {
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return err
	}

	manifest := Manifest{
		Width:  r.width,
		Height: r.height,
		Frames: r.frames,
	}
	if err := os.WriteFile(filepath.Join(r.dir, "manifest.json"), sdk.MarshalJSON(manifest), 0644); err != nil {
		return err
	}

	if len(r.tape) > 0 {
		tapeContent := strings.Join(r.tape, "\n") + "\n"
		return os.WriteFile(filepath.Join(r.dir, "recording.tape"), []byte(tapeContent), 0644)
	}
	return nil
}

func (r *Recorder) CaptureView(view string) {
	r.frameNum++
	filename := fmt.Sprintf("frame_%04d.txt", r.frameNum)

	now := time.Now()
	var delayMs int64
	if r.frameNum > 1 {
		delayMs = now.Sub(r.lastTime).Milliseconds()
	}
	r.lastTime = now

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

// Resize updates the recorder's dimensions (used by headless runner on CmdResize).
func (r *Recorder) Resize(width, height int) {
	r.width = width
	r.height = height
}

func (r *Recorder) recordResize(msg tea.WindowSizeMsg) {
	r.width = msg.Width
	r.height = msg.Height
	r.tape = append(r.tape, fmt.Sprintf("resize %d %d", msg.Width, msg.Height))
}

func roundDuration(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 100 {
		return "100ms"
	}
	rounded := (ms / 100) * 100
	return time.Duration(rounded * int64(time.Millisecond)).String()
}
