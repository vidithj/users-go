package db

import (
	"context"
	"users/model"

	"github.com/go-kivik/kivik"
)

const (
	documentConflict = "Conflict: Document update conflict."
)

//DB ...
type DB struct {
	userDB       *kivik.DB
	RegisterName string
	tag          string
	queryLimit   int
}

//InitDB ...
func InitDB(dbLoc, dbType, dbName, dbRegisterName, tag string, queryLimit int) (DB, error) {
	client, err := kivik.New(dbType, dbLoc)
	if err != nil {
		return DB{}, err
	}
	db1 := client.DB(context.Background(), dbName)
	return DB{
		db1,
		dbRegisterName,
		tag,
		queryLimit,
	}, err
}

func (db DB) GetUser(username string) (model.User, error) {
	var doc model.User
	row := db.userDB.Get(context.Background(), username)
	if row == nil {
		return model.User{}, nil
	}
	err := row.ScanDoc(&doc)
	if err != nil {
		return model.User{}, err
	}
	return doc, nil
}

func (db DB) UpdateUserAccess(userinfo model.User, req model.UpdateAccessRequest) error {
	userinfo.DoorAccess = req.Doors
	_, err := db.userDB.Put(context.Background(), req.Username, userinfo)
	for err != nil {
		if err.Error() != documentConflict {
			return err
		}
		rev, err := db.ResolveConfict(req.Username)
		if err != nil {
			return err //can't fetch resolve
		}
		userinfo.Revision = rev
		_, err = db.userDB.Put(context.Background(), req.Username, userinfo)
	}
	return nil
}

func (db DB) ResolveConfict(username string) (string, error) {
	userinfo, err := db.GetUser(username)
	if err != nil {
		return "", err
	}
	return userinfo.Revision, nil
}
