package xorm_ext

import (
	"github.com/go-xorm/xorm"

	. "github.com/gogap/xorm_ext/errorcode"
)

const (
	REPO_DEFAULT_ENGINE               = "default"
	REPO_ERR_DEFAULT_ENGINE_NOT_FOUND = "`default` xorm engine not found"
)

type TXFunc func(repo interface{}) (txResult TXResult, err error)

type DBRepo struct {
	isTransaction bool
	engines       map[string]*xorm.Engine
	defaultEngine *xorm.Engine
	txSession     *xorm.Session
}

func (p *DBRepo) SetEngines(ormEngines map[string]*xorm.Engine) {
	if defaultEngine, exist := ormEngines[REPO_DEFAULT_ENGINE]; exist {
		p.engines = ormEngines
		p.defaultEngine = defaultEngine
	} else {
		panic(REPO_ERR_DEFAULT_ENGINE_NOT_FOUND)
	}
}

func (p *DBRepo) Engines() map[string]*xorm.Engine {
	return p.engines
}

func (p *DBRepo) DefaultEngine() *xorm.Engine {
	return p.defaultEngine
}

func (p *DBRepo) IsTransaction() bool {
	return p.isTransaction
}

func (p *DBRepo) BeginTransaction(engineName string) error {
	if p.isTransaction == false {
		p.isTransaction = true
		p.txSession = p.SessionUsing(engineName)
		if p.txSession == nil {
			p.txSession = p.defaultEngine.NewSession()
		}
	} else {
		ERR_DB_TX_ALREADY_BEGINED.New()
	}
	return nil
}

func (p *DBRepo) CommitNoTransaction(engineName string, repo interface{}, txFunc TXFunc) (err error) {
	if p.isTransaction {
		return ERR_DB_IS_A_TX.New()
	}

	if p.txSession == nil {
		return ERR_DB_SESSION_IS_NIL.New()
	}

	if txFunc == nil {
		return ERR_DB_TX_NOFUNC.New()
	}

	p.txSession = p.SessionUsing(engineName)

	if ret, e := txFunc(repo); e != nil {
		return e
	} else {
		p.commitTxResult(ret)
	}

	return
}

func (p *DBRepo) CommitTransaction(repo interface{}, txFunc TXFunc) (err error) {
	if !p.isTransaction {
		return ERR_DB_NOT_A_TX.New()
	}

	if p.txSession == nil {
		return ERR_DB_SESSION_IS_NIL.New()
	}

	if txFunc == nil {
		return ERR_DB_TX_NOFUNC.New()
	}

	isNeedRollBack := true
	session := p.txSession

	if session.Begin() != nil {
		return ERR_DB_TX_CANNOT_BEGIN.New()
	}

	defer func() {
		if isNeedRollBack == true {
			session.Rollback()
		}
		return
	}()

	if ret, e := txFunc(repo); e != nil {
		return e
	} else {
		p.commitTxResult(ret)
	}

	isNeedRollBack = false
	if e := session.Commit(); e != nil {
		return ERR_DB_TX_COMMIT_ERROR.New()
	}
	return
}

func (p *DBRepo) commitTxResult(tx TXResult) (err error) {
	for _, resultItem := range tx.Items {
		switch itemType := resultItem.(type) {
		case UpdateItem:
			{
				p.UpdateItems(itemType)
			}
		case DeleteItem:
			{
				p.DeleteItems(itemType)
			}
		case interface{}:
			{
				p.InsertItems(itemType)
			}
		}
	}
	return
}

func (p *DBRepo) UpdateItems(items ...UpdateItem) (err error) {
	for _, item := range items {
		session := p.Session().Table(item.OrmObject)
		for fk, fv := range item.Filters {
			session = session.And(fk+"= ?", fv)
		}
		if _, err = session.Update(item.Params); err != nil {
			return
		}
	}
	return
}

func (p *DBRepo) InsertItems(items ...interface{}) (err error) {
	_, err = p.Session().InsertMulti(items)
	return
}

func (p *DBRepo) DeleteItems(items ...DeleteItem) (err error) {
	for _, item := range items {
		session := p.Session()
		for fk, fv := range item.Filters {
			session = session.And(fk+"= ?", fv)
		}
		if _, err = session.Delete(item.OrmObject); err != nil {
			return
		}
	}
	return
}

func (p *DBRepo) Session() *xorm.Session {
	if p.isTransaction &&
		p.txSession != nil {
		return p.txSession
	}
	return p.defaultEngine.NewSession()
}

func (p *DBRepo) NewSession() *xorm.Session {
	return p.defaultEngine.NewSession()
}

func (p *DBRepo) SessionUsing(engineName string) *xorm.Session {
	if engine, exist := p.engines[engineName]; exist {
		return engine.NewSession()
	}
	return nil
}
