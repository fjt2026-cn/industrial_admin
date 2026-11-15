package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"industrial_admin/config"
	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

// 登录策略常量，正式环境建议改从 sys_config 读取
const (
	maxLoginFail = 5
	lockMinutes  = 10
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthService(database *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: database, cfg: cfg}
}

type UserInfo struct {
	ID       uint64   `json:"id"`
	Username string   `json:"username"`
	RealName string   `json:"realName"`
	DeptID   *uint64  `json:"deptId"`
	DeptName string   `json:"deptName"`
	Roles    []string `json:"roles"`
}

type LoginResult struct {
	Token     string            `json:"token"`
	ExpiresAt int64             `json:"expiresAt"`
	User      UserInfo          `json:"user"`
	Perms     []string          `json:"perms"`
	Menus     []*utils.TreeNode `json:"menus"`
}

// recordLogin 登录留痕：成功与失败都落一条操作日志，供审计与锁定排查
func (s *AuthService) recordLogin(userID uint64, username string, result int8, msg, ip string) {
	entry := model.OperationLog{
		Username: username,
		Module:   "登录鉴权",
		Action:   "登录",
		Method:   "POST",
		URL:      "/api/v1/auth/login",
		IP:       ip,
		Result:   result,
		ErrorMsg: msg,
	}
	if userID != 0 {
		uid := userID
		entry.UserID = &uid
	}
	// 日志失败不影响登录主流程
	s.db.Create(&entry)
}

// Login 校验账号口令、处理锁定计数、签发令牌并返回菜单与权限
func (s *AuthService) Login(username, password, ip string) (*LoginResult, error) {
	var u model.User
	err := s.db.Where("username = ?", username).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordLogin(0, username, 0, "账号不存在", ip)
			return nil, utils.NewAppError(response.CodeUnauthorized, "账号或口令错误")
		}
		return nil, utils.WrapAppError(response.CodeServerError, "登录失败", err)
	}

	if u.LockUntil != nil && u.LockUntil.After(time.Now()) {
		s.recordLogin(u.ID, u.Username, 0, "账号已锁定", ip)
		return nil, utils.NewAppError(response.CodeUnauthorized,
			fmt.Sprintf("账号已锁定，请于 %s 后重试", u.LockUntil.Format("15:04:05")))
	}

	if !utils.VerifyPassword(u.PasswordHash, password) {
		fail := u.LoginFailCount + 1
		updates := map[string]any{"login_fail_count": fail}
		if fail >= maxLoginFail {
			updates["lock_until"] = time.Now().Add(lockMinutes * time.Minute)
			updates["login_fail_count"] = 0
		}
		s.db.Model(&model.User{}).Where("id = ?", u.ID).Updates(updates)
		s.recordLogin(u.ID, u.Username, 0, "账号或口令错误", ip)
		return nil, utils.NewAppError(response.CodeUnauthorized, "账号或口令错误")
	}

	if u.Status != 1 {
		s.recordLogin(u.ID, u.Username, 0, "账号已停用", ip)
		return nil, utils.NewAppError(response.CodeForbidden, "账号已停用，请联系管理员")
	}

	now := time.Now()
	s.db.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"login_fail_count": 0,
		"lock_until":       nil,
		"last_login_at":    now,
		"last_login_ip":    ip,
	})

	token, exp, err := utils.GenerateToken(s.cfg, u.ID, u.Username)
	if err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "签发令牌失败", err)
	}

	info, perms, menus, err := s.Profile(u.ID)
	if err != nil {
		return nil, err
	}

	s.recordLogin(u.ID, u.Username, 1, "", ip)

	return &LoginResult{Token: token, ExpiresAt: exp, User: *info, Perms: perms, Menus: menus}, nil
}

// Profile 返回用户信息、权限编码集合与菜单树（多角色取并集）
func (s *AuthService) Profile(userID uint64) (*UserInfo, []string, []*utils.TreeNode, error) {
	var u model.User
	if err := s.db.Preload("Dept").Where("id = ?", userID).First(&u).Error; err != nil {
		return nil, nil, nil, utils.WrapAppError(response.CodeNotFound, "用户不存在", err)
	}

	info := &UserInfo{ID: u.ID, Username: u.Username, RealName: u.RealName, DeptID: u.DeptID}
	if u.Dept != nil {
		info.DeptName = u.Dept.DeptName
	}

	var roleNames []string
	if err := s.db.Table("role r").
		Joins("JOIN user_role ur ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.status = 1 AND r.deleted_at IS NULL", userID).
		Pluck("r.role_name", &roleNames).Error; err != nil {
		return nil, nil, nil, utils.WrapAppError(response.CodeServerError, "查询角色失败", err)
	}
	info.Roles = roleNames

	perms := make([]string, 0)
	if err := s.db.Table("permission p").
		Joins("JOIN role_permission rp ON rp.perm_id = p.id").
		Joins("JOIN user_role ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ? AND p.status = 1 AND p.deleted_at IS NULL", userID).
		Distinct().Pluck("p.perm_code", &perms).Error; err != nil {
		return nil, nil, nil, utils.WrapAppError(response.CodeServerError, "查询权限失败", err)
	}

	var menus []model.Permission
	if err := s.db.Table("permission p").
		Joins("JOIN role_permission rp ON rp.perm_id = p.id").
		Joins("JOIN user_role ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ? AND p.status = 1 AND p.perm_type = 1 AND p.deleted_at IS NULL", userID).
		Distinct().Order("p.sort_no").Find(&menus).Error; err != nil {
		return nil, nil, nil, utils.WrapAppError(response.CodeServerError, "查询菜单失败", err)
	}

	nodes := make([]*utils.TreeNode, 0, len(menus))
	for _, p := range menus {
		var parent uint64
		if p.ParentID != nil {
			parent = *p.ParentID
		}
		nodes = append(nodes, &utils.TreeNode{
			ID: p.ID, ParentID: parent, Name: p.PermName,
			Code: p.PermCode, Type: p.PermType, SortNo: p.SortNo,
		})
	}

	return info, perms, utils.BuildTree(nodes), nil
}
