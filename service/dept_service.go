package service

import (
	"errors"

	"gorm.io/gorm"

	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

type DeptService struct{ db *gorm.DB }

func NewDeptService(database *gorm.DB) *DeptService { return &DeptService{db: database} }

// Tree 取全部部门并组装为树，前端直接渲染
func (s *DeptService) Tree() ([]*utils.TreeNode, error) {
	var depts []model.Dept
	if err := s.db.Order("sort_no ASC, id ASC").Find(&depts).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "查询部门失败", err)
	}

	nodes := make([]*utils.TreeNode, 0, len(depts))
	for _, d := range depts {
		var parent uint64
		if d.ParentID != nil {
			parent = *d.ParentID
		}
		nodes = append(nodes, &utils.TreeNode{
			ID: d.ID, ParentID: parent, Name: d.DeptName,
			Code: d.DeptCode, SortNo: d.SortNo,
		})
	}
	return utils.BuildTree(nodes), nil
}

type CreateDeptReq struct {
	ParentID *uint64 `json:"parentId"`
	DeptName string  `json:"deptName" binding:"required"`
	DeptCode string  `json:"deptCode" binding:"required"`
	LeaderID *uint64 `json:"leaderId"`
	SortNo   int     `json:"sortNo"`
	Remark   string  `json:"remark"`
}

// Create 新增部门：编码唯一，同级名称唯一，上级必须存在
func (s *DeptService) Create(req *CreateDeptReq) (uint64, error) {
	if req.ParentID != nil {
		var parentCount int64
		if err := s.db.Model(&model.Dept{}).Where("id = ?", *req.ParentID).Count(&parentCount).Error; err != nil {
			return 0, utils.WrapAppError(response.CodeServerError, "查询上级部门失败", err)
		}
		if parentCount == 0 {
			return 0, utils.NewAppError(response.CodeParamInvalid, "上级部门不存在")
		}
	}

	var dup int64
	if err := s.db.Model(&model.Dept{}).Where("dept_code = ?", req.DeptCode).Count(&dup).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "查询部门失败", err)
	}
	if dup > 0 {
		return 0, utils.NewAppError(response.CodeConflict, "部门编码已存在")
	}

	// 同级名称唯一：由业务层校验，数据库不建唯一约束
	scope := s.db.Model(&model.Dept{}).Where("dept_name = ?", req.DeptName)
	if req.ParentID != nil {
		scope = scope.Where("parent_id = ?", *req.ParentID)
	} else {
		scope = scope.Where("parent_id IS NULL")
	}
	var sameName int64
	if err := scope.Count(&sameName).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "查询部门失败", err)
	}
	if sameName > 0 {
		return 0, utils.NewAppError(response.CodeConflict, "同级已存在同名部门")
	}

	d := model.Dept{
		ParentID: req.ParentID, DeptName: req.DeptName, DeptCode: req.DeptCode,
		LeaderID: req.LeaderID, SortNo: req.SortNo, Remark: req.Remark, Status: 1,
	}
	if err := s.db.Create(&d).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "创建部门失败", err)
	}
	return d.ID, nil
}

// DeptQuery 部门平铺列表查询条件：关键字匹配名称或编码，可按状态过滤
type DeptQuery struct {
	utils.PageQuery
	Keyword string `form:"keyword"`
	Status  *int8  `form:"status"`
}

// List 平铺分页查询：树形展示用 Tree，列表页与导出用 List
func (s *DeptService) List(q *DeptQuery) ([]model.Dept, int64, error) {
	q.Normalize()

	tx := s.db.Model(&model.Dept{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("dept_name LIKE ? OR dept_code LIKE ?", kw, kw)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询部门总数失败", err)
	}

	var rows []model.Dept
	if err := tx.Order("sort_no ASC, id ASC").
		Offset(q.Offset()).Limit(q.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询部门失败", err)
	}
	return rows, total, nil
}

// Detail 部门详情：返回单条记录与该部门的父级名称
func (s *DeptService) Detail(id uint64) (*model.Dept, string, error) {
	var d model.Dept
	if err := s.db.Where("id = ?", id).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", utils.NewAppError(response.CodeNotFound, "部门不存在")
		}
		return nil, "", utils.WrapAppError(response.CodeServerError, "查询部门失败", err)
	}

	parentName := ""
	if d.ParentID != nil {
		var p model.Dept
		if err := s.db.Where("id = ?", *d.ParentID).First(&p).Error; err == nil {
			parentName = p.DeptName
		}
	}
	return &d, parentName, nil
}

