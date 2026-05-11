package sdk

import "testing"

func TestStatus_Predicates(t *testing.T) {
	tests := []struct {
		status    Status
		isIdle    bool
		isLoading bool
		isReady   bool
		hasError  bool
	}{
		{StatusIdle, true, false, false, false},
		{StatusLoading, false, true, false, false},
		{StatusDone, false, false, true, false},
		{StatusError, false, false, false, true},
	}

	for _, tt := range tests {
		if tt.status.IsIdle() != tt.isIdle {
			t.Errorf("%v.IsIdle() = %v", tt.status, !tt.isIdle)
		}
		if tt.status.IsLoading() != tt.isLoading {
			t.Errorf("%v.IsLoading() = %v", tt.status, !tt.isLoading)
		}
		if tt.status.IsReady() != tt.isReady {
			t.Errorf("%v.IsReady() = %v", tt.status, !tt.isReady)
		}
		if tt.status.HasError() != tt.hasError {
			t.Errorf("%v.HasError() = %v", tt.status, !tt.hasError)
		}
	}
}

func TestStatus_PluginExtension(t *testing.T) {
	// Plugins can extend with their own constants from offset 10+
	const StatusShowingDetail = Status(10)

	if StatusShowingDetail.IsIdle() || StatusShowingDetail.IsLoading() ||
		StatusShowingDetail.IsReady() || StatusShowingDetail.HasError() {
		t.Error("extended status should not match any base predicates")
	}
}
