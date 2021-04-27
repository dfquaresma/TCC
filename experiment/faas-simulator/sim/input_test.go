package sim

import (
	"reflect"
	"testing"
)

func TestInputIReproducer(t *testing.T) {
	var testData = []struct {
		desc              string
		reproduce         iInputReproducer
		numberOfNextCalls int
		want              []InputEntry
	}{
		{"OneEntry", newInputReproducer(
			[]InputEntry{{200, 0.2, "body", 0, 0.2}}, 0), 3,
			[]InputEntry{{200, 0.2, "body", 0, 0.2}, {200, 0.2, "body", 0, 0.2}, {200, 0.2, "body", 0, 0.2}},
		},
		{"ManyEntry", newInputReproducer(
			[]InputEntry{
				{200, 0.8, "body", 0, 0.8}, {200, 0.2, "body", 0, 0.2}, {200, 0.3, "body", 0, 0.3}}, 0), 5,
			[]InputEntry{
				{200, 0.8, "body", 0, 0.8}, {200, 0.2, "body", 0, 0.2}, {200, 0.3, "body", 0, 0.3},
				{200, 0.2, "body", 0, 0.2}, {200, 0.3, "body", 0, 0.3}},
		},
		{"WarmedOneEntry", newWarmedInputReproducer(
			[]InputEntry{{200, 0.2, "body", 0, 0.2}}, 0), 3,
			[]InputEntry{{200, 0.2, "body", 0, 0.2}, {200, 0.2, "body", 0, 0.2}, {200, 0.2, "body", 0, 0.2}},
		},
		{"WarmedManyEntry", newWarmedInputReproducer(
			[]InputEntry{
				{200, 0.8, "body", 0, 0.8}, {200, 0.2, "body", 0, 0.2}, {200, 0.3, "body", 0, 0.3}}, 0), 5,
			[]InputEntry{
				{200, 0.2, "body", 0, 0.2}, {200, 0.3, "body", 0, 0.3}, {200, 0.2, "body", 0, 0.2},
				{200, 0.3, "body", 0, 0.3}, {200, 0.2, "body", 0, 0.2}},
		},
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			var got []InputEntry
			for i := 0; i < d.numberOfNextCalls; i++ {
				status, responseTime, body, tsBefore, tsAfter := d.reproduce.next()
				got = append(got, InputEntry{status, responseTime, body, tsBefore, tsAfter})
			}
			if !reflect.DeepEqual(d.want, got) {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}
