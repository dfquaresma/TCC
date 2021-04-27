package sim

import (
	"reflect"
	"testing"
	"time"

	"github.com/agoussia/godes"
)

func TestFoward(t *testing.T) {
	lb := &loadBalancer{
		arrivalQueue: godes.NewFIFOQueue("arrival"),
		arrivalCond:  godes.NewBooleanControl(),
		warmUp:       0,
	}
	type Want struct {
		queueSize         int
		arrivalCondBefore bool
		arrivalCondAfter  bool
		expectedError     bool
	}
	data := []struct {
		desc string
		req  *Request
		want *Want
	}{
		{"Nil request", nil, &Want{0, false, false, true}},
		{"First request", &Request{}, &Want{1, false, true, false}},
		{"Following request", &Request{}, &Want{2, true, true, false}},
	}
	for _, d := range data {
		t.Run(d.desc, func(t *testing.T) {
			arrivalCondBefore := lb.arrivalCond.GetState()
			err := lb.forward(d.req)
			expectedError := err != nil
			arrivalCondAfter := lb.arrivalCond.GetState()
			queueSized := lb.arrivalQueue.Len()
			got := &Want{queueSized, arrivalCondBefore, arrivalCondAfter, expectedError}
			if !reflect.DeepEqual(d.want, got) {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}

type voidListener struct{}

func (t voidListener) RequestFinished(r *Request) {}

func TestResponse(t *testing.T) {
	lb := &loadBalancer{
		inputs:   [][]InputEntry{{{200, 0.5, "body", 0, 0.5}}},
		listener: voidListener{},
		warmUp:   0,
	}
	type Want struct {
		responsed     int
		reforwarded   int
		expectedError bool
	}
	data := []struct {
		desc string
		req  *Request
		want *Want
	}{
		{"Nil request", nil, &Want{0, 0, true}},
		{"Success", &Request{Status: 200}, &Want{1, 0, false}},
		{"Unavailable", &Request{Status: 503}, &Want{1, 1, false}},
	}
	for _, d := range data {
		t.Run(d.desc, func(t *testing.T) {
			err := lb.response(d.req)
			expectedError := err != nil
			responsed := lb.finishedReqs
			reforwarded := len(lb.instances)
			got := &Want{responsed, reforwarded, expectedError}
			if !reflect.DeepEqual(d.want, got) {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}

func TestLBTerminate(t *testing.T) {
	type TestData struct {
		desc string
		lb   *loadBalancer
		want bool
	}
	var testData = []TestData{
		{"NoInstance", &loadBalancer{
			arrivalCond: godes.NewBooleanControl(),
			instances:   make([]IInstance, 0),
			warmUp:      0,
		}, true},
		{"OneInstance", &loadBalancer{
			arrivalCond: godes.NewBooleanControl(),
			instances:   []IInstance{&instance{id: "0", cond: godes.NewBooleanControl()}},
			warmUp:      0,
		}, true},
		{"ManyInstances", &loadBalancer{
			arrivalCond: godes.NewBooleanControl(),
			warmUp:      0,
			instances: []IInstance{
				&instance{id: "1", cond: godes.NewBooleanControl()},
				&instance{id: "2", cond: godes.NewBooleanControl()},
				&instance{id: "3", cond: godes.NewBooleanControl()},
			},
		}, true},
	}
	checkFunc := func(want, got bool) {
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("Want: %v, got: %v", want, got)
		}
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			d.lb.terminate()
			var got bool
			for _, i := range d.lb.instances {
				got = i.IsTerminated()
				checkFunc(d.want, got)
			}
			got = d.lb.isTerminated
			checkFunc(d.want, got)
		})
	}
}

func TestNextInstanceInputs(t *testing.T) {
	type TestData struct {
		desc      string
		lb        *loadBalancer
		nextCalls int
		want      [][]InputEntry
	}
	var testData = []TestData{
		{"OneInputEntry", &loadBalancer{
			warmUp: 0,
			inputs: [][]InputEntry{{{200, 0.5, "body", 0, 0.5}}},
		}, 2, [][]InputEntry{{{200, 0.5, "body", 0, 0.5}}, {{200, 0.5, "body", 0, 0.5}}}},
		{"ManyInputEntry", &loadBalancer{
			warmUp: 0,
			inputs: [][]InputEntry{
				{{200, 0.5, "body", 0, 0.5}, {503, 0.5, "body", 0, 0.5}}, {}, {{200, 0.5, "body", 0, 0.5}}}}, 5,
			[][]InputEntry{
				{{200, 0.5, "body", 0, 0.5}, {503, 0.5, "body", 0, 0.5}}, {}, {{200, 0.5, "body", 0, 0.5}},
				{{200, 0.5, "body", 0, 0.5}, {503, 0.5, "body", 0, 0.5}}, {}}},
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			for i := 0; i < d.nextCalls; i++ {
				got := d.lb.nextInstanceInputs()
				if !reflect.DeepEqual(d.want[i], got) {
					t.Fatalf("Want: %v, got: %v", d.want[i], got)
				}
			}
		})
	}
}

func TestNextInstance_HopedRequest(t *testing.T) {
	lb := &loadBalancer{
		warmUp:      0,
		Runner:      &godes.Runner{},
		arrivalCond: godes.NewBooleanControl(),
		inputs:      [][]InputEntry{{{200, 0.5, "body", 0, 0.5}}},
		instances: []IInstance{
			&instance{id: "i0-f0", terminated: false, cond: godes.NewBooleanControl()},
			&instance{id: "i1-f0", terminated: false, cond: godes.NewBooleanControl()},
			&instance{id: "i2-f0", terminated: true, cond: godes.NewBooleanControl()},
			&instance{id: "i3-f0", terminated: false, cond: godes.NewBooleanControl()},
		},
	}
	data := []struct {
		desc string
		req  *Request
		want string
	}{
		{"Free Instance", &Request{}, "i0-f0"},
		{"Busy Instance", &Request{}, "i1-f0"},
		{"Terminated Instance", &Request{Hops: []string{"0", "1"}}, "i3-f0"},
		{"New Instance Required", &Request{Hops: []string{"0", "1", "2", "3"}}, "i4-f0"},
	}
	for _, d := range data {
		t.Run(d.desc, func(t *testing.T) {
			nextInstance := lb.nextInstance(d.req)
			nextInstance.receive(d.req)
			got := nextInstance.GetId()
			if d.want != got {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}

type TestInstance struct {
	*instance
	id         string
	terminated bool
	lastWorked float64
}

func (t *TestInstance) terminate()             { t.terminated = true }
func (t *TestInstance) scaleDown()             { t.terminated = true }
func (t *TestInstance) IsTerminated() bool     { return t.terminated }
func (t *TestInstance) GetLastWorked() float64 { return t.lastWorked }
func (t *TestInstance) GetId() string          { return t.id }
func (t *TestInstance) IsWorking() bool        { return false }

func TestTryScaleDown(t *testing.T) {
	idleness, _ := time.ParseDuration("5s")
	type TestData struct {
		desc string
		lb   *loadBalancer
		want []bool
	}
	var testData = []TestData{
		{"NoInstances", &loadBalancer{
			warmUp:           0,
			idlenessDeadline: idleness,
			instances:        make([]IInstance, 0),
		}, make([]bool, 0)},
		{"OneInstance", &loadBalancer{
			warmUp:           0,
			idlenessDeadline: idleness,
			instances:        []IInstance{&TestInstance{id: "0", terminated: false, lastWorked: -5.0}},
		}, []bool{true}},
		{"ManyInstances", &loadBalancer{
			warmUp:           0,
			idlenessDeadline: idleness,
			instances: []IInstance{
				&TestInstance{id: "0", terminated: false, lastWorked: -5.0},
				&TestInstance{id: "1", terminated: false, lastWorked: 0.0},
				&TestInstance{id: "2", terminated: false, lastWorked: -5.0},
				&TestInstance{id: "3", terminated: false, lastWorked: -1.0},
				&TestInstance{id: "4", terminated: false, lastWorked: -8.0},
			},
		}, []bool{true, false, true, false, true}},
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			d.lb.tryScaleDown()
			got := make([]bool, 0)
			for _, i := range d.lb.instances {
				got = append(got, i.IsTerminated())
			}
			if !reflect.DeepEqual(d.want, got) {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}

func TestTryScaleDownWorkingInstance(t *testing.T) {
	idleness, _ := time.ParseDuration("5s")
	instance := &instance{id: "0", cond: godes.NewBooleanControl(), terminated: false, lastWorked: -5.0}
	lb := &loadBalancer{
		warmUp:           0,
		idlenessDeadline: idleness,
		instances:        []IInstance{instance},
	}
	instance.cond.Set(true)
	lb.tryScaleDown()
	got := make([]bool, 0)
	for _, i := range lb.instances {
		got = append(got, i.IsTerminated())
	}
	want := false
	if want != got[0] {
		t.Fatalf("Want: %v, got: %v", want, got)
	}
}

func TestNextInstanceGCIOptimal(t *testing.T) {
	lb := &loadBalancer{
		warmUp:    0,
		scheduler: 2,
		instances: make([]IInstance, 0),
		inputs: [][]InputEntry{{
			{Status: 200, ResponseTime: 1, Body: "coldstart"},
			{Status: 200, ResponseTime: 0.1, Body: "normal"},
			{Status: 503, ResponseTime: 0.01, Body: "shed"},
		}},
	}
	iNoShed := lb.newInstance(&Request{Status: 200})
	statusGot, responseGot, bodyGot, befGot, aftGot := iNoShed.getReproducer().next()
	got := &InputEntry{statusGot, responseGot, bodyGot, befGot, aftGot}
	want := &InputEntry{200, 0.1, "normal", 0, 0}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("Testing request 200, Want: %v, got: %v", want, got)
	}
	iAfterShed := lb.newInstance(&Request{Status: 503})
	statusGot, responseGot, bodyGot, befGot, aftGot = iAfterShed.getReproducer().next()
	got = &InputEntry{statusGot, responseGot, bodyGot, befGot, aftGot}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("Testing request 503, Want: %v, got: %v", want, got)
	}
}
