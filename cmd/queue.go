package cmd

import "sync"

type Queue struct {
	sync.Mutex
	data []int
}

func (q *Queue) Push(value int) {
	q.Lock()
	defer q.Unlock()
	q.data = append(q.data, value)
}

func (q *Queue) Pop() (item *int) {
	q.Lock()
	defer q.Unlock()
	if len(q.data) > 0 {
		item = &q.data[0]
		q.data = q.data[1:]
	}
	return
}

func (q *Queue) Size() int {
	return len(q.data)
}
