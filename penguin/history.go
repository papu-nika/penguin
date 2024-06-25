package penguin

type History struct {
	Seq  int
	Next *History
}

func NewHistory(len int) *History {
	var history History
	firstHistory := &history
	for len > 0 {
		var h History
		h.Next = &history
		history = h
	}
	return firstHistory
}
