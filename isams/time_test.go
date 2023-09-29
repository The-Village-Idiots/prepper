package isams

import (
	"testing"
	"time"
)

func TestNewTime(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		wantHour   int
		wantMinute int
		wantErr    bool
	}{
		{"basic (AM)", "07:30", 7, 30, false},
		{"invalid (AM)", "0A:30", 0, 0, true},

		{"basic (PM)", "15:30", 15, 30, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTime(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantHour != time.Time(got).Hour() || tt.wantMinute != time.Time(got).Minute() {
				t.Errorf("NewTime() = %v, want h:%v m:%v", got, tt.wantHour, tt.wantMinute)
			}
		})
	}
}
