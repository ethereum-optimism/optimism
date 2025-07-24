package match

import "github.com/HashKeyChain/verse/op-devstack/stack"

var SecondL2EL = Second[stack.L2ELNodeID, stack.L2ELNode]()
var SecondL2CL = Second[stack.L2CLNodeID, stack.L2CLNode]()

var SecondSupervisor = Second[stack.SupervisorID, stack.Supervisor]()
