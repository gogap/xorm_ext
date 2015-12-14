package xorm_ext

import (
	"fmt"
	"github.com/go-xorm/xorm"
	"github.com/gogap/errors"
	"reflect"

	. "github.com/gogap/xorm_ext/errorcode"
)

var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()
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

func (p *DBRepo) commitNoTransaction(txFunc interface{}, engineName string, sessions []*xorm.Session, repos ...interface{}) (err error) {
	if p.isTransaction {
		err = ERR_DB_IS_A_TX.New()
		return
	}

	if p.txSession == nil {
		err = ERR_DB_SESSION_IS_NIL.New()
		return
	}

	defer func() {
		for _, session := range sessions {
			session.Close()
		}
	}()

	if err = callFunc(txFunc, repos); err != nil {
		return
	}

	return
}

func (p *DBRepo) commitTransaction(txFunc interface{}, repos ...interface{}) (err error) {
	if !p.isTransaction {
		err = ERR_DB_NOT_A_TX.New()
		return
	}

	if p.txSession == nil {
		err = ERR_DB_SESSION_IS_NIL.New()
		return
	}

	defer p.txSession.Close()

	if txFunc == nil {
		err = ERR_DB_TX_NOFUNC.New()
		return
	}

	isNeedRollBack := true

	if e := p.txSession.Begin(); e != nil {
		err = ERR_DB_TX_CANNOT_BEGIN.New().Append(e)
		return
	}

	defer func() {
		if isNeedRollBack == true {
			p.txSession.Rollback()
		}
		return
	}()

	if err = callFunc(txFunc, repos); err != nil {
		return
	}

	isNeedRollBack = false
	if err = p.txSession.Commit(); err != nil {
		err = ERR_DB_TX_COMMIT_ERROR.New()
		return
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

func callFunc(txfn interface{}, repos []interface{}) (err error) {
	switch fn := txfn.(type) {
	case TXFunc:
		{
			if err = fn(repos); err != nil {
				return
			}
		}
	default:
		err = call(txfn, repos...)
	}

	return
}

func call(fn interface{}, args ...interface{}) error {
	v := reflect.ValueOf(fn)
	if !v.IsValid() {
		return fmt.Errorf("call of nil")
	}
	typ := v.Type()
	if typ.Kind() != reflect.Func {
		return fmt.Errorf("non-function of type %s", typ)
	}
	if !goodFunc(typ) {
		return fmt.Errorf("function called with %d args; should be 1 or 2", typ.NumOut())
	}
	numIn := typ.NumIn()
	var dddType reflect.Type
	if typ.IsVariadic() {
		if len(args) < numIn-1 {
			return fmt.Errorf("wrong number of args: got %d want at least %d", len(args), numIn-1)
		}
		dddType = typ.In(numIn - 1).Elem()
	} else {
		if len(args) != numIn {
			return fmt.Errorf("wrong number of args: got %d want %d", len(args), numIn)
		}
	}
	argv := make([]reflect.Value, len(args))
	for i, arg := range args {
		value := reflect.ValueOf(arg)
		// Compute the expected type. Clumsy because of variadics.
		var argType reflect.Type
		if !typ.IsVariadic() || i < numIn-1 {
			argType = typ.In(i)
		} else {
			argType = dddType
		}

		var err error
		if argv[i], err = prepareArg(value, argType); err != nil {
			return fmt.Errorf("arg %d: %s", i, err)
		}
	}

	result := v.Call(argv)

	if !result[0].IsNil() {
		return result[0].Interface().(error)
	}

	return nil
}

func goodFunc(typ reflect.Type) bool {
	if typ.NumOut() == 1 && typ.Out(0) == errorType {
		return true
	}

	return false
}

func prepareArg(value reflect.Value, argType reflect.Type) (reflect.Value, error) {
	if !value.IsValid() {
		if !canBeNil(argType) {
			return reflect.Value{}, fmt.Errorf("value is nil; should be of type %s", argType)
		}
		value = reflect.Zero(argType)
	}
	if !value.Type().AssignableTo(argType) {
		return reflect.Value{}, fmt.Errorf("value has type %s; should be %s", value.Type(), argType)
	}
	return value, nil
}

func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}
