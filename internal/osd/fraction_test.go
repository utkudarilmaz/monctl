package osd

import "testing"

func TestFraction(t *testing.T) {
	cases := []struct {
		value, limit int
		want         float64
	}{
		{50, 100, 0.5},
		{0, 100, 0},
		{100, 100, 1},
		{150, 100, 1},
		{-5, 100, 0},
		{10, 0, 0},
	}
	for _, c := range cases {
		if got := Fraction(c.value, c.limit); got != c.want {
			t.Errorf("Fraction(%d, %d) = %v, want %v", c.value, c.limit, got, c.want)
		}
	}
}
