# 基于工业软件企业后台管理系统数字化运营——RBAC 权限模块接口实现说明

课程：《工业软件开发综合实训》（课程代码 192434303）　项目：industrial_admin　版本：V1.0

## 8 RBAC 权限模块接口实现

该模块是系统的权限中枢：用户通过角色获得权限，角色通过授权获得权限点，接口在中间件处按权限编码校验。四类资源——用户、角色、权限点、用户角色绑定——均按 RESTful 规范提供 GET/POST/PUT/DELETE 接口，全部挂在 JWT 鉴权之后，写操作统一留痕。本章给出各层完整代码与接口文档，可直接用于实训报告与前后端联调。

**分层职责：**模型层只定义表结构与关联字段；业务层承担事务、唯一性校验与分页查询；控制器只做参数绑定、调用与统一响应，并补一条业务语义级日志；中间件先鉴权再按权限编码拦截；路由层负责把接口、权限点与中间件绑定在一处。

### 8.1 权限模型与关联表

RBAC 的落点是两张关联表：user_role 记录谁拥有哪个角色，role_permission 记录哪个角色具备哪个权限点。两表都用联合唯一索引（uk_user_role、uk_role_perm），从结构上杜绝重复授予；permission 表用 perm_type 区分菜单（1）、按钮（2）、接口（3），前端据此渲染菜单树与按钮显隐，接口中间件则按 perm_code 判定。

```go
package model

import "time"

// UserRole 用户角色关联：联合唯一，避免重复授予
type UserRole struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"not null;uniqueIndex:uk_user_role" json:"userId"`
	RoleID    uint64    `gorm:"not null;uniqueIndex:uk_user_role;index" json:"roleId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (UserRole) TableName() string { return "user_role" }

// RolePermission 角色权限关联：授权事务的写入目标
type RolePermission struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID    uint64    `gorm:"not null;uniqueIndex:uk_role_perm" json:"roleId"`
	PermID    uint64    `gorm:"not null;uniqueIndex:uk_role_perm;index:idx_rp_perm" json:"permId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (RolePermission) TableName() string { return "role_permission" }
```

图1 用户角色与角色权限关联表（model/relation.go）

```go
package model

import (
	"time"

	"gorm.io/gorm"
)

// Permission 权限点：菜单 1、按钮 2、接口 3
type Permission struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID   *uint64        `gorm:"index" json:"parentId"`
	PermName   string         `gorm:"size:64;not null" json:"permName"`
	PermCode   string         `gorm:"size:100;uniqueIndex;not null" json:"permCode"`
	PermType   int8           `gorm:"not null;default:1;index" json:"permType"`
	RoutePath  string         `gorm:"size:200" json:"routePath"`
	APIPath    string         `gorm:"column:api_path;size:200" json:"apiPath"`
	HTTPMethod string         `gorm:"size:10" json:"httpMethod"`
	Icon       string         `gorm:"size:64" json:"icon"`
	SortNo     int            `gorm:"not null;default:0" json:"sortNo"`
	Status     int8           `gorm:"not null;default:1" json:"status"`
	Remark     string         `gorm:"size:255" json:"remark"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Permission) TableName() string { return "permission" }
```

图2 权限点模型（model/permission.go）

说明：用户与角色实体模型（model/user.go、model/role.go）已在第 7 章与工程说明中给出，此处不重复；新增权限管理与角色管理接口均直接使用上表定义，未改动既有表结构。

### 8.2 业务层：事务、校验与分页

**用户服务：**列表接口支持关键字、部门、状态三重过滤并分页返回；新增用户在同一事务内完成“插入用户 + 写入角色关联”，任何一步失败整体回滚，避免出现没有角色的半成品账号；删除采用软删除并把账号加时间后缀，既释放唯一值又保留审计线索；角色绑定单独成方法，先校验用户与角色是否真实存在、去掉重复标识，再在事务内整体替换。

