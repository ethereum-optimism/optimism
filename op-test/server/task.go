package server

type TestTask struct {
	// TODO go test arguments

	// select sub-test cases that should run
	// TODO used for go test
	Filter []string

	// par test case, map of params that are added
	// params are inherited for sub-tests
	// TODO: json encode this, pass as special arg
	Params map[string]map[string]string
}

type TestTaskID struct {
	Name     string
	Instance string
}
