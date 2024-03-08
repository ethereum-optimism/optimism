package server

// load test plan

// split tests in conflicting groups
// conflict = param is different value

// N workers
//    ideally allocate resources to each worker

// worker consumes a group of tests

// (later optimization) if workers are idle or resources are available, split a group

// for each test in group:

// run the test
//      construct "go test" command
// 		run "go test" in sub-process, merge test output
//           can we check that all expected tests execute, and no unexpected tests do?
// resolve dependencies that a component is requested with
// auto-generate params to tag resources with their dependencies
// create components if necessary (param conflicts)

// if no test remains with compatible params to a resource, then shut down the resource

// log test results

// exit with 1 if any test failed
