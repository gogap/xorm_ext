package xorm_ext

import (
	"github.com/go-xorm/xorm"

	. "github.com/gogap/xorm_ext/errorcode"
)

const (
	REPO_DEFAULT_ENGINE               = "default"
	REPO_ERR_DEFAULT_ENGINE_NOT_FOUND = "`default` xorm engine not found"
)

type TXFunc func(repos []interface{}) (err error)

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

func (p *DBRepo) beginTransaction(engineName string) error {
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

func (p *DBRepo) commitNoTransaction(txFunc TXFunc, engineName string, repos ...interface{}) (err error) {
	if p.isTransaction {
		return ERR_DB_IS_A_TX.New()
	}

	p.txSession = p.SessionUsing(engineName)

	if p.txSession == nil {
		return ERR_DB_SESSION_IS_NIL.New()
	}

	defer p.txSession.Close()

	if txFunc == nil {
		return ERR_DB_TX_NOFUNC.New()
	}

	if e := txFunc(repos); e != nil {
		return e
	}

	return
}

func (p *DBRepo) commitTransaction(txFunc TXFunc, repos ...interface{}) (err error) {
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

	defer session.Close()

	if session.Begin() != nil {
		return ERR_DB_TX_CANNOT_BEGIN.New()
	}

	defer func() {
		if isNeedRollBack == true {
			session.Rollback()
		}
		return
	}()

	if e := txFunc(repos); e != nil {
		return e
	}

	isNeedRollBack = false
	if e := session.Commit(); e != nil {
		return ERR_DB_TX_COMMIT_ERROR.New()
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
