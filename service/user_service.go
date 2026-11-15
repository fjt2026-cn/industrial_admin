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

// ResetPassword 管理员重置口令，可由业务层决定是否强制改密
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
