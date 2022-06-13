package base

import (
	"context"
	"fmt"
	"users/db"
	"users/model"

	"github.com/go-kit/kit/log"
)

//Service ...
type Service interface {
	Check(ctx context.Context) (bool, error)
	GetUser(ctx context.Context, username string) (model.User, error)
	UpdateUserAccess(ctx context.Context, req model.UpdateAccessRequest) (string, error)
	DoorAuthenticate(ctx context.Context, req model.DoorAuthenticate) (string, error)
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

func (s baseService) UpdateUserAccess(ctx context.Context, req model.UpdateAccessRequest) (string, error) {
	db, err := s.dbs.DB()
	if err != nil {
		return "", err
	}
	userinfo, err := db.GetUser(req.Username)
	if err != nil {
		return "", err
	}
	if err := db.UpdateUserAccess(userinfo, req); err != nil {
		return fmt.Sprintf("%s", "Update Access Failed"), err
	}
	return fmt.Sprintf("%s", "Update Access Success"), nil
}

func (s baseService) DoorAuthenticate(ctx context.Context, req model.DoorAuthenticate) (string, error) {
	db, err := s.dbs.DB()
	if err != nil {
		return "", err
	}
	userinfo, err := db.GetUser(req.Username)
	if err != nil {
		return "", err
	}
	if hasaccess, found := userinfo.DoorAccess[req.AccessDoor]; found {
		if hasaccess {
			return fmt.Sprintf("%s", "User is authorization"), nil
		}
		return fmt.Sprintf("%s", "User does not have authorization"), nil
	}
	return "", nil
}
