package sdk

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	lockIDRegex        = regexp.MustCompile(`ID:\s+(.+)`)
	lockPathRegex      = regexp.MustCompile(`Path:\s+(.+)`)
	lockOperationRegex = regexp.MustCompile(`Operation:\s+(.+)`)
	lockWhoRegex       = regexp.MustCompile(`Who:\s+(.+)`)
	lockVersionRegex   = regexp.MustCompile(`Version:\s+(.+)`)
	lockCreatedRegex   = regexp.MustCompile(`Created:\s+(.+)`)
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
	b.WriteString(fmt.Sprintf("  Lock ID:    %s\n", lock.ID))
	if lock.Who != "" {
		b.WriteString(fmt.Sprintf("  Who:        %s\n", lock.Who))
	}
	if lock.Operation != "" {
		b.WriteString(fmt.Sprintf("  Operation:  %s\n", lock.Operation))
	}
	if !lock.Created.IsZero() {
		b.WriteString(fmt.Sprintf("  Created:    %s\n", lock.Created.Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("  Age:        %s\n", formatLockAge(lock.Age())))
	}
	if lock.Path != "" {
		b.WriteString(fmt.Sprintf("  Path:       %s\n", lock.Path))
	}
	if lock.Version != "" {
		b.WriteString(fmt.Sprintf("  Version:    %s\n", lock.Version))
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
