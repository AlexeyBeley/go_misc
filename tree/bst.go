package tree

import (
	"strconv"

	"github.com/AlexeyBeley/go_common/logger"
)

var lg = &(logger.Logger{Level: logger.INFO})

type IntData int

func (data IntData) Compare(other any) int {
	otherValue, ok := other.(IntData)
	if ok {

		return int(data) - int(otherValue)
	}

	panic(other)
}

func (data IntData) String() string {
	return strconv.Itoa(int(data))
}

type BST struct {
	Data  IntData
	Left  *BST
	Right *BST
}

func (t *BST) GetData() TreeData {
	return t.Data
}

func (t *BST) GetLeft() BinaryTree {
	if t.Left == nil {
		return nil
	}
	return t.Left
}

func (t *BST) GetRight() BinaryTree {
	if t.Right == nil {
		return nil
	}
	return t.Right
}

func (t *BST) Insert(data TreeData) {
	IntDataValue, ok := data.(IntData)
	if !ok {
		panic(data)
	}

	lg.Debugf("In %d inserting data: %d\n", t.Data, data)
	if t.Data.Compare(data) > 0 {
		if t.Left == nil {
			lg.Debugf("In %d creating left\n", t.Data)
			t.Left = &BST{Data: IntDataValue}
			return
		}
		t.Left.Insert(data)
		return
	}

	if t.Right == nil {
		lg.Debugf("In %d creating right", t.Data)
		t.Right = &BST{Data: IntDataValue}
		return
	}
	t.Right.Insert(data)

}

func (t *BST) Depth() int {
	depth := 1
	if t.Left != nil {
		depth = t.Left.Depth() + 1
	}

	if t.Right != nil {
		depth = max(depth, t.Right.Depth()+1)
	}

	return depth
}
