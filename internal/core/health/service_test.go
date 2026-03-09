package health

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name string
		b    uint64
		want string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"below mib", 500_000, "500000 B"},
		{"one mib", 1 << 20, "1.0 MB"},
		{"megabytes", 100 * (1 << 20), "100.0 MB"},
		{"one gig", 1 << 30, "1.0 GB"},
		{"multi gig", 5_368_709_120, "5.0 GB"},
		{"large", 100 * (1 << 30), "100.0 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.b)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.b, got, tt.want)
			}
		})
	}
}

func TestJoinIssues(t *testing.T) {
	tests := []struct {
		name   string
		issues []string
		want   string
	}{
		{"empty", nil, ""},
		{"single", []string{"disk full"}, "disk full"},
		{"multiple", []string{"disk full", "client down"}, "disk full; client down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinIssues(tt.issues)
			if got != tt.want {
				t.Errorf("joinIssues(%v) = %q, want %q", tt.issues, got, tt.want)
			}
		})
	}
}

func TestOverallStatus_Aggregation(t *testing.T) {
	tests := []struct {
		name   string
		checks []CheckResult
		want   Status
	}{
		{
			"all healthy",
			[]CheckResult{
				{Status: StatusHealthy},
				{Status: StatusHealthy},
			},
			StatusHealthy,
		},
		{
			"one degraded",
			[]CheckResult{
				{Status: StatusHealthy},
				{Status: StatusDegraded},
			},
			StatusDegraded,
		},
		{
			"one unhealthy trumps degraded",
			[]CheckResult{
				{Status: StatusDegraded},
				{Status: StatusUnhealthy},
				{Status: StatusHealthy},
			},
			StatusUnhealthy,
		},
		{
			"empty checks",
			[]CheckResult{},
			StatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the aggregation logic from Check() to test it in isolation.
			overall := StatusHealthy
			for _, c := range tt.checks {
				if c.Status == StatusUnhealthy {
					overall = StatusUnhealthy
					break
				}
				if c.Status == StatusDegraded && overall != StatusUnhealthy {
					overall = StatusDegraded
				}
			}
			if overall != tt.want {
				t.Errorf("overall = %q, want %q", overall, tt.want)
			}
		})
	}
}
