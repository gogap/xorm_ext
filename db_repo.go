package xorm_ext

import (
	"github.com/go-xorm/xorm"
	"github.com/gogap/errors"

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

func (p *DBRepo) beginTransaction(engineName string) (err error) {
	if p.isTransaction == false {
		p.isTransaction = true
		p.txSession = p.SessionUsing(engineName)
		if p.txSession == nil {
			err = ERR_CREATE_ENGINE_FAILED.New(errors.Params{"engineName": engineName})
			return
		}
	} else {
		err = ERR_DB_TX_ALREADY_BEGINED.New()
		return
	}
	return nil
}

func (p *DBRepo) beginNoTransaction(engineName string) error {
	if p.isTransaction {
		return ERR_CAN_NOT_CONV_TO_NO_TX.New()
	}

	p.txSession = p.SessionUsing(engineName)
	if p.txSession == nil {
		return ERR_CREATE_ENGINE_FAILED.New(errors.Params{"engineName": engineName})
	}

	return nil
}

func (p *DBRepo) commitNoTransaction(txFunc TXFunc, engineName string, sessions []*xorm.Session, repos ...interface{}) (err error) {
	if p.isTransaction {
		return ERR_DB_IS_A_TX.New()
	}

	if p.txSession == nil {
		return ERR_DB_SESSION_IS_NIL.New()
	}

	defer func() {
		for _, session := range sessions {
			session.Close()
		}
	}()

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

	defer p.txSession.Close()

	if txFunc == nil {
		return ERR_DB_TX_NOFUNC.New()
	}

	isNeedRollBack := true

	if e := p.txSession.Begin(); e != nil {
		return ERR_DB_TX_CANNOT_BEGIN.New().Append(e)
	}

	defer func() {
		if isNeedRollBack == true {
			p.txSession.Rollback()
		}
		return
	}()

	if e := txFunc(repos); e != nil {
		return e
	}

	isNeedRollBack = false
	if e := p.txSession.Commit(); e != nil {
		return ERR_DB_TX_COMMIT_ERROR.New()
	}
	return
}

func (p *DBRepo) Session() *xorm.Session {
	return p.txSession
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
