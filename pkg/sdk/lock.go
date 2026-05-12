package sdk

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	lockIDRegex        = regexp.MustCompile(`(?m)^\s+ID:\s+([0-9a-f-]+)`)
	lockPathRegex      = regexp.MustCompile(`(?m)^\s+Path:\s+(.+)`)
	lockOperationRegex = regexp.MustCompile(`(?m)^\s+Operation:\s+(.+)`)
	lockWhoRegex       = regexp.MustCompile(`(?m)^\s+Who:\s+(.+)`)
	lockVersionRegex   = regexp.MustCompile(`(?m)^\s+Version:\s+(.+)`)
	lockCreatedRegex   = regexp.MustCompile(`(?m)^\s+Created:\s+(.+)`)
)

// ParseLockError extracts lock info from a terraform error message.
// Returns nil if the error is not a lock error.
func ParseLockError(errMsg string) *StateLock {
	if !strings.Contains(errMsg, "Error acquiring the state lock") {
		return nil
	}

	lock := &StateLock{}

	if m := lockIDRegex.FindStringSubmatch(errMsg); len(m) > 1 {
		lock.ID = strings.TrimSpace(m[1])
	}
	if m := lockPathRegex.FindStringSubmatch(errMsg); len(m) > 1 {
		lock.Path = strings.TrimSpace(m[1])
	}
	if m := lockOperationRegex.FindStringSubmatch(errMsg); len(m) > 1 {
		lock.Operation = strings.TrimSpace(m[1])
	}
	if m := lockWhoRegex.FindStringSubmatch(errMsg); len(m) > 1 {
		lock.Who = strings.TrimSpace(m[1])
	}
	if m := lockVersionRegex.FindStringSubmatch(errMsg); len(m) > 1 {
		lock.Version = strings.TrimSpace(m[1])
	}
	if m := lockCreatedRegex.FindStringSubmatch(errMsg); len(m) > 1 {
		t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", strings.TrimSpace(m[1]))
		if err == nil {
			lock.Created = t
		}
	}

	// Must have at least an ID to be considered a valid lock error
	if lock.ID == "" {
		return nil
	}

	return lock
}

// FormatLockInfo renders lock details for display.
func FormatLockInfo(lock *StateLock) string {
	if lock == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("State Lock Detected\n")
	b.WriteString(strings.Repeat("-", 40) + "\n")
	fmt.Fprintf(&b, "  Lock ID:    %s\n", lock.ID)
	if lock.Who != "" {
		fmt.Fprintf(&b, "  Who:        %s\n", lock.Who)
	}
	if lock.Operation != "" {
		fmt.Fprintf(&b, "  Operation:  %s\n", lock.Operation)
	}
	if !lock.Created.IsZero() {
		fmt.Fprintf(&b, "  Created:    %s\n", lock.Created.Format(time.RFC3339))
		fmt.Fprintf(&b, "  Age:        %s\n", formatLockAge(lock.Age()))
	}
	if lock.Path != "" {
		fmt.Fprintf(&b, "  Path:       %s\n", lock.Path)
	}
	if lock.Version != "" {
		fmt.Fprintf(&b, "  Version:    %s\n", lock.Version)
	}

	return b.String()
}

// formatLockAge returns a human-friendly duration string for lock age.
func formatLockAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours >= 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh%dm", hours, minutes)
}
