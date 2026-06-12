package server

import "time"

const REPL_STATE_CONNECT = (1 << 2)
const REPL_STATE_NOT_CONNECT = (1 << 1)

var (
	eStatus int32 = EngineStatus_WAITING

	ReplicaState uint8 = REPL_STATE_NOT_CONNECT
	MasterHost   string
	MasterPort   int

	clientsPendingWrite = make(map[int]*Client)

	waitingKeys = make(map[string][]*Client)
	readyKeys   = make(map[string]struct{})

	subscribers = make(map[string][]*Client)

	watchedKeys = make(map[string][]*Client)

	con_clients = 0

	lastRDB      time.Time     = time.Now()
	rdbFrequency time.Duration = 900 * time.Second
)
