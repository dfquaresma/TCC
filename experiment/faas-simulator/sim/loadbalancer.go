package sim

import (
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/agoussia/godes"
)

type iLoadBalancer interface {
	forward(r *Request) error
	response(r *Request) error
}

type loadBalancer struct {
	*godes.Runner
	isTerminated       bool
	arrivalQueue       *godes.FIFOQueue
	arrivalCond        *godes.BooleanControl
	instances          []IInstance
	idlenessDeadline   time.Duration
	inputs             [][]InputEntry
	index              int
	listener           Listener
	finishedReqs       int
	scheduler          int
	warmUp             int
}

func newLoadBalancer(idlenessDeadline time.Duration, inputs [][]InputEntry, listener Listener, scheduler int, warmUp int) *loadBalancer {
	return &loadBalancer{
		Runner:             &godes.Runner{},
		arrivalQueue:       godes.NewFIFOQueue("arrival"),
		arrivalCond:        godes.NewBooleanControl(),
		instances:          make([]IInstance, 0),
		idlenessDeadline:   idlenessDeadline,
		inputs:             inputs,
		listener:           listener,
		scheduler:          scheduler,
		warmUp:             warmUp,
	}
}

func (lb *loadBalancer) forward(r *Request) error {
	if r == nil {
		return errors.New("Error while calling the LB's forward method. Request cannot be nil.")
	}
	lb.arrivalQueue.Place(r)
	lb.arrivalCond.Set(true)
	return nil
}

func (lb *loadBalancer) response(r *Request) error {
	if r == nil {
		return errors.New("Error while calling the LB's response method. Request cannot be nil.")
	}
	if r.Status == 200 {
		lb.listener.RequestFinished(r)
		lb.finishedReqs++
	} else {
		lb.nextInstance(r).receive(r)
	}
	return nil
}

func (lb *loadBalancer) terminate() {
	if !lb.isTerminated {
		for _, i := range lb.instances {
			i.terminate()
		}
		lb.isTerminated = true
		lb.arrivalCond.Set(true)
	}
}

func (lb *loadBalancer) nextInstanceInputs() []InputEntry {
	input := lb.inputs[lb.index]
	lb.index = (lb.index + 1) % len(lb.inputs)
	return input
}

func (lb *loadBalancer) nextInstance(r *Request) IInstance {
	var selected IInstance
	// sorting instances to have the most recently used ones ahead on the array
	sort.SliceStable(lb.instances, func(i, j int) bool { return lb.instances[i].GetLastWorked() > lb.instances[j].GetLastWorked() })
	for _, i := range lb.instances {
		if !i.IsWorking() && !i.IsTerminated() && !r.hasBeenProcessed(i.GetId()) {
			selected = i
			break
		}
	}
	if selected == nil {
		selected = lb.newInstance(r)
	}
	return selected
}

func (lb *loadBalancer) newInstance(r *Request) IInstance {
	newInstanceId := lb.getNewInstanceID()
	var reproducer iInputReproducer
	nextInstanceInput := lb.nextInstanceInputs()
	switch lb.scheduler {
	case 1: // Optimized Scheduler
		if r.Status != 503 {
			reproducer = newWarmedInputReproducer(nextInstanceInput, lb.warmUp)	
		} else {
			reproducer = newInputReproducer(nextInstanceInput, lb.warmUp)	
		}	
	case 2: // Optimized Scheduler considering GCI
		reproducer = newWarmedInputReproducer(nextInstanceInput, lb.warmUp)
	default: // Normal Scheduler
		reproducer = newInputReproducer(nextInstanceInput, lb.warmUp)
	}
	newInstance := newInstance(newInstanceId, lb, lb.idlenessDeadline, reproducer)
	godes.AddRunner(newInstance)
	// inserts the instance ahead of the array
	lb.instances = append([]IInstance{newInstance}, lb.instances...)
	return newInstance
}

func (lb *loadBalancer) getNewInstanceID() string {
	instanceCount := strconv.Itoa(len(lb.instances))
	fileId := strconv.Itoa(lb.index)
	return "i" + instanceCount + "-f" + fileId
}

func (lb *loadBalancer) Run() {
	for {
		lb.arrivalCond.Wait(true)
		lb.tryScaleDown()
		if lb.arrivalQueue.Len() > 0 {
			r := lb.arrivalQueue.Get().(*Request)
			lb.nextInstance(r).receive(r)
		} else {
			lb.arrivalCond.Set(false)
			if lb.isTerminated {
				break
			}
		}
	}
}

func (lb *loadBalancer) tryScaleDown() {
	for _, i := range lb.instances {
		if !i.IsWorking() && godes.GetSystemTime()-i.GetLastWorked() >= lb.idlenessDeadline.Seconds() {
			i.terminate()
		}
	}
}

func (lb *loadBalancer) getFinishedReqs() int {
	return lb.finishedReqs
}

func (lb *loadBalancer) getTotalCost() float64 {
	var totalCost float64
	for _, i := range lb.instances {
		totalCost += i.GetUpTime()
	}
	return totalCost
}

func (lb *loadBalancer) getTotalEfficiency() float64 {
	var totalEfficiency float64
	for _, i := range lb.instances {
		totalEfficiency += i.GetEfficiency()
	}
	return totalEfficiency / float64(len(lb.instances))
}
