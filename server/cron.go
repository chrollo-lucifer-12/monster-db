package server

import (
	"time"

	"github.com/redis-server/core"
)

func ServerCronHandler(loop *EventLoop, id int64, clientData interface{}) int {
	now := time.Now()

	if now.Sub(lastRDB) > rdbFrequency {
		core.TriggerRDB()
		lastRDB = now
	}

	core.DeleteExpiredKey()

	return 1000
}
