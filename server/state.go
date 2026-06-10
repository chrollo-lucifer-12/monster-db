package server

import "time"

var (
	eStatus int32 = EngineStatus_WAITING

	clientsPendingWrite = make(map[int]*Client)

	waitingKeys = make(map[string][]*Client)
	readyKeys   = make(map[string]struct{})

	subscribers     = make(map[string][]*Client)
	subscriberCount = make(map[string]int)

	con_clients = 0

	lastRDB      time.Time     = time.Now()
	rdbFrequency time.Duration = 900 * time.Second
)
