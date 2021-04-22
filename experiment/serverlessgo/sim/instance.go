package sim

import (
	"strconv"
	"strings"
	"time"

	"github.com/agoussia/godes"
)

type IInstance interface {
	receive(r *Request)
	terminate()
	IsWorking() bool
	IsTerminated() bool
	IsAvailable() bool
	GetLastWorked() float64
	GetId() string
	GetBusyTime() float64
	GetUpTime() float64
	GetIdleTime() float64
	GetEfficiency() float64
	GetCreatedTime() float64
	getReproducer() iInputReproducer
}

type instance struct {
	*godes.Runner
	id               string
	lb               iLoadBalancer
	terminated       bool
	cond             *godes.BooleanControl
	req              *Request
	createdTime      float64
	terminateTime    float64
	lastWorked       float64
	busyTime         float64
	idlenessDeadline time.Duration
	reproducer       iInputReproducer
	index            int
	tsAvailableAt    float64   // TimeStamp when the instance becomes available
	shedRT           []float64 // RT, Response Time
	shedRTIndex      int
}

func newInstance(id string, lb iLoadBalancer, idlenessDeadline time.Duration, reproducer iInputReproducer) *instance {
	return &instance{
		Runner:           &godes.Runner{},
		lb:               lb,
		id:               id,
		cond:             godes.NewBooleanControl(),
		createdTime:      godes.GetSystemTime(),
		lastWorked:       godes.GetSystemTime(),
		idlenessDeadline: idlenessDeadline,
		reproducer:       reproducer,
	}
}

func (i *instance) receive(r *Request) {
	i.req = r
	i.req.updateHops(i.id)
	i.cond.Set(true)
}

func (i *instance) terminate() {
	if !i.IsTerminated() {
		if i.GetLastWorked()+i.idlenessDeadline.Seconds() > godes.GetSystemTime() {
			i.terminateTime = godes.GetSystemTime()
		} else {
			i.terminateTime = i.GetLastWorked() + i.idlenessDeadline.Seconds()
		}
		i.terminated = true
		i.cond.Set(true)
	}
}

func (i *instance) nextShed() (int, float64) {
	status := 503
	ResponseTime := i.shedRT[i.shedRTIndex] / 1000000000
	i.shedRTIndex = (i.shedRTIndex + 1) % len(i.shedRT)
	return status, ResponseTime
}

func (i *instance) dealWithTruncatedInput(body string, responseTime, tsafter, tsbefore float64) {
	unavailableTime := tsafter - tsbefore
	i.tsAvailableAt = godes.GetSystemTime() + unavailableTime
	i.shedRT = append(i.shedRT, responseTime)
	rts := strings.Split(body, ":")
	for j := 1; j < len(rts); j++ {
		rtFloat64, err := strconv.ParseFloat(rts[j], 64)
		if err == nil {
			i.shedRT = append(i.shedRT, rtFloat64)
		}
	}
}

func (i *instance) next() (int, float64) {
	var status int
	var responseTime float64
	if i.IsAvailable() {
		var body string
		var tsbefore, tsafter float64
		status, responseTime, body, tsbefore, tsafter = i.reproducer.next()
		if status == 503 {
			i.dealWithTruncatedInput(body, responseTime, tsbefore, tsafter)
		}
	} else {
		status, responseTime = i.nextShed()
	}
	return status, responseTime
}

func (i *instance) Run() {
	for {
		i.cond.Wait(true)
		if i.IsTerminated() {
			i.cond.Set(false)
			break
		}
		status, responseTime := i.next()
		i.req.updateStatus(status)
		i.req.updateResponseTime(responseTime)
		i.busyTime += responseTime

		godes.Advance(responseTime)
		i.lastWorked = godes.GetSystemTime()
		i.lb.response(i.req)

		i.cond.Set(false)
	}
}

func (i *instance) IsWorking() bool {
	return i.cond.GetState()
}

func (i *instance) IsTerminated() bool {
	return i.terminated
}

func (i *instance) IsAvailable() bool {
	return godes.GetSystemTime() >= i.tsAvailableAt
}

func (i *instance) GetId() string {
	return i.id
}

func (i *instance) GetUpTime() float64 {
	return i.terminateTime - i.createdTime
}

func (i *instance) GetIdleTime() float64 {
	return i.GetUpTime() - i.GetBusyTime()
}

func (i *instance) GetBusyTime() float64 {
	return i.busyTime
}

func (i *instance) GetLastWorked() float64 {
	return i.lastWorked
}

func (i *instance) GetEfficiency() float64 {
	return i.GetBusyTime() / i.GetUpTime()
}

func (i *instance) GetCreatedTime() float64 {
	return i.createdTime
}

func (i *instance) getReproducer() iInputReproducer {
	return i.reproducer
}
