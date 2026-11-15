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

func (s *RoleService) List() ([]model.Role, error) {
	var roles []model.Role
	if err := s.db.Order("sort_no ASC, id ASC").Find(&roles).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}
	return roles, nil
}

// Create 新增角色，编码与名称均唯一
func (s *RoleService) Create(name, code string, dataScope int8, remark string) (uint64, error) {
	var dup int64
	if err := s.db.Model(&model.Role{}).
		Where("role_code = ? OR role_name = ?", code, name).Count(&dup).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}
	if dup > 0 {
		return 0, utils.NewAppError(response.CodeConflict, "角色名称或编码已存在")
	}
	r := model.Role{RoleName: name, RoleCode: code, DataScope: dataScope, Status: 1, Remark: remark}
	if err := s.db.Create(&r).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "创建角色失败", err)
	}
	return r.ID, nil
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

	return s.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; e != nil {
			return e
		}
		for _, pid := range permIDs {
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