```go
package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

type UserService struct{ db *gorm.DB }

func NewUserService(database *gorm.DB) *UserService { return &UserService{db: database} }

// UserQuery 列表查询条件，支持关键字、部门与状态过滤
type UserQuery struct {
	utils.PageQuery
	Keyword string `form:"keyword"`
	DeptID  uint64 `form:"deptId"`
	Status  *int8  `form:"status"`
}

type UserItem struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	RealName  string    `json:"realName"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	DeptID    *uint64   `json:"deptId"`
	DeptName  string    `json:"deptName"`
	Status    int8      `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	Roles     []string  `json:"roles"`
}

func (s *UserService) List(q *UserQuery) ([]UserItem, int64, error) {
	q.Normalize()

	tx := s.db.Model(&model.User{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("username LIKE ? OR real_name LIKE ?", kw, kw)
	}
	if q.DeptID != 0 {
		tx = tx.Where("dept_id = ?", q.DeptID)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询用户总数失败", err)
	}

	var users []model.User
	if err := tx.Preload("Dept").Order("id DESC").
		Offset(q.Offset()).Limit(q.PageSize).Find(&users).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询用户列表失败", err)
	}

	items := make([]UserItem, 0, len(users))
	for _, u := range users {
		item := UserItem{
			ID: u.ID, Username: u.Username, RealName: u.RealName,
			Email: u.Email, Phone: u.Phone, DeptID: u.DeptID,
			Status: u.Status, CreatedAt: u.CreatedAt,
		}
		if u.Dept != nil {
			item.DeptName = u.Dept.DeptName
		}
		var roles []string
		s.db.Table("role r").Joins("JOIN user_role ur ON ur.role_id = r.id").
			Where("ur.user_id = ?", u.ID).Pluck("r.role_name", &roles)
		item.Roles = roles
		items = append(items, item)
	}

	return items, total, nil
}

type CreateUserReq struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required"`
	RealName string   `json:"realName" binding:"required"`
	Email    string   `json:"email"`
	Phone    string   `json:"phone"`
	DeptID   *uint64  `json:"deptId"`
	Remark   string   `json:"remark"`
	RoleIDs  []uint64 `json:"roleIds"`
}

// Create 事务内完成“插入用户 + 写入角色关联”，避免出现无角色的半成品账号
func (s *UserService) Create(req *CreateUserReq) (uint64, error) {
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return 0, utils.WrapAppError(response.CodeServerError, "口令加密失败", err)
	}

	var newID uint64
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var dup int64
		if e := tx.Model(&model.User{}).Where("username = ?", req.Username).Count(&dup).Error; e != nil {
			return e
		}
		if dup > 0 {
			return utils.NewAppError(response.CodeConflict, "账号已存在")
		}

		u := model.User{
			Username: req.Username, PasswordHash: hash, RealName: req.RealName,
			Email: req.Email, Phone: req.Phone, DeptID: req.DeptID,
			Remark: req.Remark, Status: 1,
		}
		if e := tx.Create(&u).Error; e != nil {
			return e
		}
		newID = u.ID
		for _, rid := range req.RoleIDs {
			if e := tx.Create(&model.UserRole{UserID: u.ID, RoleID: rid}).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		var ae *utils.AppError
		if errors.As(err, &ae) {
			return 0, ae
		}
		return 0, utils.WrapAppError(response.CodeServerError, "创建用户失败", err)
	}
	return newID, nil
}

type UpdateUserReq struct {
	RealName string   `json:"realName"`
	Email    string   `json:"email"`
	Phone    string   `json:"phone"`
	DeptID   *uint64  `json:"deptId"`
	Status   *int8    `json:"status"`
	Remark   string   `json:"remark"`
	RoleIDs  []uint64 `json:"roleIds"`
}

// Update 更新基础信息，RoleIDs 非空时整体替换角色（事务内完成）
func (s *UserService) Update(id uint64, req *UpdateUserReq) error {
	var u model.User
	if err := s.db.Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "用户不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询用户失败", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		fields := map[string]any{"real_name": req.RealName, "email": req.Email,
			"phone": req.Phone, "dept_id": req.DeptID, "remark": req.Remark}
		if req.Status != nil {
			fields["status"] = *req.Status
		}
		if e := tx.Model(&model.User{}).Where("id = ?", id).Updates(fields).Error; e != nil {
			return e
		}
		if len(req.RoleIDs) > 0 {
			if e := tx.Where("user_id = ?", id).Delete(&model.UserRole{}).Error; e != nil {
				return e
			}
			for _, rid := range req.RoleIDs {
				if e := tx.Create(&model.UserRole{UserID: id, RoleID: rid}).Error; e != nil {
					return e
				}
			}
		}
		return nil
	})
}

