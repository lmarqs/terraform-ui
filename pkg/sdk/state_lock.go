package sdk

import "time"

// StateLock represents an active terraform state lock.
type StateLock struct {
	ID        string
	Path      string
	Operation string
	Who       string
	Version   string
	Created   time.Time
}

// Age returns how old the lock is.
func (l *StateLock) Age() time.Duration {
	return time.Since(l.Created)
}
