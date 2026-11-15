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

// Remove 删除部门：存在子部门或成员时拒绝
func (s *DeptService) Remove(id uint64) error {
	var children int64
	if err := s.db.Model(&model.Dept{}).Where("parent_id = ?", id).Count(&children).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "查询子部门失败", err)
	}
	if children > 0 {
		return utils.NewAppError(response.CodeConflict, "存在子部门，无法删除")
	}

	var members int64
	if err := s.db.Model(&model.User{}).Where("dept_id = ?", id).Count(&members).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "查询部门成员失败", err)
	}
	if members > 0 {
		return utils.NewAppError(response.CodeConflict, "部门下仍有成员，无法删除")
	}

	res := s.db.Delete(&model.Dept{}, id)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "部门不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "删除部门失败", res.Error)
	}
	return nil
}