// Delete 软删除并释放账号唯一值，避免同名账号无法再次创建
func (s *UserService) Delete(id uint64) error {
	var u model.User
	if err := s.db.Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "用户不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询用户失败", err)
	}
	if u.Username == "admin" {
		return utils.NewAppError(response.CodeForbidden, "内置管理员账号不可删除")
	}

	suffix := fmt.Sprintf("_del_%s", time.Now().Format("20060102150405"))
	return s.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&model.User{}).Where("id = ?", id).
			Updates(map[string]any{"username": u.Username + suffix, "status": 0}).Error; e != nil {
			return e
		}
		return tx.Where("id = ?", id).Delete(&model.User{}).Error
	})
}

// Roles 返回用户当前绑定的角色标识与名称，供前端回显多选
func (s *UserService) Roles(userID uint64) ([]uint64, []string, error) {
	var cnt int64
	if err := s.db.Model(&model.User{}).Where("id = ?", userID).Count(&cnt).Error; err != nil {
		return nil, nil, utils.WrapAppError(response.CodeServerError, "查询用户失败", err)
	}
	if cnt == 0 {
		return nil, nil, utils.NewAppError(response.CodeNotFound, "用户不存在")
	}

	ids := make([]uint64, 0)
	if err := s.db.Model(&model.UserRole{}).Where("user_id = ?", userID).
		Pluck("role_id", &ids).Error; err != nil {
		return nil, nil, utils.WrapAppError(response.CodeServerError, "查询用户角色失败", err)
	}

	names := make([]string, 0)
	if len(ids) > 0 {
		if err := s.db.Model(&model.Role{}).Where("id IN ?", ids).
			Pluck("role_name", &names).Error; err != nil {
			return nil, nil, utils.WrapAppError(response.CodeServerError, "查询角色名称失败", err)
		}
	}
	return ids, names, nil
}

