package jp2

// Range simply stores a start and end point for our rate queue
type Range struct {
	start int
	end   int
}

// EmptyRange is just a range with the zero values for start and end, aliased
// as "EmptyRange" for clarity
var EmptyRange = Range{}

// RangeQueue encompasses appending and shifting onto a slice of ranges
type RangeQueue struct {
	seen  map[Range]bool
	queue []Range
}

// Append adds a new Range of a - b to the end of the queue unless it's been
// added before
func (rq *RangeQueue) Append(a, b int) {
	if rq.seen == nil {
		rq.seen = make(map[Range]bool)
	}

	var r = Range{a, b}
	if rq.seen[r] {
		return
	}
	rq.seen[r] = true
	rq.queue = append(rq.queue, r)
}

// Shift takes the first entry off the queue and returns it.  If there are no
// values to return, the EmptyRange is returned instead
func (rq *RangeQueue) Shift() Range {
	var l = len(rq.queue)
	if l == 0 {
		return EmptyRange
	}

	var r Range
	r, rq.queue = rq.queue[0], rq.queue[1:]
	return r
}
