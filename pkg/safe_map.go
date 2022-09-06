package pkg

import "sync"

type SafeMap struct {
	sync.Mutex
	data map[int]string
}

func NewSafeMap(count int) (safeMap *SafeMap) {
	safeMap = &SafeMap{data: make(map[int]string, count)}
	for i := 0; i < count; i++ {
		safeMap.Put(i, "")
	}
	return
}

func (m *SafeMap) Put(k int, val string) {
	m.Lock()
	defer m.Unlock()
	m.data[k] = val
}

func (m *SafeMap) Remove(k int) {
	m.Lock()
	defer m.Unlock()
	delete(m.data, k)
}

func (m *SafeMap) Get() *int {
	m.Lock()
	defer m.Unlock()
	for k, _ := range m.data {
		return &k
	}
	return nil
}

func (m *SafeMap) GetKeys() (result []int) {
	m.Lock()
	defer m.Unlock()
	for k, _ := range m.data {
		result = append(result, k)
	}
	return
}

func (m *SafeMap) GetAndRemove() (target *int) {
	m.Lock()
	defer m.Unlock()
	for k, _ := range m.data {
		target = &k
		break
	}
	if target != nil {
		delete(m.data, *target)
	}
	return
}

func (m *SafeMap) GetLowestAndRemove() (target *int) {
	m.Lock()
	defer m.Unlock()
	for k := range m.data {
		if target == nil || k < *target {
			target = &k
		}
	}
	if target != nil {
		delete(m.data, *target)
	}
	return
}

func (m *SafeMap) Size() int {
	m.Lock()
	defer m.Unlock()
	return len(m.data)
}
