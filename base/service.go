package base

import (
	"context"
	"users/db"
	"users/model"

	"github.com/go-kit/kit/log"
)

//Service ...
type Service interface {
	Check(ctx context.Context) (bool, error)
	GetUser(ctx context.Context, username string) (model.User, error)
	UpdateUserAccess(ctx context.Context, req model.UpdateAccessRequest) error
	DoorAuthenticate(ctx context.Context, req model.DoorAuthenticate) (bool, error)
}

type baseService struct {
	logger log.Logger
	dbs    *db.RoundRobin
}

//NewService ...
func NewService(l log.Logger, dbs *db.RoundRobin) Service {
	return baseService{
		logger: l,
		dbs:    dbs,
	}
}

//Check ...
func (s baseService) Check(ctx context.Context) (bool, error) {
	return true, nil
}
func (s baseService) GetUser(ctx context.Context, username string) (model.User, error) {
	db, err := s.dbs.DB()
	if err != nil {
		return model.User{}, err
	}
	return db.GetUser(username)
}

func (s baseService) UpdateUserAccess(ctx context.Context, req model.UpdateAccessRequest) error {
	db, err := s.dbs.DB()
	if err != nil {
		return err
	}
	userinfo, err := db.GetUser(req.Username)
	if err != nil {
		return err
	}
	return db.UpdateUserAccess(userinfo, req)
}

func (s baseService) DoorAuthenticate(ctx context.Context, req model.DoorAuthenticate) (bool, error) {
	db, err := s.dbs.DB()
	if err != nil {
		return false, err
	}
	userinfo, err := db.GetUser(req.Username)
	if err != nil {
		return false, err
	}
	if hasaccess, found := userinfo.DoorAccess[req.AccessDoor]; found {
		return hasaccess, nil
	}
	return false, nil
}
