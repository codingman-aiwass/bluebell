package sqls

type Paging struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func (p *Paging) Offset() int {
	offset := 0
	if p.Page > 0 {
		offset = (p.Page - 1) * p.Limit
	}
	return offset
}

func (p *Paging) TotalPage() int {
	if p.Total == 0 || p.Limit == 0 {
		return 0
	}
	totalPage := int(p.Total) / p.Limit
	if int(p.Total)%p.Limit > 0 {
		totalPage = totalPage + 1
	}
	return totalPage
}
