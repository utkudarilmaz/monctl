package osd

// total is the number of boxes in the system indicator.
const total = 16

// Chiclets converts a brightness to filled boxes of the indicator.
func Chiclets(value, limit int) (filled, boxes int) {
	if limit <= 0 {
		return 0, total
	}
	value = max(0, min(value, limit))
	return (value*total + limit/2) / limit, total
}
