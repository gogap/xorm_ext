package errorcode

import (
	"github.com/gogap/errors"
)

var (
	ERR_DB_SESSION_IS_NIL     = errors.T(11001, "db session is nil")
	ERR_DB_TX_ALREADY_BEGINED = errors.T(11002, "transaction already begin")
	ERR_DB_NOT_A_TX           = errors.T(11003, "non-tx could not be commit")
	ERR_DB_IS_A_TX            = errors.T(11004, "is-tx could not be commit")
	ERR_DB_TX_COMMIT_ERROR    = errors.T(11005, "commit error")
	ERR_DB_TX_NOFUNC          = errors.T(11006, "not a function")
	ERR_DB_TX_CANNOT_BEGIN    = errors.T(11007, "could not begin an transaction")

	ERR_NOT_COMBINE_WITH_DBREPO = errors.T(11008, "your db repository struct should combine with DBRepo")
	ERR_REFLACT_NEW_REPO        = errors.T(11009, "create new repo error")
	ERR_DB_IS_NIL               = errors.T(11010, "xorm Db is nil")
)
