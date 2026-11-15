package service

import (
	"errors"

	"gorm.io/gorm"

	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

type PermissionService struct{ db *gorm.DB }

func NewPermissionService(database *gorm.DB) *PermissionService {
	return &PermissionService{db: database}
}

// PermissionQuery 权限点列表查询条件：支持关键字、类型与状态过滤
type PermissionQuery struct {
	utils.PageQuery
	Keyword  string `form:"keyword"`
	PermType *int8  `form:"permType"`
	Status   *int8  `form:"status"`
}

// List 分页查询权限点，按排序号升序返回，方便前端直接渲染树
func (s *PermissionService) List(q *PermissionQuery) ([]model.Permission, int64, error) {
	q.Normalize()

	tx := s.db.Model(&model.Permission{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("perm_name LIKE ? OR perm_code LIKE ?", kw, kw)
	}
	if q.PermType != nil {
		tx = tx.Where("perm_type = ?", *q.PermType)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询权限点总数失败", err)
	}

	var list []model.Permission
	if err := tx.Order("sort_no ASC, id ASC").
		Offset(q.Offset()).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询权限点失败", err)
	}
	return list, total, nil
}

// PermissionReq 新增与修改共用的入参：编码为业务唯一键，类型限定菜单/按钮/接口
type PermissionReq struct {
	ParentID   *uint64 `json:"parentId"`
	PermName   string  `json:"permName" binding:"required,max=64"`
	PermCode   string  `json:"permCode" binding:"required,max=100"`
	PermType   int8    `json:"permType" binding:"required,oneof=1 2 3"`
	RoutePath  string  `json:"routePath" binding:"max=200"`
	APIPath    string  `json:"apiPath" binding:"max=200"`
	HTTPMethod string  `json:"httpMethod" binding:"max=10"`
	Icon       string  `json:"icon" binding:"max=64"`
	SortNo     int     `json:"sortNo"`
	Status     *int8   `json:"status"`
	Remark     string  `json:"remark" binding:"max=255"`
}

// Create 新增权限点：编码全局唯一，父子关系只允许挂到同级或更高级的节点
func (s *PermissionService) Create(req *PermissionReq) (uint64, error) {
	var dup int64
	if err := s.db.Model(&model.Permission{}).
		Where("perm_code = ?", req.PermCode).Count(&dup).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "查询权限点失败", err)
	}
	if dup > 0 {
		return 0, utils.NewAppError(response.CodeConflict, "权限编码已存在")
	}

	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}
	p := model.Permission{
		ParentID: req.ParentID, PermName: req.PermName, PermCode: req.PermCode,
		PermType: req.PermType, RoutePath: req.RoutePath, APIPath: req.APIPath,
		HTTPMethod: req.HTTPMethod, Icon: req.Icon, SortNo: req.SortNo,
		Status: status, Remark: req.Remark,
	}
	if err := s.db.Create(&p).Error; err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "创建权限点失败", err)
	}
	return p.ID, nil
}

// Update 修改权限点：编码仍须唯一，允许调整状态与排序
func (s *PermissionService) Update(id uint64, req *PermissionReq) error {
	var p model.Permission
	if err := s.db.Where("id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "权限点不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询权限点失败", err)
	}

	var dup int64
	if err := s.db.Model(&model.Permission{}).
		Where("perm_code = ? AND id <> ?", req.PermCode, id).Count(&dup).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "查询权限点失败", err)
	}
	if dup > 0 {
		return utils.NewAppError(response.CodeConflict, "权限编码已存在")
	}

	fields := map[string]any{
		"parent_id": req.ParentID, "perm_name": req.PermName, "perm_code": req.PermCode,
		"perm_type": req.PermType, "route_path": req.RoutePath, "api_path": req.APIPath,
		"http_method": req.HTTPMethod, "icon": req.Icon, "sort_no": req.SortNo,
		"remark": req.Remark,
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if err := s.db.Model(&model.Permission{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "修改权限点失败", err)
	}
	return nil
}

// Delete 删除权限点：存在子节点时拒绝；先清授权关联再软删除，避免留下悬空授权
func (s *PermissionService) Delete(id uint64) error {
	var p model.Permission
	if err := s.db.Where("id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "权限点不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询权限点失败", err)
	}

	var children int64
	if err := s.db.Model(&model.Permission{}).Where("parent_id = ?", id).Count(&children).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "查询子权限失败", err)
	}
	if children > 0 {
		return utils.NewAppError(response.CodeConflict, "该权限点下仍有子节点，请先删除子节点")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("perm_id = ?", id).Delete(&model.RolePermission{}).Error; e != nil {
			return e
		}
		return tx.Where("id = ?", id).Delete(&model.Permission{}).Error
	})
}
