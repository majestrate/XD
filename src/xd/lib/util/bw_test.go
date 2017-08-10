package util

import "testing"

func TestFormatRate(t *testing.T) {

	rate := float32(1000000.5)

	t.Logf("rate %f %s", rate, FormatRate(rate))
}
