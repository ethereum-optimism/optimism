package engine_controller

type EngineController interface {
}

type simpleEngineController struct {
}

func NewEngineController() EngineController {
	return &simpleEngineController{}
}
