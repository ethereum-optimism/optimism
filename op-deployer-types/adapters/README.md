# op-deployer-types adapters

Adapters translate legacy op-deployer data shapes into ABI-shaped script input
objects for the selected contracts ref.

They intentionally live as static files next to the generated types instead of
as Go code in the fixed `op-deployer` runtime. Core `op-deployer` owns only the
generic mapping interpreter and validates each mapping against the artifact ABI.
If a contracts artifact changes a script input, CI should fail until the static
mapping for that contracts ref is updated.
