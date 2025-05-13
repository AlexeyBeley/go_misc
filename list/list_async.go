package list

import (
	"sync"
	"time"
)

type AsyncList struct {
	Data string
	Next *AsyncList
}

type InsertLock struct {
	Mutex sync.Mutex
}

func (l *InsertLock) Lock() {
	l.Mutex.Lock()
}

func (l *InsertLock) Unlock() {
	l.Mutex.Unlock()
}

func (l *AsyncList) Insert(dataList []string) {
	start := time.Now()
	var wg sync.WaitGroup

	locker := &InsertLock{}
	for _, data := range dataList {
		lg.DebugF("trigger writing data: %v\n", string(data))
		wg.Add(1)
		go l.InsertSingle(data, locker, &wg)
		lg.DebugF("moving to next after data: %v\n", string(data))
	}
	wg.Wait()

	elapsed := time.Since(start)
	lg.InfoF("Function execution time: %v", elapsed)
}

func (l *AsyncList) InsertSingle(data string, locker *InsertLock, wg *sync.WaitGroup) {

	if l.Next != nil {
		l.Next.InsertSingle(data, locker, wg)
		return
	}

	lg.DebugF("func InsertSingle. mutex: %v, data: %v\n", locker, data)

	locker.Lock()
	if l.Next != nil {
		locker.Unlock()
		l.Next.InsertSingle(data, locker, wg)
		return
	}
	defer wg.Done()
	newLink := AsyncList{Data: data}
	l.Next = &newLink
	locker.Unlock()

}

func (l *AsyncList) Print() {
	for l.Next != nil {
		lg.InfoF("Link data: %v\n", l.Data)
		l = l.Next
	}

	lg.InfoF("Link data: %v\n", l.Data)
}

func (l *AsyncList) Len() int {
	len := 0
	for l.Next != nil {
		l = l.Next
		len += 1
	}
	return len
}
