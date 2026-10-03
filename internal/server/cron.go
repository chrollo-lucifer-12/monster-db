package server

import (
	"time"

	core "github.com/redis-server/internal/store"
)

func serverCronHandler(loop *EventLoop, id int64, clientData interface{}) int {
	now := time.Now()

	if now.Sub(lastRDB) > rdbFrequency {
		core.TriggerRDB()
		lastRDB = now
	}

	core.DeleteExpiredKey()

	return 1000
}