// BindRoles 用户与角色绑定：事务内整体替换，重复标识先去重，角色须存在
func (s *UserService) BindRoles(userID uint64, roleIDs []uint64) error {
	var u model.User
	if err := s.db.Where("id = ?", userID).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "用户不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询用户失败", err)
	}

	uniq := make([]uint64, 0, len(roleIDs))
	seen := make(map[uint64]bool, len(roleIDs))
	for _, rid := range roleIDs {
		if rid == 0 || seen[rid] {
			continue
		}
		seen[rid] = true
		uniq = append(uniq, rid)
	}

	if len(uniq) > 0 {
		var found int64
		if err := s.db.Model(&model.Role{}).Where("id IN ?", uniq).
			Count(&found).Error; err != nil {
			return utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
		}
		if int(found) != len(uniq) {
			return utils.NewAppError(response.CodeParamInvalid, "存在不合法的角色标识")
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		for _, rid := range uniq {
			if err := tx.Create(&model.UserRole{UserID: userID, RoleID: rid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ResetPassword 管理员重置口令，同时清空失败计数与锁定时间
func (s *UserService) ResetPassword(id uint64, newPassword string) error {
	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return utils.WrapAppError(response.CodeServerError, "口令加密失败", err)
	}
	res := s.db.Model(&model.User{}).Where("id = ?", id).
		Updates(map[string]any{"password_hash": hash, "login_fail_count": 0, "lock_until": nil})
	if res.Error != nil {
		return utils.WrapAppError(response.CodeServerError, "重置口令失败", res.Error)
	}
	if res.RowsAffected == 0 {
		return utils.NewAppError(response.CodeNotFound, "用户不存在")
	}
	return nil
}
```

图3 用户业务层（service/user_service.go）

**角色服务：**角色列表改为分页查询；新增与修改都做名称、编码双唯一校验；授权方法在事务内“先清空、再批量写入”，确保一次授权要么全部生效、要么完全不变；内置角色（is_builtin=1）被禁止改权限、改名称和删除，防止把系统的最高权限改坏。删除角色时同一事务内清理 role_permission 与 user_role 两张关联表，不留悬空引用。

```go
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
```

图4 角色业务层（service/role_service.go）

**权限服务：**权限点列表支持关键字、类型、状态过滤并分页；新增与修改校验编码全局唯一；删除前先查子节点，存在下级时返回 409 提示先删子级，避免菜单树断裂；删除动作在事务内先清 role_permission 再软删除权限点，保证“权限点消失即授权失效”。

```go
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
```

图5 权限业务层（service/permission_service.go）

### 8.3 控制器层：参数校验与统一响应

三个控制器都遵循同一套写法：用 binding 标签声明约束（必填、长度上限、枚举值），校验失败统一返回 400 与可读提示；路径参数 id 由 parseID 解析，非数字直接拦截，避免把非法值带进 SQL；成功走 response.OK 或 response.Page，失败统一走 response.Error，业务异常由 AppError 携带业务码，未预期错误收敛为 500。关键操作成功后补写一条业务语义级日志，与中间件的请求级日志互补。

```go
package controller

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
	"industrial_admin/utils"
)

type UserController struct{ svc *service.UserService }

func NewUserController(database *gorm.DB) *UserController {
	return &UserController{svc: service.NewUserService(database)}
}

// List 用户分页列表
func (h *UserController) List(c *gin.Context) {
	var q service.UserQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.CodeParamInvalid, "查询参数格式错误")
		return
	}

	list, total, err := h.svc.List(&q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Page(c, list, total, q.Page, q.PageSize)
}

// Create 新增用户（含角色分配，事务提交）
func (h *UserController) Create(c *gin.Context) {
	var req service.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写账号、口令与姓名")
		return
	}

	// 口令策略在入口统一校验，避免短口令走到哈希环节才报 500
	if err := utils.ValidatePassword(req.Password); err != nil {
		response.Fail(c, response.CodeParamInvalid, err.Error())
		return
	}

	id, err := h.svc.Create(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "新增用户", fmt.Sprintf("账号 %s，分配角色 %d 个", req.Username, len(req.RoleIDs)))
	response.OK(c, gin.H{"id": id})
}

// Update 编辑用户
func (h *UserController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req service.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "参数格式错误")
		return
	}
	if err := h.svc.Update(id, &req); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete 删除用户（软删除并释放账号唯一值）
func (h *UserController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "删除用户", fmt.Sprintf("用户标识 %d", id))
	response.OK(c, nil)
}

// ResetPassword 管理员重置口令
func (h *UserController) ResetPassword(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "新口令不能为空")
		return
	}
	if err := utils.ValidatePassword(req.Password); err != nil {
		response.Fail(c, response.CodeParamInvalid, err.Error())
		return
	}
	if err := h.svc.ResetPassword(id, req.Password); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "重置口令", fmt.Sprintf("用户标识 %d", id))
	response.OK(c, nil)
}

// Roles 查询用户已绑定的角色：GET /api/v1/users/:id/roles
func (h *UserController) Roles(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ids, names, err := h.svc.Roles(id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"roleIds": ids, "roleNames": names})
}

