package maintenance

import (
	"context"
	"testing"
	"time"
)

// mustime parses a time string, panicking if parse fails. stands for must
// parse time.
func mustime(src string) time.Time {
	t, err := time.Parse(time.RFC3339, src)
	if err != nil {
		panic("mustime parse failed: " + err.Error())
	}

	return t
}

func TestTimeResolution(t *testing.T) {
	tests := []struct {
		T      time.Time
		Expect time.Duration
	}{
		{mustime("2023-11-06T21:26:38Z"), time.Second},
		{mustime("2023-11-06T21:26:00Z"), time.Minute},
		{mustime("2023-11-06T21:00:00Z"), time.Hour},

		{mustime("2023-11-06T00:00:00Z"), time.Hour},
	}

	for _, tt := range tests {
		if r := timeResolution(tt.T); r != tt.Expect {
			t.Errorf("bad resolution (t=%v, res=%v, expect=%v)", tt.T, r, tt.Expect)
		}
	}
}

func TestDailyInterval(t *testing.T) {
	// Create a fake daily time which is on the previous day to test day
	// truncation.
	d := Daily{
		Time: time.Now().Add(time.Second).Add(24 * -time.Hour),
	}

	d.Start(context.Background())
	defer d.Stop()

	start := time.Now()
	select {
	case <-d.Chan():
		t.Logf("got reply at %v (which was %v after test start with a target of exactly 1s)", time.Now(), time.Since(start))
	case ti := <-time.After(3 * time.Second):
		t.Errorf("waited until %v for daily task scheduled for %v", ti, d.Time)
	}
}
