package barbarian

type CircuitBreaker interface {
	Name() string
	Execute(req func() (interface{}, error)) (interface{}, error)
}