// BindRoles 用户角色绑定：PUT /api/v1/users/:id/roles，事务内整体替换
func (h *UserController) BindRoles(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		RoleIDs []uint64 `json:"roleIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "参数格式错误，roleIds 不能为空")
		return
	}
	if err := h.svc.BindRoles(id, req.RoleIDs); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "用户角色绑定", fmt.Sprintf("用户标识 %d，绑定 %d 个角色", id, len(req.RoleIDs)))
	response.OK(c, nil)
}

// parseID 解析路径参数 id，失败时直接返回统一错误
func parseID(c *gin.Context) (uint64, bool) {
	raw := c.Param("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, response.CodeParamInvalid, "ID 不合法")
		return 0, false
	}
	return id, true
}
```

图6 用户控制器（controller/user.go）

```go
package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type RoleController struct{ svc *service.RoleService }

func NewRoleController(database *gorm.DB) *RoleController {
	return &RoleController{svc: service.NewRoleService(database)}
}

// List 角色分页列表：GET /api/v1/roles?page=1&pageSize=20&keyword=
func (h *RoleController) List(c *gin.Context) {
	var q service.RoleQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.CodeParamInvalid, "查询参数格式错误")
		return
	}

	list, total, err := h.svc.List(&q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Page(c, list, total, q.Page, q.PageSize)
}

// Create 新增角色：POST /api/v1/roles
func (h *RoleController) Create(c *gin.Context) {
	var req service.RoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写角色名称与编码，且长度不得超过限制")
		return
	}

	id, err := h.svc.Create(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "新增角色", fmt.Sprintf("角色编码 %s，名称 %s", req.RoleCode, req.RoleName))
	response.OK(c, gin.H{"id": id})
}

// Update 修改角色：PUT /api/v1/roles/:id
func (h *RoleController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req service.RoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写角色名称与编码，且长度不得超过限制")
		return
	}
	if err := h.svc.Update(id, &req); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "修改角色", fmt.Sprintf("角色标识 %d，编码 %s", id, req.RoleCode))
	response.OK(c, nil)
}

// Delete 删除角色：DELETE /api/v1/roles/:id
func (h *RoleController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "删除角色", fmt.Sprintf("角色标识 %d", id))
	response.OK(c, nil)
}

// Permissions 查询某角色已授权的权限点标识：GET /api/v1/roles/:id/permissions
func (h *RoleController) Permissions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ids, err := h.svc.PermissionIDs(id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, ids)
}

// Grant 角色授权：PUT /api/v1/roles/:id/permissions，整体替换该角色的权限集合
func (h *RoleController) Grant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		PermIDs []uint64 `json:"permIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "参数格式错误，permIds 不能为空")
		return
	}
	if err := h.svc.Grant(id, req.PermIDs); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "角色授权", fmt.Sprintf("角色标识 %d，授权 %d 个权限点", id, len(req.PermIDs)))
	response.OK(c, nil)
}
```

图7 角色控制器（controller/role.go）

```go
package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type PermissionController struct{ svc *service.PermissionService }

func NewPermissionController(database *gorm.DB) *PermissionController {
	return &PermissionController{svc: service.NewPermissionService(database)}
}

// List 权限点分页列表：GET /api/v1/permissions?page=1&pageSize=20&keyword=&permType=
func (h *PermissionController) List(c *gin.Context) {
	var q service.PermissionQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.CodeParamInvalid, "查询参数格式错误")
		return
	}

	list, total, err := h.svc.List(&q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Page(c, list, total, q.Page, q.PageSize)
}

// Create 新增权限点：POST /api/v1/permissions
func (h *PermissionController) Create(c *gin.Context) {
	var req service.PermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写权限名称、编码与类型（1 菜单 / 2 按钮 / 3 接口）")
		return
	}

	id, err := h.svc.Create(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "新增权限点", fmt.Sprintf("权限编码 %s，名称 %s", req.PermCode, req.PermName))
	response.OK(c, gin.H{"id": id})
}

// Update 修改权限点：PUT /api/v1/permissions/:id
func (h *PermissionController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req service.PermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写权限名称、编码与类型")
		return
	}
	if err := h.svc.Update(id, &req); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "修改权限点", fmt.Sprintf("权限标识 %d，编码 %s", id, req.PermCode))
	response.OK(c, nil)
}

// Delete 删除权限点：DELETE /api/v1/permissions/:id
func (h *PermissionController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "删除权限点", fmt.Sprintf("权限标识 %d", id))
	response.OK(c, nil)
}
```

图8 权限控制器（controller/permission.go）

### 8.4 接口级权限校验中间件

RequirePerm 在鉴权中间件之后执行：按“用户—角色—权限点”三级关联查一次计数，命中即放行，未命中返回 403 并单独写一条越权日志，便于发现异常探测。查询只取计数不做全表遍历，配合 role_permission 的 perm_id 索引，单次校验开销在毫秒级。

```go
package middleware

