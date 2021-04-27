// This binary is needed because the simulation can only run in the main thread.
package main

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"sort"
	"time"

	"github.com/gcinterceptor/gci-simulator/serverless/sim"
)

type collectorListener []sim.Request

func (rs *collectorListener) RequestFinished(req *sim.Request) {
	*rs = append(*rs, *req)
}

func main() {
	duration := 80 * time.Millisecond
	idlenessDeadline := 6 * time.Millisecond
	input := [][]sim.InputEntry{
		{{200, 0.015, "body", 0, 0.015}, {200, 0.008, "body", 0, 0.008}, {503, 0.0001, "body", 0, 0.0001}},
		{{200, 0.011, "body", 0, 0.011}},
		{{200, 0.005, "body", 0, 0.005}, {200, 0.005, "body", 0, 0.005}, {503, 0.0002, "body", 0, 0.0002}},
	}
	scheduler := 0
	var simulatedRequests collectorListener
	warmup := 0
	res := sim.Run(duration, idlenessDeadline, sim.NewConstantInterArrival(0.01), input, &simulatedRequests, scheduler, warmup)

	if len(res.Instances) != 5 {
		log.Fatalf("number of instances - want:5 got:%+v", len(res.Instances))
	}
	expectedRequests := []sim.Request{
		{ID: 0, Status: 200, CreatedTime: 0.00, ResponseTime: 0.015, Hops: []string{"i0-f0"}},        // response time from {200, 0.015} of instance 0
		{ID: 1, Status: 200, CreatedTime: 0.01, ResponseTime: 0.011, Hops: []string{"i1-f1"}},        // response time from {200, 0.011} of instance 1
		{ID: 2, Status: 200, CreatedTime: 0.02, ResponseTime: 0.008, Hops: []string{"i0-f0"}},        // response time from {200, 0.008} of instance 0
		{ID: 3, Status: 200, CreatedTime: 0.03, ResponseTime: 0.0051, Hops: []string{"i0-f0", "i2-f2"}}, // response time from {503, 0.0001} of instance 0 plus {200, 0.005} of instance 2
		{ID: 4, Status: 200, CreatedTime: 0.04, ResponseTime: 0.005, Hops: []string{"i2-f2"}},        // response time from {200, 0.005} of instance 2
		{ID: 5, Status: 200, CreatedTime: 0.05, ResponseTime: 0.0152, Hops: []string{"i2-f2", "i3-f0"}}, // response time from {503, 0.0002} of instance 0 plus {200, 0.015} of instance 3
		{ID: 6, Status: 200, CreatedTime: 0.06, ResponseTime: 0.011, Hops: []string{"i4-f1"}},           // response time from {200, 0.011} of instance 4
		{ID: 7, Status: 200, CreatedTime: 0.07, ResponseTime: 0.008, Hops: []string{"i3-f0"}},           // response time from {200, 0.008} of instance 3
	}
	if len(expectedRequests) != len(simulatedRequests) {
		log.Fatalf("number of requests - want:%+v got:%+v", len(expectedRequests), len(simulatedRequests))
	}
	for i, rg := range simulatedRequests {
		rw := expectedRequests[i]
		if rw.ID != rg.ID {
			log.Fatalf("request's ID output - want:%+v got:%+v", rw.ID, rg.ID)
		}
		if rw.Status != rg.Status {
			log.Fatalf("request's Status output - want:%+v got:%+v", rw.Status, rg.Status)
		}
		if rw.ResponseTime != rg.ResponseTime {
			log.Fatalf("request's ResponseTime output - want:%+v got:%+v", rw.ResponseTime, rg.ResponseTime)
		}
		if !reflect.DeepEqual(rw.Hops, rg.Hops) {
			log.Fatalf("request's Hops output - want:%+v got:%+v", rw.Hops, rg.Hops)
		}
		if math.Abs(rw.CreatedTime-rg.CreatedTime) > 0.0001 {
			log.Fatalf("request's createdtime output - want:%+v got:%+v", rw.CreatedTime, rg.CreatedTime)
		}
	}
	instanceData := []struct {
		id          string
		createdTime float64
		upTime      float64
		efficiency  float64
	}{
		{"i0-f0", 0.00, 0.0361, 0.6398},
		{"i1-f1", 0.01, 0.017, 0.6470},
		{"i2-f2", 0.0301, 0.0262, 0.3893},
		{"i3-f0", 0.0502, 0.030, 0.7700},
		{"i4-f1", 0.06, 0.017, 0.6470},
	}
	instancesUsed := res.Instances
	if len(instanceData) != len(instancesUsed) {
		log.Fatalf("number of instances - want:%+v got:%+v", len(instanceData), len(instancesUsed))
	}
	sort.SliceStable(instancesUsed, func(i, j int) bool { return instancesUsed[i].GetId() < instancesUsed[j].GetId() })
	for i, ig := range instancesUsed {
		iw := instanceData[i]
		if iw.id != ig.GetId() {
			log.Fatalf("Instance's ID output - want:%+v got:%+v", iw.id, ig.GetId())
		}
		if math.Abs(iw.upTime-ig.GetUpTime()) > 0.001 {
			log.Fatalf("Instance's upTime output - want:%+v got:%+v", iw.upTime, ig.GetUpTime())
		}
		if math.Abs(iw.efficiency-ig.GetEfficiency()) > 0.005 {
			log.Fatalf("Instance's efficiency output - want:%+v got:%+v", iw.efficiency, ig.GetEfficiency())
		}
		if math.Abs(iw.createdTime-ig.GetCreatedTime()) > 0.001 {
			log.Fatalf("Instance's createdtime output - want:%+v got:%+v", iw.createdTime, ig.GetCreatedTime())
		}
	}
	if math.Abs(1000*res.Cost-126.3) > 0.5 { // 36.1 + 17 + 26.2 + 30 + 17 = 126.3
		// where 36.1, 17, 26.2, 30 and 17 are the uptime of instances 0, 1, 2, 3 and 4, respectively
		log.Fatalf("instances cost - want:%+v got:%+v", 126.3, 1000*res.Cost)
	}
	if math.Abs(res.Efficiency-0.61866396416) > 0.001 { // (23.1/36.1 + 11/17 + 10.2/26.2 + 23.1/30 + 11/17) / 5 = 0.61866396416
		// where 23.1/36.1, 11/17, 10.2/26.2,  23.1/30 and 11/17 are the uptime of instances 0, 1, 2, 3 and 4, respectively
		log.Fatalf("instances efficiency - want:%+v got:%+v", 0.61866396416, res.Efficiency)
	}
	fmt.Println("OK")
}
