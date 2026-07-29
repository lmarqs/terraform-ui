package sdk

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ApplyCounts is terraform's post-apply resource tally — what actually changed,
// as opposed to what the plan proposed.
type ApplyCounts struct {
	Added     int
	Changed   int
	Destroyed int
}

// String renders the tally in terraform's own wording.
func (c ApplyCounts) String() string {
	return fmt.Sprintf("%d added, %d changed, %d destroyed", c.Added, c.Changed, c.Destroyed)
}

// Add records one completed resource action in the tally.
func (c *ApplyCounts) Add(action Action) {
	switch action {
	case ActionCreate:
		c.Added++
	case ActionUpdate:
		c.Changed++
	case ActionDelete:
		c.Destroyed++
	}
}

// Empty reports whether the tally accounts for no resources at all.
func (c ApplyCounts) Empty() bool {
	return c.Added == 0 && c.Changed == 0 && c.Destroyed == 0
}

// applyTallyField matches one "<n> <verb>" pair of terraform's tally line.
var applyTallyField = regexp.MustCompile(`(\d+) (added|changed|destroyed)`)

// applyProgressLine matches terraform's per-resource completion notice, e.g.
// "aws_instance.web: Creation complete after 3s [id=i-123]".
var applyProgressLine = regexp.MustCompile(`: (Creation|Modifications|Destruction) complete after`)

// ParseApplyProgress reports which action a terraform per-resource completion
// notice finished. terraform emits these as it goes and prints no tally when the
// run fails, so they are the only account of what an interrupted apply managed
// to do.
func ParseApplyProgress(line string) (Action, bool) {
	m := applyProgressLine.FindStringSubmatch(line)
	if m == nil {
		return ActionNoOp, false
	}
	switch m[1] {
	case "Creation":
		return ActionCreate, true
	case "Modifications":
		return ActionUpdate, true
	default:
		return ActionDelete, true
	}
}

// ParseApplyCounts reads the tally out of terraform's completion line
// ("Apply complete! Resources: 1 added, 0 changed, 0 destroyed."). Fields absent
// from the line — a destroy run reports only destroyed — stay zero.
//
// Reports false for any other line, including the plan summary, so a caller can
// tell "terraform did not say" apart from "terraform said nothing changed".
func ParseApplyCounts(line string) (ApplyCounts, bool) {
	if !strings.Contains(line, "Resources:") {
		return ApplyCounts{}, false
	}

	matches := applyTallyField.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return ApplyCounts{}, false
	}

	var counts ApplyCounts
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		switch m[2] {
		case "added":
			counts.Added = n
		case "changed":
			counts.Changed = n
		case "destroyed":
			counts.Destroyed = n
		}
	}
	return counts, true
}

// ApplyTally reads terraform's apply output as it streams and keeps both
// resource tallies: the one terraform prints on completion, and the running
// count of per-resource completion notices that is all a failed run leaves
// behind.
//
// It is an io.Writer so it can ride alongside the UI stream in an
// io.MultiWriter. Reading the tally from the output rather than from the
// BubbleTea message loop is deliberate: in headless mode the streamed lines are
// delivered after the operation's result message, too late to describe it.
type ApplyTally struct {
	mu       sync.Mutex
	buf      []byte
	reported *ApplyCounts
	applied  ApplyCounts
}

// Write splits p on newlines and folds each complete line into the tallies.
func (t *ApplyTally) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	for {
		idx := bytes.IndexByte(t.buf, '\n')
		if idx < 0 {
			break
		}
		t.fold(string(t.buf[:idx]))
		t.buf = t.buf[idx+1:]
	}
	return len(p), nil
}

func (t *ApplyTally) fold(line string) {
	if counts, ok := ParseApplyCounts(line); ok {
		t.reported = &counts
	}
	if action, ok := ParseApplyProgress(line); ok {
		t.applied.Add(action)
	}
}

// Reported returns the tally terraform printed on its completion line, or nil
// when it printed none — which is the case for every failed run.
func (t *ApplyTally) Reported() *ApplyCounts {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.reported
}

// Applied returns the tally of per-resource completion notices seen so far.
func (t *ApplyTally) Applied() ApplyCounts {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.applied
}
