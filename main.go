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

func main() {
	// todo 10,000 concurrent go routines to insert to list
	// todo reader/writer simulator into queue to see how slow reader affects the queue

	fmt.Printf("%v\n", testListString)
	fmt.Printf("%v\n", testListIO)
	fmt.Printf("%v\n", testListAsync)
	testBinaryTree()
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

func testBinaryTree() {
	//async counter
	itemsNum := 100

	myTree := tree.BinaryTree{Data: itemsNum / 4}
	fmt.Printf("Hello world: %v\n", myTree)

	for range itemsNum {
		myTree.Insert(rand.Intn(itemsNum / 2))
	}

	//myList.Print()
	log.Printf("Depth: %d\n", myTree.Depth())

	myTree.Print()
}
