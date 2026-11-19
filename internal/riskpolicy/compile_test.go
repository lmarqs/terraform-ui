package riskpolicy

import (
	"context"
	"testing"
)

func TestCompile_GivenSource_ShouldReportWhetherItCompiledAndKeepTheSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{"valid", `classification := {"level": "low", "reasons": []}`, true},
		{"parse failure", `classification := {`, false},
		{"denied capability", `classification := http.send({"method": "GET", "url": "https://example.invalid"})`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := evaluationSnapshot(t, tc.body)
			policy := Compile(context.Background(), snapshot)
			if policy.Compiled() != tc.want {
				t.Fatalf("Compiled() = %v, want %v", policy.Compiled(), tc.want)
			}
			if policy.Snapshot().Digest() != snapshot.Digest() {
				t.Fatalf("Snapshot().Digest() = %q, want %q", policy.Snapshot().Digest(), snapshot.Digest())
			}
		})
	}
}
