package service

type PowerCtl interface {
	Shutdown() error
	Restart() error
	Logout() error
}

type PowerCtlImp struct {
}
