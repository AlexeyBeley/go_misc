package tree

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AlexeyBeley/go_common/logger"
)

var lg = &(logger.Logger{Level: logger.INFO})

type BinaryTree struct {
	Data  int
	Left  *BinaryTree
	Right *BinaryTree
}

func (t *BinaryTree) Insert(data int) {
	lg.Debugf("In %d inserting data: %d\n", t.Data, data)
	if data <= t.Data {
		if t.Left == nil {
			lg.Debugf("In %d creating left\n", t.Data)
			t.Left = &BinaryTree{Data: data}
			return
		}
		t.Left.Insert(data)
		return
	}

	if t.Right == nil {
		lg.Debugf("In %d creating right", t.Data)
		t.Right = &BinaryTree{Data: data}
		return
	}
	t.Right.Insert(data)

}

func (t *BinaryTree) Depth() int {
	depth := 1
	if t.Left != nil {
		depth = t.Left.Depth() + 1
	}

	if t.Right != nil {
		depth = max(depth, t.Right.Depth()+1)
	}

	return depth
}

func (t *BinaryTree) Print() {
	lines, _ := FillChildren(t)
	fmt.Print(strings.Join(lines, "\n"))
}

func FillChildren(t *BinaryTree) ([]string, int) {
	leftLines := []string{}
	leftRootPosition := 0
	rightLines := []string{}
	rightRootPosition := 0
	leftBlockWidth := 0
	rightBlockWidth := 0

	if t.Left != nil {
		leftLines, leftRootPosition = FillChildren(t.Left)
		leftBlockWidth = len(leftLines[0])
	}

	if t.Right != nil {
		rightLines, rightRootPosition = FillChildren(t.Right)
		rightBlockWidth = len(rightLines[0])
	}

	if len(leftLines) > len(rightLines) {
		for range len(leftLines) - len(rightLines) {
			rightLines = append(rightLines, fillWithSpaces(rightBlockWidth))
		}
	} else if len(leftLines) < len(rightLines) {
		for range len(rightLines) - len(leftLines) {
			leftLines = append(leftLines, fillWithSpaces(leftBlockWidth))
		}
	}

	rootLine, rootPosition := fillRootLine(strconv.Itoa(t.Data), leftRootPosition, rightRootPosition, leftBlockWidth, rightBlockWidth)

	retLines := []string{rootLine}
	for i := range len(leftLines) {
		retLines = append(retLines, leftLines[i]+fillWithSpaces(len(rootLine) - len(leftLines[i]) - len(rightLines[i]))+rightLines[i])
	}

	return retLines, rootPosition
}

func fillWithSpaces(count int) string {
	ret := ""
	for range count {
		ret += " "
	}
	return ret
}
func fillRootLine(data string, leftRootPosition, rightRootPosition, leftBlockWidth, rightBlockWidth int) (string, int) {
	rootLine := data

	rootPosition := (leftRootPosition + (leftBlockWidth + len(rootLine) + rightRootPosition)) / 2
	newLineLength := leftBlockWidth + len(rootLine) + rightBlockWidth
	rootLine = fillWithSpaces(rootPosition) + rootLine + fillWithSpaces(newLineLength-(rootPosition+len(rootLine)))
	return rootLine, rootPosition
}

func getChildrenPerDepth(t *BinaryTree, rootDepth int, mapChildrenPerDepth *map[int]int) {
	if t == nil {
		return
	}
	(*mapChildrenPerDepth)[rootDepth] += 1
	getChildrenPerDepth(t.Left, rootDepth+1, mapChildrenPerDepth)
	getChildrenPerDepth(t.Right, rootDepth+1, mapChildrenPerDepth)

}

func maxMap(srcMap map[int]int) int {
	ret := 0
	for _, value := range srcMap {
		if value > ret {
			ret = value
		}
	}
	return ret
}
