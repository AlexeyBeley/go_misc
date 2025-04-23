package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"strconv"

	"github.com/AlexeyBeley/go_common/logger"
	"github.com/AlexeyBeley/go_misc/list"
	"github.com/AlexeyBeley/go_misc/tree"
)

var lg = &(logger.Logger{})

type Addable interface {
	GetNext() Addable
	SetNext(Addable)
	Count() int
}

type Internal struct {
	Data   string
	NextIn Addable
}

func (something *Internal) GetNext() Addable {
	return something.NextIn
}

func (something *Internal) SetNext(nextAddable Addable) {
	something.NextIn = nextAddable
}

func (something *Internal) Count() int {
	if something.NextIn == nil {
		return 1
	}
	return 1 + something.NextIn.Count()
}

func Add(something Addable, toAdd Addable) {
	if something.GetNext() == nil {
		something.SetNext(toAdd)
		return
	}
	Add(something.GetNext(), toAdd)
}

func Test() {
	something := &Internal{}
	Add(something, &Internal{})
	Add(something, &Internal{})
	Add(something, &Internal{})

	lg.Infof("Count: %d", something.Count())

}

func testListString() {
	myList := list.StringList{Data: "1"}
	fmt.Printf("Hello world: %v\n", myList)

	for i := range 10 {
		myList.Insert(strconv.Itoa(i))
	}

	myList.Print()
}

func testListIO() {
	myList := list.IOList{Data: []byte("1")}
	fmt.Printf("Hello world: %v\n", myList)

	for i := range 10 {
		myList.Insert(bytes.NewBufferString(strconv.Itoa(i)))
	}

	myList.Print()
}

func testListAsync() {
	//measure execution
	// log level debug/info
	runtime.GOMAXPROCS(100)
	myList := list.AsyncList{Data: "1"}
	fmt.Printf("Hello world: %v\n", myList)

	lstData := [100000]string{}
	for i := range lstData {
		lstData[i] = strconv.Itoa(i)
	}
	myList.Insert(lstData[:])

	//myList.Print()
	log.Printf("Len: %v\n", myList.Len())
}

func testBST() {
	//async counter
	itemsNum := 20
	var data tree.IntData = tree.IntData(itemsNum / 40)
	myTree := tree.BST{Data: data}
	fmt.Printf("Hello world: %v\n", myTree)

	for range itemsNum {
		myTree.Insert(tree.IntData(rand.Intn(itemsNum / 2)))
	}

	//myList.Print()
	log.Printf("Depth: %d\n", myTree.Depth())

	tree.PrintTree(&myTree)
}

func testAVLTree() {
	//async counter
	/*	itemsNum := 20

		myTree := tree.AVLTree{BST: tree.BST{Data: itemsNum / 4}}

		fmt.Printf("Hello world: %v\n", myTree)

		for range itemsNum {
			data := rand.Intn(itemsNum / 2)
			myTree.Insert(data)
			myTree.Print()
		}
	*/
	//myList.Print()
	//log.Printf("Depth: %d\n", myTree.Depth())
}

func main() {
	// todo 10,000 concurrent go routines to insert to list
	// todo reader/writer simulator into queue to see how slow reader affects the queue
	// todo: async binary search with pool of goroutines and a stop signal to stop the search
	fmt.Printf("%T\n", testListString)
	fmt.Printf("%T\n", testListIO)
	fmt.Printf("%T\n", testListAsync)
	fmt.Printf("%T\n", testBST)
	fmt.Printf("%T\n", testAVLTree)
	fmt.Printf("%T\n", Test)
	//testBST()
}
