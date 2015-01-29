package main

import (
	"fmt"

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

	logicFunc := func(repos []interface{}) (err error) {
		fmt.Println("enter logic")

		var uRepo UserRepo
		for _, repo := range repos {
			switch repo := repo.(type) {
			case UserRepo:
				{
					//this userRepo is a new instance of DBUserRepo
					fmt.Println("get new user repo")
					uRepo = repo
				}
			}
		}

		userName := uRepo.GetUser()
		fmt.Println(userName)

		return
	}

	err := dbTXCommitter.Transaction(logicFunc, userRepo)
	//Or
	//err := dbTXCommitter.TransactionUsing("xormEngineName", userRepo, logicFunc)
	if err != nil {
		fmt.Println(err)
	}
}