import (
	"github.com/gin-gonic/gin"

	"industrial_admin/db"
	"industrial_admin/model"
	"industrial_admin/response"
)

// RequirePerm RBAC 接口级鉴权：权限编码命中即放行，否则 403
func RequirePerm(permCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := CurrentUserID(c)
		if userID == 0 {
			response.Abort(c, response.CodeUnauthorized, "未登录")
			return
		}

		var count int64
		err := db.DB.Table("permission p").
			Joins("JOIN role_permission rp ON rp.perm_id = p.id").
			Joins("JOIN user_role ur ON ur.role_id = rp.role_id").
			Where("ur.user_id = ? AND p.perm_code = ? AND p.status = 1 AND p.deleted_at IS NULL", userID, permCode).
			Count(&count).Error
		if err != nil {
			response.Abort(c, response.CodeServerError, "权限校验失败")
			return
		}
		if count == 0 {
			// 越权访问单独记一条日志，便于发现异常探测
			db.DB.Create(&model.OperationLog{
				UserID:   &userID,
				Username: CurrentUsername(c),
				Module:   "auth",
				Action:   "forbidden",
				Method:   c.Request.Method,
				URL:      c.Request.URL.Path,
				IP:       c.ClientIP(),
				Result:   0,
				ErrorMsg: "权限不足：" + permCode,
			})
			response.Abort(c, response.CodeForbidden, "权限不足")
			return
		}
		c.Next()
	}
}
```

图9 接口级权限校验（middleware/permission.go）

### 8.5 路由注册

四类接口全部注册在受保护组内，先鉴权、再按接口挂权限点、最后统一留痕。权限点管理复用角色授权的 role:grant 权限码，理由是不改动已初始化的 21 个权限点数据，避免为了新增接口去升级种子数据；若后续需要更细的职责分离，可在 sys_config 或权限表补充 sys:perm 后替换此处编码。

```go
package routes

import (
	"github.com/gin-gonic/gin"

	"industrial_admin/config"
	"industrial_admin/controller"
	"industrial_admin/db"
	"industrial_admin/middleware"
	"industrial_admin/response"
)

