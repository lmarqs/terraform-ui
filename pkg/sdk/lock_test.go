package sdk

import (
	"strings"
	"testing"
	"time"
)

func TestParseLockError_ValidLockError(t *testing.T) {
	errMsg := `running terraform plan: Error acquiring the state lock

Lock Info:
  ID:        17481e0a-f9c6-c65b-ee64-00c83eab930f
  Path:      medprev-iac-terraformstatebucket-1jx8qx8mjxe38/sa-east-1/terraform.tfstate
  Operation: OperationTypePlan
  Who:       lmarqs@omarchy
  Version:   1.14.9
  Created:   2026-05-09 05:59:28.382300067 +0000 UTC

Terraform acquires a state lock to protect the state from being written
by multiple users at the same time. Please resolve the issue above and try
again. For most commands, you can disable locking with the "-lock=false"
flag, but this is not recommended.`

	lock := ParseLockError(errMsg)
	if lock == nil {
		t.Fatal("ParseLockError returned nil, want non-nil")
	}

	if lock.ID != "17481e0a-f9c6-c65b-ee64-00c83eab930f" {
		t.Errorf("ID = %q, want %q", lock.ID, "17481e0a-f9c6-c65b-ee64-00c83eab930f")
	}
	if lock.Path != "medprev-iac-terraformstatebucket-1jx8qx8mjxe38/sa-east-1/terraform.tfstate" {
		t.Errorf("Path = %q, want %q", lock.Path, "medprev-iac-terraformstatebucket-1jx8qx8mjxe38/sa-east-1/terraform.tfstate")
	}
	if lock.Operation != "OperationTypePlan" {
		t.Errorf("Operation = %q, want %q", lock.Operation, "OperationTypePlan")
	}
	if lock.Who != "lmarqs@omarchy" {
		t.Errorf("Who = %q, want %q", lock.Who, "lmarqs@omarchy")
	}
	if lock.Version != "1.14.9" {
		t.Errorf("Version = %q, want %q", lock.Version, "1.14.9")
	}
	if lock.Created.IsZero() {
		t.Error("Created is zero, want parsed time")
	}
	expectedTime := time.Date(2026, 5, 9, 5, 59, 28, 382300067, time.UTC)
	if !lock.Created.Equal(expectedTime) {
		t.Errorf("Created = %v, want %v", lock.Created, expectedTime)
	}
}

func TestParseLockError_NotALockError(t *testing.T) {
	cases := []struct {
		name   string
		errMsg string
	}{
		{"empty string", ""},
		{"generic error", "Error: something went wrong"},
		{"partial match", "Error acquiring the state"},
		{"permission error", "Error: Insufficient permissions to access state"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lock := ParseLockError(tc.errMsg)
			if lock != nil {
				t.Errorf("ParseLockError(%q) = %+v, want nil", tc.errMsg, lock)
			}
		})
	}
}

func TestParseLockError_LockErrorWithoutID(t *testing.T) {
	errMsg := `Error acquiring the state lock

Lock Info:
  Path: some/path
  Operation: OperationTypePlan`

	lock := ParseLockError(errMsg)
	if lock != nil {
		t.Errorf("ParseLockError without ID = %+v, want nil", lock)
	}
}

func TestParseLockError_MinimalLockError(t *testing.T) {
	errMsg := `Error acquiring the state lock

Lock Info:
  ID:        abc-123`

	lock := ParseLockError(errMsg)
	if lock == nil {
		t.Fatal("ParseLockError returned nil, want non-nil")
	}
	if lock.ID != "abc-123" {
		t.Errorf("ID = %q, want %q", lock.ID, "abc-123")
	}
	if lock.Path != "" {
		t.Errorf("Path = %q, want empty", lock.Path)
	}
	if lock.Who != "" {
		t.Errorf("Who = %q, want empty", lock.Who)
	}
}

func TestFormatLockInfo_Nil(t *testing.T) {
	result := FormatLockInfo(nil)
	if result != "" {
		t.Errorf("FormatLockInfo(nil) = %q, want empty", result)
	}
}

func TestFormatLockInfo_Full(t *testing.T) {
	lock := &StateLock{
		ID:        "17481e0a-f9c6-c65b-ee64-00c83eab930f",
		Path:      "bucket/path/terraform.tfstate",
		Operation: "OperationTypePlan",
		Who:       "lmarqs@omarchy",
		Version:   "1.14.9",
		Created:   time.Date(2026, 5, 9, 5, 59, 28, 0, time.UTC),
	}

	result := FormatLockInfo(lock)

	if !strings.Contains(result, "State Lock Detected") {
		t.Error("FormatLockInfo missing title")
	}
	if !strings.Contains(result, lock.ID) {
		t.Error("FormatLockInfo missing lock ID")
	}
	if !strings.Contains(result, lock.Who) {
		t.Error("FormatLockInfo missing Who")
	}
	if !strings.Contains(result, lock.Operation) {
		t.Error("FormatLockInfo missing Operation")
	}
	if !strings.Contains(result, lock.Path) {
		t.Error("FormatLockInfo missing Path")
	}
	if !strings.Contains(result, lock.Version) {
		t.Error("FormatLockInfo missing Version")
	}
	if !strings.Contains(result, "Age:") {
		t.Error("FormatLockInfo missing Age")
	}
}

func TestFormatLockInfo_Minimal(t *testing.T) {
	lock := &StateLock{
		ID: "abc-123",
	}

	result := FormatLockInfo(lock)
	if !strings.Contains(result, "abc-123") {
		t.Error("FormatLockInfo missing lock ID")
	}
	// Should not contain empty labels
	if strings.Contains(result, "Who:") {
		t.Error("FormatLockInfo should not show Who when empty")
	}
	if strings.Contains(result, "Operation:") {
		t.Error("FormatLockInfo should not show Operation when empty")
	}
}

func TestFormatLockAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{65 * time.Minute, "1h5m"},
		{25 * time.Hour, "1d1h0m"},
		{49 * time.Hour, "2d1h0m"},
	}

	for _, tc := range cases {
		got := formatLockAge(tc.d)
		if got != tc.want {
			t.Errorf("formatLockAge(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
