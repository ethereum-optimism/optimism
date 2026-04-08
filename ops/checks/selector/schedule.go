package selector

// Schedule computes a parallel execution schedule for selected checks,
// respecting prerequisite ordering. Returns the estimated wall-clock time
// and the execution layers (each layer runs in parallel).
type Schedule struct {
	Layers       []Layer // execution layers in order
	WallClock    float64 // estimated wall-clock time in seconds
	TotalCPU     float64 // sum of all check durations
	Parallelism  int     // max checks per layer
}

// Layer is a set of checks that can run in parallel.
type Layer struct {
	Checks   []string
	Duration float64 // wall time = max duration in this layer
}

// ComputeSchedule builds a parallel execution schedule.
// Checks are grouped into layers: a check goes into the earliest layer
// where all its prerequisites have completed. Within a layer, checks
// run in parallel, so the layer's wall time is the max duration.
func ComputeSchedule(selections []Selection, maxParallelism int) Schedule {
	if maxParallelism <= 0 {
		maxParallelism = 8 // default
	}

	// Build lookup maps
	duration := make(map[string]float64)
	prereqs := make(map[string][]string)
	for _, sel := range selections {
		duration[sel.CheckID] = sel.RunCost
		prereqs[sel.CheckID] = sel.Prerequisites
	}

	// Compute the layer for each check using topological ordering.
	// A check's layer = max(layer of prerequisites) + 1.
	// Checks with no prerequisites go in layer 0.
	layerOf := make(map[string]int)
	selected := make(map[string]bool)
	for _, sel := range selections {
		selected[sel.CheckID] = true
	}

	var computeLayer func(id string) int
	computeLayer = func(id string) int {
		if l, ok := layerOf[id]; ok {
			return l
		}
		maxPrereqLayer := -1
		for _, pid := range prereqs[id] {
			if !selected[pid] {
				continue // prerequisite not in selection set
			}
			pl := computeLayer(pid)
			if pl > maxPrereqLayer {
				maxPrereqLayer = pl
			}
		}
		l := maxPrereqLayer + 1
		layerOf[id] = l
		return l
	}

	for _, sel := range selections {
		computeLayer(sel.CheckID)
	}

	// Group checks by layer
	maxLayer := 0
	for _, l := range layerOf {
		if l > maxLayer {
			maxLayer = l
		}
	}

	layerChecks := make([][]string, maxLayer+1)
	for id, l := range layerOf {
		layerChecks[l] = append(layerChecks[l], id)
	}

	// If a layer has more checks than maxParallelism, split into sub-layers
	var layers []Layer
	for _, checks := range layerChecks {
		for i := 0; i < len(checks); i += maxParallelism {
			end := i + maxParallelism
			if end > len(checks) {
				end = len(checks)
			}
			batch := checks[i:end]

			// Layer wall time = max duration in the batch
			var maxDur float64
			for _, id := range batch {
				if duration[id] > maxDur {
					maxDur = duration[id]
				}
			}

			layers = append(layers, Layer{
				Checks:   batch,
				Duration: maxDur,
			})
		}
	}

	// Compute totals
	var wallClock, totalCPU float64
	for _, layer := range layers {
		wallClock += layer.Duration
		for _, id := range layer.Checks {
			totalCPU += duration[id]
		}
	}

	return Schedule{
		Layers:      layers,
		WallClock:   wallClock,
		TotalCPU:    totalCPU,
		Parallelism: maxParallelism,
	}
}
