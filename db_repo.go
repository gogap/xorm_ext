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
	Logic       int = 0
	BeforeLogic int = 1
	AfterLogic  int = 2
	OnError     int = 3
	AfterCommit int = 4
)

const (
	REPO_DEFAULT_ENGINE               = "default"
	REPO_ERR_DEFAULT_ENGINE_NOT_FOUND = "`default` xorm engine not found"
)

type logicFuncs struct {
	BeforeLogic interface{}
	AfterLogic  interface{}
	OnError     interface{}
	Logic       interface{}
	AfterCommit interface{}
}

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

	funcs := getFuncs(txFunc)

	if funcs.BeforeLogic != nil {
		if _, err = callFunc(funcs.BeforeLogic, funcs, repos); err != nil {
			return
		}
	}

	var values []interface{}
	if funcs.Logic != nil {
		if values, err = callFunc(funcs.Logic, funcs, repos); err != nil {
			return
		}
	}

	if funcs.AfterLogic != nil {
		if values, err = callFunc(funcs.AfterLogic, funcs, repos); err != nil {
			return
		}
	}

	if funcs.AfterCommit != nil {
		if _, err = callFunc(funcs.AfterCommit, funcs, values); err != nil {
			return
		}
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

	funcs := getFuncs(txFunc)

	if funcs.BeforeLogic != nil {
		if _, err = callFunc(funcs.BeforeLogic, funcs, repos); err != nil {
			return
		}
	}

	var values []interface{}

	if funcs.Logic != nil {
		if values, err = callFunc(funcs.Logic, funcs, repos); err != nil {
			return
		}
	}

	if funcs.AfterLogic != nil {
		if values, err = callFunc(funcs.AfterLogic, funcs, repos); err != nil {
			return
		}
	}

	isNeedRollBack = false
	if err = p.txSession.Commit(); err != nil {
		err = ERR_DB_TX_COMMIT_ERROR.New()
		return
	}

	if funcs.AfterCommit != nil {
		if _, err = callFunc(funcs.AfterCommit, funcs, values); err != nil {
			return
		}
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

func getFuncs(fn interface{}) (funcs logicFuncs) {
	switch fn := fn.(type) {
	case TXFunc, func(repos []interface{}) (err error):
		{
			funcs.Logic = fn
		}
	case map[int]interface{}:
		{
			if hookBeforefn, exist := fn[BeforeLogic]; exist { //hook before
				funcs.BeforeLogic = hookBeforefn
			}

			if logicfn, exist := fn[Logic]; exist {
				funcs.Logic = logicfn
			}

			if hookAfterfn, exist := fn[AfterLogic]; exist { //hook after logic func
				funcs.AfterLogic = hookAfterfn
			}

			if errfn, exist := fn[OnError]; exist { //error callback
				funcs.OnError = errfn
			}

			if afterCommitfn, exist := fn[AfterCommit]; exist { //correct callback
				funcs.AfterCommit = afterCommitfn
			}
		}
	default:
		funcs.Logic = fn
	}

	return
}

func callFunc(fn interface{}, funcs logicFuncs, args []interface{}) (values []interface{}, err error) {
	if fn == nil {
		return
	}

	switch logicFunc := fn.(type) {
	case TXFunc:
		{
			err = logicFunc(args)
		}
	case func(repos []interface{}) (err error):
		{
			err = logicFunc(args)
		}
	default:
		if values, err = call(fn, args...); err != nil {
			if funcs.OnError != nil {
				call(funcs.OnError, err)
			}
		}

	}

	return
}

func call(fn interface{}, args ...interface{}) ([]interface{}, error) {
	v := reflect.ValueOf(fn)
	if !v.IsValid() {
		return nil, fmt.Errorf("call of nil")
	}
	typ := v.Type()
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("non-function of type %s", typ)
	}
	if !goodFunc(typ) {
		return nil, fmt.Errorf("the last return value should be an error type")
	}
	numIn := typ.NumIn()
	var dddType reflect.Type
	if typ.IsVariadic() {
		if len(args) < numIn-1 {
			return nil, fmt.Errorf("wrong number of args: got %d want at least %d, type: %v", len(args), numIn-1, typ)
		}
		dddType = typ.In(numIn - 1).Elem()
	} else {
		if len(args) != numIn {
			return nil, fmt.Errorf("wrong number of args: got %d want %d, type: %v", len(args), numIn, typ)
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
			return nil, fmt.Errorf("arg %d: %s", i, err)
		}
	}

	result := v.Call(argv)
	resultLen := len(result)

	var resultValues []interface{}

	for _, v := range result {
		resultValues = append(resultValues, v.Interface())
	}

	if resultLen == 1 {
		if resultValues[0] != nil {
			return nil, resultValues[0].(error)
		}
	} else if resultLen > 1 {
		if resultValues[resultLen-1] != nil {
			return resultValues[0 : resultLen-1], resultValues[resultLen-1].(error)
		} else {
			return resultValues[0 : resultLen-1], nil
		}
	}

	return nil, nil
}

func goodFunc(typ reflect.Type) bool {
	if typ.NumOut() > 0 && typ.Out(typ.NumOut()-1) == errorType {
		return true
	} else if typ.NumOut() == 0 {
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
