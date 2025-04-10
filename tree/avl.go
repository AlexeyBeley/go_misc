package tree

type AVLTree struct {
	BST
	Left  *AVLTree
	Right *AVLTree
}

func (t *AVLTree) Insert(data TreeData) {
	intDataValue, ok := data.(IntData)
	if !ok {
		panic(data)
	}
	lg.Debugf("In %d inserting data: %d\n", t.Data, data)
	if intDataValue <= t.Data {
		if t.Left == nil {
			lg.Debugf("In %d creating left\n", t.Data)
			t.Left = &AVLTree{BST: BST{Data: intDataValue}}
			return
		}
		t.Left.Insert(data)
		return
	}

	if t.Right == nil {
		lg.Debugf("In %d creating right", t.Data)
		t.Right = &AVLTree{BST: BST{Data: intDataValue}}
		return
	}
	t.Right.Insert(data)

}
