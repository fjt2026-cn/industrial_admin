package service

import (
	"errors"
	"strconv"
	"sync"

	"gorm.io/gorm"

	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

// 参数缓存：启动后首次读取时加载，修改后刷新，避免高频查询配置表
var (
	cfgCacheMu sync.RWMutex
	cfgCache   = make(map[string]string)
	cfgLoaded  bool
)

type ConfigService struct{ db *gorm.DB }

func NewConfigService(database *gorm.DB) *ConfigService { return &ConfigService{db: database} }

// Reload 全量刷新缓存
func (s *ConfigService) Reload() error {
	var rows []model.SysConfig
	if err := s.db.Where("status = 1").Find(&rows).Error; err != nil {
		return utils.WrapAppError(response.CodeServerError, "加载参数失败", err)
	}

	cfgCacheMu.Lock()
	cfgCache = make(map[string]string, len(rows))
	for _, r := range rows {
		cfgCache[r.ConfigKey] = r.ConfigValue
	}
	cfgLoaded = true
	cfgCacheMu.Unlock()
	return nil
}

// Get 按键读取参数，未配置时返回默认值
func (s *ConfigService) Get(key, def string) string {
	cfgCacheMu.RLock()
	if cfgLoaded {
		v, ok := cfgCache[key]
		cfgCacheMu.RUnlock()
		if ok {
			return v
		}
		return def
	}
	cfgCacheMu.RUnlock()

	if err := s.Reload(); err != nil {
		return def
	}
	cfgCacheMu.RLock()
	defer cfgCacheMu.RUnlock()
	if v, ok := cfgCache[key]; ok {
		return v
	}
	return def
}

// List 按分类查询参数，分类为空则返回全部
func (s *ConfigService) List(category string) ([]model.SysConfig, error) {
	tx := s.db.Model(&model.SysConfig{})
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	var rows []model.SysConfig
	if err := tx.Order("category ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "查询参数失败", err)
	}
	return rows, nil
}

// Update 修改参数值：按 value_type 校验格式，成功后刷新缓存
func (s *ConfigService) Update(id uint64, value string) error {
	var row model.SysConfig
	if err := s.db.Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.NewAppError(response.CodeNotFound, "参数不存在")
		}
		return utils.WrapAppError(response.CodeServerError, "查询参数失败", err)
	}

	switch row.ValueType {
	case 2: // 整数
		if _, err := strconv.Atoi(value); err != nil {
			return utils.NewAppError(response.CodeParamInvalid, "参数值必须为整数")
		}
	case 3: // 布尔
		if value != "true" && value != "false" {
			return utils.NewAppError(response.CodeParamInvalid, "参数值必须为 true 或 false")
		}
	}

	res := s.db.Model(&model.SysConfig{}).Where("id = ?", id).
		Updates(map[string]any{"config_value": value})
	if res.Error != nil {
		return utils.WrapAppError(response.CodeServerError, "更新参数失败", res.Error)
	}

	return s.Reload()
}
