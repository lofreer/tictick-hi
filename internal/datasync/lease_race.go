package datasync

import (
	"errors"

	"github.com/lofreer/tictick-hi/internal/data"
)

func isDataSyncLeaseRace(err error) bool {
	return errors.Is(err, data.ErrNotFound) || errors.Is(err, data.ErrInvalidState)
}
