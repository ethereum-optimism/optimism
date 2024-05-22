package docgen

func copyDocRouter(dr DocRouter) DocRouter {
	var cloneRouter func(dr DocRouter) DocRouter
	var cloneRoutes func(drt DocRoutes) DocRoutes

	cloneRoutes = func(drts DocRoutes) DocRoutes {
		rts := DocRoutes{}

		for pat, drt := range drts {
			rt := DocRoute{Pattern: drt.Pattern}
			if len(drt.Handlers) > 0 {
				rt.Handlers = DocHandlers{}
				for meth, dh := range drt.Handlers {
					rt.Handlers[meth] = dh
				}
			}
			if drt.Router != nil {
				rr := cloneRouter(*drt.Router)
				rt.Router = &rr
			}
			rts[pat] = rt
		}

		return rts
	}

	cloneRouter = func(dr DocRouter) DocRouter {
		cr := DocRouter{}
		cr.Middlewares = make([]DocMiddleware, len(dr.Middlewares))
		copy(cr.Middlewares, dr.Middlewares)
		cr.Routes = cloneRoutes(dr.Routes)
		return cr
	}

	return cloneRouter(dr)
}
