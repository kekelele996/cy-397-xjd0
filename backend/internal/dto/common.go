package dto

// Pagination 通用分页入参。
type Pagination struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// Normalize 返回规范化后的分页参数。
func (p Pagination) Normalize() (page, pageSize int) {
	if p.Page <= 0 {
		page = 1
	} else {
		page = p.Page
	}
	if p.PageSize <= 0 {
		pageSize = 10
	} else if p.PageSize > 100 {
		pageSize = 100
	} else {
		pageSize = p.PageSize
	}
	return page, pageSize
}

// Offset 返回当前页偏移量。
func (p Pagination) Offset() int {
	page, pageSize := p.Normalize()
	return (page - 1) * pageSize
}