// isDescendant 判断 targetID 是否为 rootID 的后代，用于阻止“把父级改到自己下级”形成环
func (s *DeptService) isDescendant(rootID, targetID uint64) (bool, error) {
	cur := targetID
	for i := 0; i < 32; i++ { // 层级上限保护，避免脏数据造成死循环
		var d model.Dept
		if err := s.db.Where("id = ?", cur).First(&d).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		if d.ParentID == nil {
			return false, nil
		}
		if *d.ParentID == rootID {
			return true, nil
		}
		cur = *d.ParentID
	}
	return false, nil
}

// Update 修改部门：编码与同级名称仍需唯一，且不允许把上级调整为自己的下级
func (s *DeptService) Update(id uint64, req *CreateDeptReq) error {
	var d model.Dept
	if err := s.db.Where("id = ?", id).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "部门不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询部门失败", err)
	}

	if req.ParentID != nil {
		if *req.ParentID == id {
			return utils.NewAppError(response.CodeParamInvalid, "上级部门不能是自身")
		}
		var parentCount int64
		if err := s.db.Model(&model.Dept{}).Where("id = ?", *req.ParentID).
			Count(&parentCount).Error; err != nil {
			return utils.WrapAppError(response.CodeServerError, "查询上级部门失败", err)
		}
		if parentCount == 0 {
			return utils.NewAppError(response.CodeParamInvalid, "上级部门不存在")
		}
		desc, err := s.isDescendant(id, *req.ParentID)
		if err != nil {
			return utils.WrapAppError(response.CodeServerError, "查询部门层级失败", err)
		}
		if desc {
			return utils.NewAppError(response.CodeParamInvalid, "上级部门不能是当前部门的下级")
		}
	}

	var dup int64
	if err := s.db.Model(&model.Dept{}).
		Where("dept_code = ? AND id <> ?", req.DeptCode, id).Count(&dup).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "查询部门失败", err)
	}
	if dup > 0 {
		return utils.NewAppError(response.CodeConflict, "部门编码已存在")
	}

	scope := s.db.Model(&model.Dept{}).Where("dept_name = ? AND id <> ?", req.DeptName, id)
	if req.ParentID != nil {
		scope = scope.Where("parent_id = ?", *req.ParentID)
	} else {
		scope = scope.Where("parent_id IS NULL")
	}
	var sameName int64
	if err := scope.Count(&sameName).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "查询部门失败", err)
	}
	if sameName > 0 {
		return utils.NewAppError(response.CodeConflict, "同级已存在同名部门")
	}

	fields := map[string]any{
		"parent_id": req.ParentID, "dept_name": req.DeptName, "dept_code": req.DeptCode,
		"leader_id": req.LeaderID, "sort_no": req.SortNo, "remark": req.Remark,
	}
	if err := s.db.Model(&model.Dept{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "修改部门失败", err)
	}
	return nil
}

// Remove 删除部门：事务内校验并删除；存在子部门或成员时整体拒绝
func (s *DeptService) Remove(id uint64) error {
	// 校验与删除放在同一事务内，避免校验通过后被并发新增子部门
	return s.db.Transaction(func(tx *gorm.DB) error {
		var children int64
		if e := tx.Model(&model.Dept{}).Where("parent_id = ?", id).
			Count(&children).Error; e != nil {
			return utils.WrapAppError(response.CodeServerError, "查询子部门失败", e)
		}
		if children > 0 {
			return utils.NewAppError(response.CodeConflict, "存在子部门，无法删除")
		}

		var members int64
		if e := tx.Model(&model.User{}).Where("dept_id = ?", id).
			Count(&members).Error; e != nil {
			return utils.WrapAppError(response.CodeServerError, "查询部门成员失败", e)
		}
		if members > 0 {
			return utils.NewAppError(response.CodeConflict, "部门下仍有成员，无法删除")
		}

		var d model.Dept
		if e := tx.Where("id = ?", id).First(&d).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return utils.NewAppError(response.CodeNotFound, "部门不存在")
			}
			return utils.WrapAppError(response.CodeServerError, "查询部门失败", e)
		}
		return tx.Delete(&model.Dept{}, id).Error
	})
}
