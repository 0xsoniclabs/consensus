package election

// heapBuffer is a min-heap of Atropos decisions ordered by Frames.
// it is an easy to maintain structure that keeps continuous sequences (possibly multiple patches of them)
// together and allows for efficient delivery of whole sequence when min frame Atropos of sequence aligns with 'frameToDeliver'
type heapBuffer []*AtroposDecision

func (h heapBuffer) Len() int           { return len(h) }
func (h heapBuffer) Less(i, j int) bool { return h[i].Frame < h[j].Frame }
func (h heapBuffer) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *heapBuffer) Push(x any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*AtroposDecision))
}

func (h *heapBuffer) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
