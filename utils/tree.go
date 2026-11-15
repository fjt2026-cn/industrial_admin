package utils

// TreeNode 通用树节点，供部门树、权限树复用
type TreeNode struct {
	ID       uint64      `json:"id"`
	ParentID uint64      `json:"parentId"`
	Name     string      `json:"name"`
	Code     string      `json:"code"`
	Type     int8        `json:"type,omitempty"`
	SortNo   int         `json:"sortNo"`
	Children []*TreeNode `json:"children,omitempty"`
}

// BuildTree 把扁平列表组装成树，parentID 为 0 或找不到父节点时视为顶层
func BuildTree(list []*TreeNode) []*TreeNode {
	if len(list) == 0 {
		return []*TreeNode{}
	}
	index := make(map[uint64]*TreeNode, len(list))
	for _, n := range list {
		n.Children = nil
		index[n.ID] = n
	}
	roots := make([]*TreeNode, 0, len(list))
	for _, n := range list {
		if p, ok := index[n.ParentID]; ok && n.ParentID != 0 && p.ID != n.ID {
			p.Children = append(p.Children, n)
			continue
		}
		roots = append(roots, n)
	}
	return roots
}
