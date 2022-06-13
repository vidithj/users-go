package db

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/kit/sd"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/go-kivik/couchdb"
)

//RoundRobin will be dope round robinner based off of gokit round robin function
type RoundRobin struct {
	DBs        []DB
	c          uint64
	ch         chan sd.Event
	instancer  *consulsd.Instancer
	mtx        sync.RWMutex
	dbusername string
	dbpassword string
	dbName     string
	dbType     string
	queryLimit int
}

//NewRoundRobin returns a new round robin
func NewRoundRobin(instancer *consulsd.Instancer, dbusername, dbpwd, dbName string, dbType string, queryLimit int) *RoundRobin {
	rr := &RoundRobin{
		DBs:        make([]DB, 0),
		c:          0,
		ch:         make(chan sd.Event),
		instancer:  instancer,
		dbusername: dbusername,
		dbpassword: dbpwd,
		dbName:     dbName,
		dbType:     dbType,
		queryLimit: queryLimit,
	}
	go rr.receive()
	instancer.Register(rr.ch)
	return rr
}

func (rr *RoundRobin) receive() {
	for event := range rr.ch {
		rr.Update(event)
	}
}

//Update will update the endpoints based on the received information
func (rr *RoundRobin) Update(ev sd.Event) {
	rr.mtx.Lock()
	defer rr.mtx.Unlock()

	if ev.Err != nil {
		return
	}
	insts := ev.Instances
	sort.Strings(insts)
	dbs := make([]DB, 0, len(insts))
	rand.Seed(time.Now().Unix())
	for _, i := range insts {
		newDB, _ := InitDB(i, rr.dbType, rr.dbName, rr.dbName, "", rr.queryLimit)
		if err := newDB.userDB.Client().Authenticate(context.Background(), couchdb.BasicAuth(rr.dbusername, rr.dbpassword)); err != nil {
			continue
		}
		newDB.userDB.Client().CreateDB(context.Background(), rr.dbName, nil)
		dbs = append(dbs, newDB)

	}
	rr.DBs = dbs
}

//DB picks a db to send it too
func (rr *RoundRobin) DB() (DB, error) {
	rr.mtx.RLock()
	defer rr.mtx.RUnlock()
	if len(rr.DBs) <= 0 {
		return DB{}, errors.New("no DBs available")
	}
	old := atomic.AddUint64(&rr.c, 1) - 1
	idx := old % uint64(len(rr.DBs))
	return rr.DBs[idx], nil
}
