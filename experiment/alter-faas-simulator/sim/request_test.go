package sim

import (
	"reflect"
	"testing"
)

func TestHasBeenProcessed(t *testing.T) {
	var testData = []struct {
		desc    string
		req     *Request
		instace string
		want    bool
	}{
		{"EmptyHop", &Request{}, "0", false},
		{"OneHopTrue", &Request{Hops: []string{"1"}}, "1", true},
		{"OneHopFalse", &Request{Hops: []string{"1"}}, "2", false},
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			got := d.req.hasBeenProcessed(d.instace)
			if d.want != got {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}

func TestUpdateHops(t *testing.T) {
	type data struct {
		desc  string
		req   *Request
		value string
		want  []string
	}
	var updateHop = []data{
		{"UpdateEmptyHop", &Request{}, "1", []string{"1"}},
		{"UpdateNotEmptyHop", &Request{Hops: []string{"0", "5"}}, "2", []string{"0", "5", "2"}},
	}
	for _, d := range updateHop {
		t.Run(d.desc, func(t *testing.T) {
			d.req.updateHops(d.value)
			got := d.req.Hops
			if !reflect.DeepEqual(d.want, got) {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	type data struct {
		desc  string
		req   *Request
		value int
		want  int
	}
	var updateHop = []data{
		{"DefaultStatus", &Request{}, 503, 503},
		{"NotDefaultStatus", &Request{Status: 503}, 200, 200},
	}
	for _, d := range updateHop {
		t.Run(d.desc, func(t *testing.T) {
			d.req.updateStatus(d.value)
			got := d.req.Status
			if d.want != got {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}

func TestUpdateResponseTime(t *testing.T) {
	type data struct {
		desc  string
		req   *Request
		value float64
		want  float64
	}
	var updateHop = []data{
		{"DefaultResponse", &Request{}, 0.5, 0.5},
		{"NotDefaultResponse", &Request{ResponseTime: 1.5}, 0.8, 2.3},
	}
	for _, d := range updateHop {
		t.Run(d.desc, func(t *testing.T) {
			d.req.updateResponseTime(d.value)
			got := d.req.ResponseTime
			if d.want != got {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}
