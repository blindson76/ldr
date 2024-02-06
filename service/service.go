package service

type ServiceInterface interface {
	Init() error
	Start() error
	Stop() error
}
