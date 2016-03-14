package xorm_ext

import (
	"reflect"

	"github.com/go-xorm/xorm"
	. "github.com/gogap/xorm_ext/errorcode"
)

type Inheriter interface {
	Inherit(originalRepo interface{}) (err error)
}

type Deriver interface {
	Derive() (v interface{}, err error)
}

type TransactionCommiter interface {
	Transaction(txFunc interface{}, repos ...interface{}) (err error)
	TransactionUsing(txFunc interface{}, name string, repos ...interface{}) (err error)
	NoTransaction(txFunc interface{}, repos ...interface{}) (err error)
	NoTransactionUsing(txFunc interface{}, name string, repos ...interface{}) (err error)
}

type DBTXCommiter struct {
}

func (p *DBTXCommiter) Transaction(txFunc interface{}, originRepos ...interface{}) (err error) {
	return p.TransactionUsing(txFunc, REPO_DEFAULT_ENGINE, originRepos...)
}
func (p *DBTXCommiter) TransactionUsing(txFunc interface{}, name string, originRepos ...interface{}) (err error) {
	reposLen := 0
	if originRepos != nil {
		reposLen = len(originRepos)
	}

	if reposLen < 1 {
		err = ERR_DB_ONE_REPO_AT_LEAST.New()
		return
	}

	if txFunc == nil {
		err = ERR_DB_TX_NOFUNC.New()
		return
	}

	newRepos := []interface{}{}
	newDBRepos := []*DBRepo{}

	for _, originRepo := range originRepos {

		dbRepo := getRepo(originRepo)

		if dbRepo == nil {
			err = ERR_STRUCT_NOT_COMBINE_WITH_DBREPO.New()
			return
		}

		var newDbRepo *DBRepo
		var newRepoI interface{}

		if newDbRepo, newRepoI, err = createNewRepo(originRepo); err != nil {
			return
		}

		newDbRepo.engines = dbRepo.engines
		newDbRepo.defaultEngine = dbRepo.defaultEngine
		newRepos = append(newRepos, newRepoI)
		newDBRepos = append(newDBRepos, newDbRepo)
	}

	if e := newDBRepos[0].beginTransaction(name); e != nil {
		return ERR_DB_TX_CANNOT_BEGIN.New().Append(e)
	}

	if reposLen > 1 {
		for i := 1; i < reposLen; i++ {
			newDBRepos[i].txSession = newDBRepos[0].txSession
			newDBRepos[i].isTransaction = newDBRepos[0].isTransaction
		}
	}

	return newDBRepos[0].commitTransaction(txFunc, newRepos...)
}

func (p *DBTXCommiter) NoTransaction(txFunc interface{}, originRepos ...interface{}) (err error) {
	return p.NoTransactionUsing(txFunc, REPO_DEFAULT_ENGINE, originRepos...)
}

func (p *DBTXCommiter) NoTransactionUsing(txFunc interface{}, name string, originRepos ...interface{}) (err error) {
	reposLen := 0
	if originRepos != nil {
		reposLen = len(originRepos)
	}

	if reposLen < 1 {
		err = ERR_DB_ONE_REPO_AT_LEAST.New()
		return
	}

	if txFunc == nil {
		err = ERR_DB_TX_NOFUNC.New()
		return
	}

	newRepos := []interface{}{}
	newDBRepos := []*DBRepo{}

	var sessions []*xorm.Session

	for _, originRepo := range originRepos {

		dbRepo := getRepo(originRepo)

		if dbRepo == nil {
			err = ERR_STRUCT_NOT_COMBINE_WITH_DBREPO.New()
			return
		}

		var newDbRepo *DBRepo
		var newRepoI interface{}

		if newDbRepo, newRepoI, err = createNewRepo(originRepo); err != nil {
			return
		}

		newDbRepo.engines = dbRepo.engines
		newDbRepo.defaultEngine = dbRepo.defaultEngine
		newRepos = append(newRepos, newRepoI)

		if e := newDbRepo.beginNoTransaction(name); e != nil {
			return e
		}

		newDBRepos = append(newDBRepos, newDbRepo)

		if newDbRepo.txSession != nil {
			sessions = append(sessions, newDbRepo.txSession)
		}
	}

	defer func() {
		for _, session := range sessions {
			session.Close()
		}
	}()

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

func createNewRepo(originRepo interface{}) (newDbRepo *DBRepo, newRepoI interface{}, err error) {

	iRepo := reflect.Indirect(reflect.ValueOf(originRepo))

	vRepo := reflect.ValueOf(iRepo.Interface())
	newRepoV := reflect.New(vRepo.Type())
	if !newRepoV.IsValid() {
		err = ERR_CAN_NOT_REFLACT_NEW_REPO.New()
		return
	}

	if deriver, ok := originRepo.(Deriver); ok {
		if repo, e := deriver.Derive(); e != nil {
			err = e
			return
		} else {
			newDbRepo = getRepo(repo)
			newRepoI = repo
		}
	} else {
		vRepo := reflect.ValueOf(iRepo.Interface())
		newRepoV := reflect.New(vRepo.Type())
		if !newRepoV.IsValid() {
			err = ERR_CAN_NOT_REFLACT_NEW_REPO.New()
			return
		}

		newRepoI = newRepoV.Interface()
		newDbRepo = getRepo(newRepoI)

		if inheriter, ok := newRepoI.(Inheriter); ok {
			if err = inheriter.Inherit(originRepo); err != nil {
				return
			}
		}

		if newDbRepo == nil {
			err = ERR_STRUCT_NOT_COMBINE_WITH_DBREPO.New()
			return
		}
	}
	return
}
