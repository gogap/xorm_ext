package errorcode

import (
	"github.com/gogap/errors"
)

var (
	ERR_DB_SESSION_IS_NIL     = errors.T(11006, "db session is nil")
	ERR_DB_TX_ALREADY_BEGINED = errors.T(11002, "transaction already begin")
	ERR_DB_NOT_A_TX           = errors.T(11005, "non-tx could not be commit")
	ERR_DB_TX_COMMIT_ERROR    = errors.T(11000, "commit error")
	ERR_DB_TX_NOFUNC          = errors.T(11001, "not a function")
	ERR_DB_TX_CANNOT_BEGIN    = errors.T(11004, "could not begin an transaction")
)
