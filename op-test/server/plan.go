package server

type PlannedSubTest struct {
	Name     string
	Params   map[string]string
	SubTests []*PlannedSubTest
}

type PlannedTest struct {
	PlannedSubTest
	Package    string
	SourcePath string
}

type Plan struct {
	// TODO
	Tests []PlannedTest
}
