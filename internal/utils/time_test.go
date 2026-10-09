package utils

import "testing"

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{-10, "0ms"},
		{0, "0ms"},
		{50, "50ms"},
		{999, "999ms"},
		{1000, "1.00s"},
		{2494, "2.49s"},
		{9990, "9.99s"},
		{10000, "10.0s"},
		{15200, "15.2s"},
		{59900, "59.9s"},
		{60000, "1m0s"},
		{65000, "1m5s"},
		{553116, "9m13s"},
		{3599000, "59m59s"},
		{3600000, "1h0m0s"},
		{3661000, "1h1m1s"},
		{7325000, "2h2m5s"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q; want %q", tt.ms, got, tt.want)
		}
	}
}
