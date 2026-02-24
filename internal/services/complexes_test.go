package services

import "testing"

func TestNextComplexStatus(t *testing.T) {
	tests := []struct {
		name            string
		currentStatus   string
		newCount        int
		threshold       int
		thresholdStatus string
		want            string
	}{
		{
			name:            "planning becomes collecting on first request",
			currentStatus:   complexStatusPlanning,
			newCount:        1,
			threshold:       10,
			thresholdStatus: "PLANNED",
			want:            complexStatusCollecting,
		},
		{
			name:            "non planning switches to threshold status",
			currentStatus:   "COLLECTING",
			newCount:        10,
			threshold:       10,
			thresholdStatus: "PLANNED",
			want:            "PLANNED",
		},
		{
			name:            "non planning keeps current status below threshold",
			currentStatus:   "COLLECTING",
			newCount:        3,
			threshold:       10,
			thresholdStatus: "PLANNED",
			want:            "COLLECTING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextComplexStatus(tt.currentStatus, tt.newCount, tt.threshold, tt.thresholdStatus)
			if got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}
