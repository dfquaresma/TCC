package sim

import (
	"time"

	"github.com/agoussia/godes"
)

// TODO(david): Document the fields of this struct.
type Results struct {
	Instances      []IInstance
	Cost           float64
	Efficiency     float64
	RequestCount   int64
	SimulationTime int64
}

// Run executes a simulation.
// TODO(david): document each parameters.
func Run(duration, idlenessDeadline time.Duration, ia InterArrival, entries [][]InputEntry, listener Listener, scheduler int, warmUp int) Results {
	before := time.Now()
	lb := newLoadBalancer(idlenessDeadline, entries, listener, scheduler, warmUp)
	reqID := int64(0)

	godes.AddRunner(lb)
	godes.Run()
	for godes.GetSystemTime() < duration.Seconds() {
		lb.forward(newRequest(reqID, godes.GetSystemTime()))
		godes.Advance(ia.next())
		reqID++
	}
	lb.terminate()
	godes.WaitUntilDone()

	return Results{
		Instances:      lb.instances,
		Cost:           lb.getTotalCost(),
		Efficiency:     lb.getTotalEfficiency(),
		RequestCount:   reqID,
		SimulationTime: time.Since(before).Nanoseconds() / 1000000000,
	}
}
