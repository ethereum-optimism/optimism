package selector

// Schedule computes a parallel execution schedule for execution items,
// respecting prerequisite ordering.
type Schedule struct {
	Layers      []Layer
	WallClock   float64 // estimated wall-clock time in seconds
	TotalCPU    float64 // sum of all item durations
	Parallelism int
}

// Layer is a set of items that can run in parallel.
type Layer struct {
	ItemIDs  []string
	Duration float64 // wall time = max duration in this layer
}

// ComputeSchedule builds a parallel execution schedule from execution items.
// Items are grouped into layers based on prerequisites. Within a layer,
// items run in parallel, so the layer's wall time is the max duration.
func ComputeSchedule(items []ExecutionItem, maxParallelism int) Schedule {
	if maxParallelism <= 0 {
		maxParallelism = 8
	}

	duration := make(map[string]float64)
	prereqs := make(map[string][]string)
	for _, item := range items {
		duration[item.ID] = item.RunCost
		prereqs[item.ID] = item.Prerequisites
	}

	layerOf := make(map[string]int)
	itemSet := make(map[string]bool)
	for _, item := range items {
		itemSet[item.ID] = true
	}

	var computeLayer func(id string) int
	computeLayer = func(id string) int {
		if l, ok := layerOf[id]; ok {
			return l
		}
		maxPrereqLayer := -1
		for _, pid := range prereqs[id] {
			if !itemSet[pid] {
				continue
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

	for _, item := range items {
		computeLayer(item.ID)
	}

	maxLayer := 0
	for _, l := range layerOf {
		if l > maxLayer {
			maxLayer = l
		}
	}

	layerItems := make([][]string, maxLayer+1)
	for id, l := range layerOf {
		layerItems[l] = append(layerItems[l], id)
	}

	var layers []Layer
	for _, ids := range layerItems {
		for i := 0; i < len(ids); i += maxParallelism {
			end := i + maxParallelism
			if end > len(ids) {
				end = len(ids)
			}
			batch := ids[i:end]

			var maxDur float64
			for _, id := range batch {
				if duration[id] > maxDur {
					maxDur = duration[id]
				}
			}

			layers = append(layers, Layer{
				ItemIDs:  batch,
				Duration: maxDur,
			})
		}
	}

	var wallClock, totalCPU float64
	for _, layer := range layers {
		wallClock += layer.Duration
		for _, id := range layer.ItemIDs {
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
