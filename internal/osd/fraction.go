package osd

// Fraction converts a brightness to a share between 0 and 1.
func Fraction(value, limit int) float64 {
	if limit <= 0 {
		return 0
	}
	return float64(max(0, min(value, limit))) / float64(limit)
}
