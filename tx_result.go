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
	InsertItems []interface{}
	UpdateItems []UpdateItem
	DeleteItems []DeleteItem
}

func (p *TXResult) AppendUpdateItem(item UpdateItem) *TXResult {
	p.UpdateItems = append(p.UpdateItems, item)
	return p
}

func (p *TXResult) AppendInsertItem(item interface{}) *TXResult {
	p.InsertItems = append(p.InsertItems, item)
	return p
}

func (p *TXResult) AppendDeleteItem(item DeleteItem) *TXResult {
	p.DeleteItems = append(p.DeleteItems, item)
	return p
}
