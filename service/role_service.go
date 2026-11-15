package service

import (
	"errors"

	"gorm.io/gorm"

	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

type RoleService struct{ db *gorm.DB }

func NewRoleService(database *gorm.DB) *RoleService { return &RoleService{db: database} }

// RoleQuery 角色列表查询条件：关键字匹配名称或编码，可按状态过滤
type RoleQuery struct {
	utils.PageQuery
	Keyword string `form:"keyword"`
	Status  *int8  `form:"status"`
}

// List 分页查询角色：返回当页数据与总数，供前端表格分页
func (s *RoleService) List(q *RoleQuery) ([]model.Role, int64, error) {
	q.Normalize()

	tx := s.db.Model(&model.Role{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("role_name LIKE ? OR role_code LIKE ?", kw, kw)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询角色总数失败", err)
	}

	var roles []model.Role
	if err := tx.Order("sort_no ASC, id ASC").
		Offset(q.Offset()).Limit(q.PageSize).Find(&roles).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}
	return roles, total, nil
}

// RoleReq 角色新增与修改共用入参：编码唯一，内置角色由系统维护
type RoleReq struct {
	RoleName  string `json:"roleName" binding:"required,max=50"`
	RoleCode  string `json:"roleCode" binding:"required,max=50"`
	DataScope int8   `json:"dataScope" binding:"oneof=0 1 2 3"`
	SortNo    int    `json:"sortNo"`
	Status    *int8  `json:"status"`
	Remark    string `json:"remark" binding:"max=255"`
}

// Create 新增角色，编码与名称均唯一
func (s *RoleService) Create(req *RoleReq) (uint64, error) {
	var dup int64
	if err := s.db.Model(&model.Role{}).
		Where("role_code = ? OR role_name = ?", req.RoleCode, req.RoleName).Count(&dup).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}
	if dup > 0 {
		return 0, utils.NewAppError(response.CodeConflict, "角色名称或编码已存在")
	}

	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}
	r := model.Role{
		RoleName: req.RoleName, RoleCode: req.RoleCode, DataScope: req.DataScope,
		SortNo: req.SortNo, Status: status, Remark: req.Remark,
	}
	if err := s.db.Create(&r).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "创建角色失败", err)
	}
	return r.ID, nil
}

// Update 修改角色信息：内置角色只允许调整排序与备注，不允许改名称、编码与状态
func (s *RoleService) Update(id uint64, req *RoleReq) error {
	var r model.Role
	if err := s.db.Where("id = ?", id).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "角色不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}

	if r.IsBuiltin == 1 {
		return utils.NewAppError(response.CodeForbidden, "内置角色的名称与状态不可修改")
	}

	var dup int64
	if err := s.db.Model(&model.Role{}).
		Where("(role_code = ? OR role_name = ?) AND id <> ?", req.RoleCode, req.RoleName, id).
		Count(&dup).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}
	if dup > 0 {
		return utils.NewAppError(response.CodeConflict, "角色名称或编码已存在")
	}

	fields := map[string]any{
		"role_name": req.RoleName, "role_code": req.RoleCode,
		"data_scope": req.DataScope, "sort_no": req.SortNo, "remark": req.Remark,
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if err := s.db.Model(&model.Role{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "修改角色失败", err)
	}
	return nil
}

// Delete 删除角色：内置角色禁止删除；事务内清理授权与用户关联，避免悬空引用
func (s *RoleService) Delete(id uint64) error {
	var r model.Role
	if err := s.db.Where("id = ?", id).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "角色不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}
	if r.IsBuiltin == 1 {
		return utils.NewAppError(response.CodeForbidden, "内置角色不可删除")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("role_id = ?", id).Delete(&model.RolePermission{}).Error; e != nil {
			return e
		}
		if e := tx.Where("role_id = ?", id).Delete(&model.UserRole{}).Error; e != nil {
			return e
		}
		return tx.Where("id = ?", id).Delete(&model.Role{}).Error
	})
}

// PermissionIDs 返回角色已授权的权限点标识
func (s *RoleService) PermissionIDs(roleID uint64) ([]uint64, error) {
	ids := make([]uint64, 0)
	if err := s.db.Model(&model.RolePermission{}).
		Where("role_id = ?", roleID).Pluck("perm_id", &ids).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "查询授权失败", err)
	}
	return ids, nil
}

// Grant 授权：事务内先清空旧关联再批量写入，避免半授权中间态
func (s *RoleService) Grant(roleID uint64, permIDs []uint64) error {
	var r model.Role
	if err := s.db.Where("id = ?", roleID).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "角色不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}
	if r.IsBuiltin == 1 {
		return utils.NewAppError(response.CodeForbidden, "内置角色的权限不可修改")
	}

	// 先校验提交的权限点是否都存在：早期版本直接写库，非法 id 会触发外键约束错误
	// 被统一兜底成 500，对调用方不可解释；这里提前拦下并返回 400，同时去掉重复 id。
	ids := make([]uint64, 0, len(permIDs))
	seen := make(map[uint64]struct{}, len(permIDs))
	for _, pid := range permIDs {
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		ids = append(ids, pid)
	}
	if len(ids) > 0 {
		var existCount int64
		if e := s.db.Model(&model.Permission{}).Where("id IN ?", ids).Count(&existCount).Error; e != nil {
			return utils.WrapAppError(response.CodeServerError, "查询权限点失败", e)
		}
		if int(existCount) != len(ids) {
			return utils.NewAppError(response.CodeParamInvalid, "存在无效的权限点标识")
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; e != nil {
			return e
		}
		for _, pid := range ids {
			if e := tx.Create(&model.RolePermission{RoleID: roleID, PermID: pid}).Error; e != nil {
				return e
			}
		}
		return nil
	})
}

// AllPermissions 返回全部启用权限点，供前端渲染授权树
func (s *RoleService) AllPermissions() ([]model.Permission, error) {
	var perms []model.Permission
	if err := s.db.Where("status = 1").Order("sort_no ASC, id ASC").Find(&perms).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "查询权限点失败", err)
	}
	return perms, nil
}
