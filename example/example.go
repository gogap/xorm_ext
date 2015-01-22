package main

import (
	"fmt"

	"reflect"

	"github.com/go-xorm/xorm"

	"github.com/gogap/xorm_ext"
)

type User struct {
	UserName string
	Password string
}

type UserRepo interface {
	GetUser() string
}

type DBUserRepo struct {
	xorm_ext.DBRepo
}

func NewUserRepo(ormEngines map[string]*xorm.Engine) *DBUserRepo {
	repo := new(DBUserRepo)
	repo.SetEngines(ormEngines)
	return repo
}

func (p *DBUserRepo) GetUser() string {
	//p.Session().Query(sqlStr, ...)
	return "unknown"
}

func main() {

	engines := map[string]*xorm.Engine{xorm_ext.REPO_DEFAULT_ENGINE: new(xorm.Engine)}

	userRepo := NewUserRepo(engines)

	dbTXCommitter := new(xorm_ext.DBTXCommiter)

	logicFunc := func(repo interface{}) (txResult xorm_ext.TXResult, err error) {
		fmt.Println("enter logic")

		fmt.Println(reflect.TypeOf(repo))

		//this userRepo is a new instance of DBUserRepo
		if userRepo, ok := repo.(*DBUserRepo); ok {
			userName := userRepo.GetUser()
			fmt.Println(userName)
		}
		return
	}

	err := dbTXCommitter.CommitTX(userRepo, logicFunc)
	//Or
	//err := dbTXCommitter.CommitTXUsing("xormEngineName", userRepo, logicFunc)
	if err != nil {
		fmt.Println(err)
	}
}
