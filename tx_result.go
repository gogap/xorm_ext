package xorm_ext

type UpdateItem struct {
	OrmObject interface{}
	Filters   map[string]interface{}
	Params    map[string]interface{}
}

type DeleteItem struct {
	OrmObject interface{}
	Filters   map[string]interface{}
}

type TXResult struct {
	Items []interface{}
}

func (p *TXResult) Append(items ...interface{}) *TXResult {
	p.Items = append(p.Items, items...)
	return p
}
