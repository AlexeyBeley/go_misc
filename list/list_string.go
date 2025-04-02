package list

import "fmt"

type StringList struct{
	Data string
	Next *StringList
}


func (l *StringList) Insert(data string){
	for l.Next != nil {
		l = l.Next
	}

	newLink := StringList{Data: data}
	l.Next = &newLink
}

func (l *StringList)Print(){
	for l.Next != nil {
		fmt.Printf("Link data: %v\n", l.Data)
		l = l.Next
	}
	fmt.Printf("Link data: %v\n", l.Data)
}
