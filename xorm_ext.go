package xorm_ext

import (
	"reflect"

	. "github.com/gogap/xorm_ext/errorcode"
)

type TransactionCommiter interface {
	Transaction(repo interface{}, txFunc TXFunc) (err error)
	TransactionUsing(name string, repo interface{}, txFunc TXFunc) (err error)
}

type DBTXCommiter struct {
}

func (p *DBTXCommiter) Transaction(originRepo interface{}, txFunc TXFunc) (err error) {
	return p.TransactionUsing(REPO_DEFAULT_ENGINE, originRepo, txFunc)
}
func (p *DBTXCommiter) TransactionUsing(name string, originRepo interface{}, txFunc TXFunc) (err error) {
	iRepo := reflect.Indirect(reflect.ValueOf(originRepo))

	dbRepo := getRepo(originRepo)

	if dbRepo == nil {
		err = ERR_NOT_COMBINE_WITH_DBREPO.New()
		return
	}

	vRepo := reflect.ValueOf(iRepo.Interface())
	newRepoV := reflect.New(vRepo.Type())
	if !newRepoV.IsValid() {
		err = ERR_REFLACT_NEW_REPO.New()
		return
	}

	newRepoI := newRepoV.Interface()
	newDbRepo := getRepo(newRepoI)

	if newDbRepo == nil {
		err = ERR_NOT_COMBINE_WITH_DBREPO.New()
		return
	}
	newDbRepo.engines = dbRepo.engines
	newDbRepo.defaultEngine = dbRepo.defaultEngine

	if e := newDbRepo.BeginTransaction(name); e != nil {
		return ERR_DB_TX_CANNOT_BEGIN.New()
	}
	return newDbRepo.CommitTransaction(newRepoI, txFunc)
}

func setRepo(iface interface{}, value reflect.Value) {

}

func deepSetFields(iface interface{}, vType reflect.Type, value reflect.Value, fields []reflect.Value) interface{} {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)

	if ifv.Kind() == reflect.Ptr {
		ifv = ifv.Elem()
		ift = ifv.Type()
	}

	if ift == vType {
		//		reflect.Value.Set(value)
		return iface
	}

	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		switch v.Kind() {
		case reflect.Struct:
			deepIns := deepSetFields(v.Interface(), vType, value, fields)
			if deepIns != nil {
				return deepIns
			}
		}
	}
	return nil
}

func getRepo(v interface{}) *DBRepo {
	values := []reflect.Value{}
	deepRepo := deepFields(v, reflect.TypeOf(new(DBRepo)), values)
	if deepRepo, ok := deepRepo.(*DBRepo); ok {
		return deepRepo
	}
	return nil
}

func deepFields(iface interface{}, vType reflect.Type, fields []reflect.Value) interface{} {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)

	if ift == vType {

		return iface
	}

	if ifv.Kind() == reflect.Ptr {
		ifv = ifv.Elem()
		ift = ifv.Type()
	}

	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		switch v.Kind() {
		case reflect.Struct:
			var deepIns interface{}
			if v.CanAddr() {
				deepIns = deepFields(v.Addr().Interface(), vType, fields)
			} else {
				deepIns = deepFields(v.Interface(), vType, fields)
			}

			if deepIns != nil {
				return deepIns
			}
		}
	}
	return nil
}
