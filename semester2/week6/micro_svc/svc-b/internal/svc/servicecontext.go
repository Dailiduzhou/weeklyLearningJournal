package svc

import "github.com/Dailiduzhou/weeklyLearningJournal/semester2/week6/micro_svc/svc-b/internal/config"

type ServiceContext struct {
	Config config.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
