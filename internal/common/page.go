package common

// PageResult 分页结果
type PageResult struct {
	PageIndex int64       `json:"page_index"` // 页码
	PageSize  int64       `json:"page_size"`  // 页大小
	Total     int64       `json:"total"`      // 数据总量
	Data      interface{} `json:"data"`       // 数据
}

// NewPageResult NewPageResult
func NewPageResult(pageIndex int64, pageSize int64, total int64, data interface{}) *PageResult {
	return &PageResult{
		PageIndex: pageIndex,
		PageSize:  pageSize,
		Total:     total,
		Data:      data,
	}
}
