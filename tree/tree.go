package tree

import (
	"fmt"
	"strings"
)

type TreeData interface {
	Compare(other any) int
	String() string
}

type BinaryTree interface {
	Insert(TreeData)
	GetData() TreeData
	GetLeft() BinaryTree
	GetRight() BinaryTree
}

func FindInTree(bt BinaryTree, data TreeData) BinaryTree {
	if data.Compare(bt.GetData()) == 0 {
		return bt
	}

	if data.Compare(bt.GetData()) < 0 {
		return FindInTree(bt.GetLeft(), data)
	}

	return FindInTree(bt.GetRight(), data)
}

func PrintTree(bt BinaryTree) string {
	lines, _ := GenerateChildrenLines(bt)
	response := strings.Join(lines, "\n") + "\n"
	fmt.Print(response)
	return response
}

func GenerateChildrenLines(bt BinaryTree) ([]string, int) {
	leftLines := []string{}
	leftRootPosition := 0
	rightLines := []string{}
	rightRootPosition := 0
	leftBlockWidth := 0
	rightBlockWidth := 0

	if bt == nil{
		return []string{}, 0
	}

	lft := bt.GetLeft()  
	fmt.Printf("V: %v, T: %T\n", lft, lft)
	if lft != nil{
		leftLines, leftRootPosition = GenerateChildrenLines(bt.GetLeft())
		if len(leftLines) > 0 {
		leftBlockWidth = len(leftLines[0])
	   }
	}

	if bt.GetRight() != nil {
		rightLines, rightRootPosition = GenerateChildrenLines(bt.GetRight())
		if len(rightLines) > 0 {
		rightBlockWidth = len(rightLines[0])
		}
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

	intdata := bt.GetData()
	fmt.Printf("V: %v, T: %T \n", intdata, intdata)
	strdata :=  intdata.String()
	fmt.Printf("V: %v, T: %T \n", strdata, strdata)
	
	rootLine, rootPosition := fillRootLine(bt.GetData().String(), leftRootPosition, rightRootPosition, leftBlockWidth, rightBlockWidth)

	retLines := []string{rootLine}
	for i := range len(leftLines) {
		retLines = append(retLines, leftLines[i]+fillWithSpaces(len(rootLine)-len(leftLines[i])-len(rightLines[i]))+rightLines[i])
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
