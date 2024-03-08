package server

type ResourceHandle interface {
	HealthCheck() error
	Close() error
}

type Resource struct {
	Handle ResourceHandle
	Params map[string]string
}

type ResourceID string

type ResourceManager struct {
	resources map[ResourceID]*Resource
}

func (rm *ResourceManager) Close() error {
	// TODO
	return nil
}
