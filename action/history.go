package penguin

// History is a linked list of ping packet sequence numbers.
type History struct {
	Data   []HistoryData
	MaxLen int
}

type HistoryData struct {
	Seq        int
	isRecieved bool
}

// Remove the first element if the history is full
func (h *History) AppendFirst(data HistoryData) *History {
	if len(h.Data) == h.MaxLen {
		h.Data = h.Data[1:]
	}
	h.Data = append(h.Data, data)
	return h
}

func (h *History) Recieved(seq int) {
	for i, d := range h.Data {
		if d.Seq == seq {
			h.Data[i].isRecieved = true
		}
	}
}

func (h *History) IsFull() bool {
	return len(h.Data) == h.MaxLen
}

// 指定のインデックスが成功かどうかを返す
func (h *History) IsSuccessIndex(i int) bool {
	if i < 0 || i >= len(h.Data) {
		return false
	}
	return h.Data[i].isRecieved
}

func (h History) Len() int {
	return len(h.Data)
}
