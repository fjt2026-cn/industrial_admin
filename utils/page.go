package utils

// PageQuery 列表接口统一分页参数
type PageQuery struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"pageSize" json:"pageSize"`
}

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 50
)

// Normalize 兜底并限制单页数量，避免一次拉取全表
func (p *PageQuery) Normalize() {
	if p.Page < 1 {
		p.Page = defaultPage
	}
	if p.PageSize < 1 {
		p.PageSize = defaultPageSize
	}
	if p.PageSize > maxPageSize {
		p.PageSize = maxPageSize
	}
}

func (p PageQuery) Offset() int {
	return (p.Page - 1) * p.PageSize
}
