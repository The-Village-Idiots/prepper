package logging

import (
	"testing"
	"unicode/utf8"
)

func TestStripColors(t *testing.T) {
	tests := []struct {
		data   string
		expect string
	}{
		{
			"2023/10/28 19:04:45 /home/user/prepper/main.go:216 [0m[33m[0.048ms] [rows:-] SELECT DATABASE()",
			"2023/10/28 19:04:45 /home/user/prepper/main.go:216 [0.048ms] [rows:-] SELECT DATABASE()",
		},
		{
			"abcd\nefgh",
			"abcd\nefgh",
		},
	}

	for _, tt := range tests {
		a := StripColors([]byte(tt.data))

		if !utf8.Valid(a) {
			t.Errorf("invalid utf-8 returned (input: %q, output: %q)", tt.data, tt.expect)
		}

		if string(a) != tt.expect {
			t.Errorf("invalid response (expect: %q, got: %q)", tt.expect, string(a))
		}
	}
}
