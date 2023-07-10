package fluxauth

import (
	"fmt"
	"testing"
	"time"
)

func Test_deletionTimeNotPassed(t *testing.T) {
	tests := []struct {
		name   string
		time   time.Time
		result bool
	}{
		{
			name: "time passed",
			time: time.Now().Add(-20 * time.Minute),
		},
		{
			name:   "time not passed",
			time:   time.Now().Add(-5 * time.Minute),
			result: true,
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			if tc.result != deletionTimeNotPassed(tc.time) {
				t.Fatalf("Expected deletionTimeNotPassed to be %v for %v", tc.result, tc.time.String())
			}
		})
	}
}
