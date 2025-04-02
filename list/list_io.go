package list

import (
	"fmt"
	"io"
	"log"
)

type IOList struct {
	Data []byte
	Next *IOList
}

func (l *IOList) Insert(data io.ReadWriter) {
	for l.Next != nil {
		l = l.Next
	}
	dataBytes := make([]byte, 10)
	n, err := data.Read(dataBytes)
	if err != nil {
		log.Fatalf("Failed to read data after: %d bytes", n)
	}

	newLink := IOList{Data: dataBytes[:n]}
	l.Next = &newLink
}

func (l *IOList) Print() {
	for l.Next != nil {
		fmt.Printf("Link data: %v\n", l.Data)
		l = l.Next
	}
	fmt.Printf("Link data: %v\n", l.Data)
}