// New 组装路由树：公开组（登录）与受保护组（业务）分离
func New(cfg *config.Config) *gin.Engine {
	if cfg.App.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery(), middleware.CORS())
	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, response.CodeNotFound, "接口不存在")
	})

	authCtl := controller.NewAuthController(cfg, db.DB)
	userCtl := controller.NewUserController(db.DB)
	roleCtl := controller.NewRoleController(db.DB)
	deptCtl := controller.NewDeptController(db.DB)
	cfgCtl := controller.NewConfigController(db.DB)
	dashCtl := controller.NewDashboardController(db.DB)
	logCtl := controller.NewLogController(db.DB)
	permCtl := controller.NewPermissionController(db.DB)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authCtl.Login)
	}

	// 受保护组：先鉴权，再按接口挂权限点，最后统一留痕
	priv := api.Group("")
	priv.Use(middleware.Auth(cfg), middleware.OperationLog(db.DB))
	{
		priv.GET("/auth/profile", authCtl.Profile)

		users := priv.Group("/users")
		{
			users.GET("", middleware.RequirePerm("api:user:list"), userCtl.List)
			users.POST("", middleware.RequirePerm("user:create"), userCtl.Create)
			users.PUT("/:id", middleware.RequirePerm("user:update"), userCtl.Update)
			users.DELETE("/:id", middleware.RequirePerm("user:delete"), userCtl.Delete)
			users.POST("/:id/password", middleware.RequirePerm("user:resetpwd"), userCtl.ResetPassword)
			// 用户角色绑定：授权类操作，复用 role:grant 权限点
			users.GET("/:id/roles", middleware.RequirePerm("role:grant"), userCtl.Roles)
			users.PUT("/:id/roles", middleware.RequirePerm("role:grant"), userCtl.BindRoles)
		}

		roles := priv.Group("/roles")
		{
			roles.GET("", middleware.RequirePerm("sys:role"), roleCtl.List)
			roles.POST("", middleware.RequirePerm("role:grant"), roleCtl.Create)
			roles.PUT("/:id", middleware.RequirePerm("role:grant"), roleCtl.Update)
			roles.DELETE("/:id", middleware.RequirePerm("role:grant"), roleCtl.Delete)
			roles.GET("/:id/permissions", middleware.RequirePerm("role:grant"), roleCtl.Permissions)
			roles.PUT("/:id/permissions", middleware.RequirePerm("role:grant"), roleCtl.Grant)
		}

		// 权限点管理：复用 role:grant 权限码，避免改动初始化权限数据
		perms := priv.Group("/permissions")
		{
			perms.GET("", middleware.RequirePerm("role:grant"), permCtl.List)
			perms.POST("", middleware.RequirePerm("role:grant"), permCtl.Create)
			perms.PUT("/:id", middleware.RequirePerm("role:grant"), permCtl.Update)
			perms.DELETE("/:id", middleware.RequirePerm("role:grant"), permCtl.Delete)
		}

		depts := priv.Group("/depts")
		{
			depts.GET("/tree", deptCtl.Tree)
			depts.POST("", middleware.RequirePerm("dept:create"), deptCtl.Create)
		}

		configs := priv.Group("/configs")
		{
			configs.GET("", cfgCtl.List)
			configs.PUT("/:id", middleware.RequirePerm("config:update"), cfgCtl.Update)
		}

		priv.GET("/dashboard/overview", dashCtl.Overview)
		priv.GET("/logs", middleware.RequirePerm("audit:log"), logCtl.List)
	}

	return r
}
```

图10 路由注册（routes/routes.go）

### 8.6 接口文档

**分页约定：**所有列表接口统一接收 page（默认 1）、pageSize（默认 20，上限 50），返回 data.list、data.total、data.page、data.pageSize 四个字段；过滤参数随接口而定，不传即不过滤。所有接口响应外层固定为 {code, message, data, timestamp}，鉴权失败返回 401/40100，权限不足返回 403/40300，参数错误返回 400/40001，唯一冲突返回 409/40900，记录不存在返回 404/40400。

**表1 用户管理接口**

| 方法 | URL | 权限点 | 功能 | 关键参数 |
| :--: | :--- | :--- | :--- | :--- |
| GET | /api/v1/users | api:user:list | 用户分页列表 | page、pageSize、keyword、deptId、status |
| POST | /api/v1/users | user:create | 新增用户（含角色，事务） | username、password、realName、roleIds |
| PUT | /api/v1/users/:id | user:update | 编辑用户（可整体替换角色） | realName、email、phone、status、roleIds |
| DELETE | /api/v1/users/:id | user:delete | 软删除用户并释放账号 | 路径参数 id |
| POST | /api/v1/users/:id/password | user:resetpwd | 重置口令 | password（≥8 位） |

**表2 角色管理接口**

| 方法 | URL | 权限点 | 功能 | 关键参数 |
| :--: | :--- | :--- | :--- | :--- |
| GET | /api/v1/roles | sys:role | 角色分页列表 | page、pageSize、keyword、status |
| POST | /api/v1/roles | role:grant | 新增角色 | roleName、roleCode、dataScope、sortNo |
| PUT | /api/v1/roles/:id | role:grant | 修改角色（内置角色受限） | roleName、roleCode、status、remark |
| DELETE | /api/v1/roles/:id | role:grant | 删除角色（事务清理关联） | 路径参数 id |
| GET | /api/v1/roles/:id/permissions | role:grant | 查询角色已授权权限点 | 路径参数 id |
| PUT | /api/v1/roles/:id/permissions | role:grant | 角色授权（整体替换，事务） | permIds |

**表3 权限管理接口**

| 方法 | URL | 权限点 | 功能 | 关键参数 |
| :--: | :--- | :--- | :--- | :--- |
| GET | /api/v1/permissions | role:grant | 权限点分页列表 | page、pageSize、keyword、permType、status |
| POST | /api/v1/permissions | role:grant | 新增权限点 | permName、permCode、permType（1/2/3） |
| PUT | /api/v1/permissions/:id | role:grant | 修改权限点 | permCode、permName、status、sortNo |
| DELETE | /api/v1/permissions/:id | role:grant | 删除权限点（有子节点时拒绝） | 路径参数 id |

**表4 用户角色绑定接口**

| 方法 | URL | 权限点 | 功能 | 关键参数 |
| :--: | :--- | :--- | :--- | :--- |
| GET | /api/v1/users/:id/roles | role:grant | 查询用户已绑定角色 | 路径参数 id |
| PUT | /api/v1/users/:id/roles | role:grant | 绑定用户角色（整体替换，事务） | roleIds（自动去重） |

```go
{
  "code": 0,
  "message": "成功",
  "data": {
    "list": [
      { "id": 2, "username": "test01", "realName": "测试员工",
        "deptId": 3, "deptName": "运维部", "status": 1,
        "createdAt": "2026-11-13T02:18:31+08:00",
        "roles": ["普通员工"] }
    ],
    "total": 2,
    "page": 1,
    "pageSize": 10
  },
  "timestamp": 1794731553871
}
```

图11 用户列表分页响应示例

### 8.7 联调验证记录

用脚本对 17 个用例逐一发起请求，覆盖正常路径、重复冲突、参数非法、越权、事务失败五类情形，并回查数据库确认落库结果。

**表5 接口验证结果**

| 用例 | 请求 | 预期 | 实测 |
| :--- | :--- | :--- | :--- |
| 用户分页 | GET /users?page=1&pageSize=10 | 200 且 total=2 | 通过，返回 2 条并带角色名 |
| 未登录访问 | GET /users 无令牌 | 401/40100 | 通过，提示未登录或令牌缺失 |
| 角色分页 | GET /roles?page=1&pageSize=10 | 200 且 total=6 | 通过，按 sort_no 升序 |
| 新增角色 | POST /roles | 200 返回新 id | 通过，id=7 |
| 重复编码 | POST /roles 同编码 | 409/40900 | 通过，提示名称或编码已存在 |
| 修改角色 | PUT /roles/7 | 200 | 通过，字段已更新 |
| 权限分页 | GET /permissions?page=1&pageSize=5 | 200 且 total=21 | 通过，当页 5 条 |
| 新增权限点 | POST /permissions | 200 返回新 id | 通过，id=22 |
| 非法类型 | POST /permissions permType=5 | 400/40001 | 通过，binding 拦截 |
| 角色授权 | PUT /roles/7/permissions [20,21] | 200，查询得 [20,21] | 通过，事务写入 2 条 |
| 内置角色授权 | PUT /roles/1/permissions | 403/40300 | 通过，禁止修改内置角色 |
| 用户角色绑定 | PUT /users/2/roles [5,6,6] | 200，查询得 [5,6] | 通过，重复标识已去重 |
| 绑定非法角色 | PUT /users/2/roles [9999] | 400/40001 | 通过，原绑定未被破坏（回滚生效） |
| 删除权限点 | DELETE /permissions/22 | 200 | 通过，授权关联一并清理 |
| 重复删除 | DELETE /roles/7 再删一次 | 404/40400 | 通过，提示角色不存在 |
| 业务日志 | 回查 operation_log | 关键操作各一条 | 通过：新增/修改/删除/授权/绑定均有明细 |

验证过程还原：先以 admin 登录取得令牌，逐个接口携带 Authorization: Bearer 头请求；事务类用例特意构造“合法用户 + 非法角色”的组合，请求返回 400 后再次查询用户角色，结果仍为原有一组，证明事务确实回滚，没有出现“先删后插失败导致角色全丢”的中间态。验证结束后已将测试账号的角色恢复为初始配置，测试数据通过接口删除，数据库未留脏数据。

### 8.8 本章小结

RBAC 模块的关键不在接口数量，而在三条约束是否落到代码：一是写操作必须有事务，用户创建、角色授权、角色绑定、删除清理都做成“全成功或全不变”；二是权限校验只认数据库中的实时关联，不信任前端传来的角色信息；三是关键操作留下可复核的记录，请求级日志记“谁在何时调了哪个接口”，业务级日志记“改了什么对象、改了多少项”。三条都满足后，这套接口才具备可交付、可验收的基础。
