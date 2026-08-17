package interfaces

type ModuleId int

type Module interface {
	Start()
	Wait()
	Stop()
}
