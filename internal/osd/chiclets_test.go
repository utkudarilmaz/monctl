package osd

import "testing"

func TestChiclets(t *testing.T) {
	tests := []struct{ value, limit, want int }{
		{0, 100, 0}, {100, 100, 16}, {50, 100, 8}, {5, 100, 1}, {3, 100, 0}, {97, 100, 16},
		{128, 255, 8}, {300, 100, 16}, {-5, 100, 0}, {10, 0, 0},
	}
	for _, tt := range tests {
		filled, boxes := Chiclets(tt.value, tt.limit)
		if filled != tt.want || boxes != 16 {
			t.Errorf("Chiclets(%d, %d) = %d of %d, want %d of 16", tt.value, tt.limit, filled, boxes, tt.want)
		}
	}
}
