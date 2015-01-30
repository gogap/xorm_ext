package xorm_ext

import (
	"reflect"

	. "github.com/gogap/xorm_ext/errorcode"
)

type TransactionCommiter interface {
	Transaction(txFunc TXFunc, repos ...interface{}) (err error)
	TransactionUsing(txFunc TXFunc, name string, repos ...interface{}) (err error)
	NoTransaction(txFunc TXFunc, repos ...interface{}) (err error)
	NoTransactionUsing(txFunc TXFunc, name string, repos ...interface{}) (err error)
}

type DBTXCommiter struct {
}

func (p *DBTXCommiter) Transaction(txFunc TXFunc, originRepos ...interface{}) (err error) {
	return p.TransactionUsing(txFunc, REPO_DEFAULT_ENGINE, originRepos...)
}
func (p *DBTXCommiter) TransactionUsing(txFunc TXFunc, name string, originRepos ...interface{}) (err error) {
	reposLen := 0
	if originRepos != nil {
		reposLen = len(originRepos)
	}

	if reposLen < 1 {
		err = ERR_DB_ONE_REPO_AT_LEAST.New()
		return
	}

	newRepos := []interface{}{}
	newDBRepos := []*DBRepo{}

	for _, originRepo := range originRepos {
		iRepo := reflect.Indirect(reflect.ValueOf(originRepo))

		dbRepo := getRepo(originRepo)

		if dbRepo == nil {
			err = ERR_STRUCT_NOT_COMBINE_WITH_DBREPO.New()
			return
		}

		vRepo := reflect.ValueOf(iRepo.Interface())
		newRepoV := reflect.New(vRepo.Type())
		if !newRepoV.IsValid() {
			err = ERR_CAN_NOT_REFLACT_NEW_REPO.New()
			return
		}

		newRepoI := newRepoV.Interface()
		newDbRepo := getRepo(newRepoI)

		if newDbRepo == nil {
			err = ERR_STRUCT_NOT_COMBINE_WITH_DBREPO.New()
			return
		}

		newDbRepo.engines = dbRepo.engines
		newDbRepo.defaultEngine = dbRepo.defaultEngine
		newRepos = append(newRepos, newRepoI)
		newDBRepos = append(newDBRepos, newDbRepo)
	}

	if e := newDBRepos[0].beginTransaction(name); e != nil {
		return ERR_DB_TX_CANNOT_BEGIN.New()
	}

	if reposLen > 1 {
		for i := 1; i < reposLen; i++ {
			newDBRepos[i].txSession = newDBRepos[0].txSession
			newDBRepos[i].isTransaction = newDBRepos[0].isTransaction
		}
	}

	return newDBRepos[0].commitTransaction(txFunc, newRepos...)
}

func (p *DBTXCommiter) NoTransaction(txFunc TXFunc, originRepos ...interface{}) (err error) {
	return p.NoTransactionUsing(txFunc, REPO_DEFAULT_ENGINE, originRepos...)
}
func (p *DBTXCommiter) NoTransactionUsing(txFunc TXFunc, name string, originRepos ...interface{}) (err error) {
	reposLen := 0
	if originRepos != nil {
		reposLen = len(originRepos)
	}

	if reposLen < 1 {
		err = ERR_DB_ONE_REPO_AT_LEAST.New()
		return
	}

	newRepos := []interface{}{}
	newDBRepos := []*DBRepo{}

	for _, originRepo := range originRepos {
		iRepo := reflect.Indirect(reflect.ValueOf(originRepo))

		dbRepo := getRepo(originRepo)

		if dbRepo == nil {
			err = ERR_STRUCT_NOT_COMBINE_WITH_DBREPO.New()
			return
		}

		vRepo := reflect.ValueOf(iRepo.Interface())
		newRepoV := reflect.New(vRepo.Type())
		if !newRepoV.IsValid() {
			err = ERR_CAN_NOT_REFLACT_NEW_REPO.New()
			return
		}

		newRepoI := newRepoV.Interface()
		newDbRepo := getRepo(newRepoI)

		if newDbRepo == nil {
			err = ERR_STRUCT_NOT_COMBINE_WITH_DBREPO.New()
			return
		}

		newDbRepo.engines = dbRepo.engines
		newDbRepo.defaultEngine = dbRepo.defaultEngine
		newRepos = append(newRepos, newRepoI)
		newDBRepos = append(newDBRepos, newDbRepo)
	}

	return newDBRepos[0].commitNoTransaction(txFunc, name, newRepos...)
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
