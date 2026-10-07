package dto

// KnowledgeListQuery FAQ 列表查询参数。
type KnowledgeListQuery struct {
	Pagination
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
}

// SaveKnowledgeRequest 创建/更新 FAQ 请求。
type SaveKnowledgeRequest struct {
	Category string `json:"category" binding:"required,max=64"`
	Question string `json:"question" binding:"required,max=512"`
	Answer   string `json:"answer" binding:"required"`
}
