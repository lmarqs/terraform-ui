package sdk

import (
	"testing"
	"time"
)

func TestStateLock_Age(t *testing.T) {
	lock := &StateLock{
		Created: time.Now().Add(-5 * time.Minute),
	}

	age := lock.Age()
	if age < 4*time.Minute || age > 6*time.Minute {
		t.Errorf("Age() = %v, want approximately 5 minutes", age)
	}
}
