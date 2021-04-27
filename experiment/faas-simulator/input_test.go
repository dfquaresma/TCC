package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gcinterceptor/gci-simulator/serverless/sim"
)

func TestBuildEntryArray_Error(t *testing.T) {
	var testData = []struct {
		desc string
		row  [][]string
	}{
		{"EmptyEntry", [][]string{}},
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			_, err := buildEntryArray(d.row)
			if err == nil {
				t.Fatal("Error expected")
			}
		})
	}
}

func TestReadRecords_Success(t *testing.T) {
	in := `status,request_time
200,0.019
200,0.023
503,0.001`

	want := [][]string{{"200", "0.019"}, {"200", "0.023"}, {"503", "0.001"}}
	got, err := readRecords(strings.NewReader(in), "test.csv")
	if err != nil {
		t.Fatalf("Error not expected: %q", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("Want: %v, got: %v", want, got)
	}
}

func TestToEntry_Success(t *testing.T) {
	var testData = []struct {
		desc string
		row  []string
		want sim.InputEntry
	}{
		
		{
			desc: "Success",
			// Row format: id,status,response_time,body,tsbefore,tsafter
			row: []string{"1", "200", "0.019", "body", "0", "0.019"},
			want: sim.InputEntry{200, 0.019, "body", 0, 0.019},
		},
		{
			desc: "Error",
			row: []string{"2", "503", "0.250", "body", "0", "0.250"},
			want: sim.InputEntry{503, 0.250, "body", 0, 0.250},
		},
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			got, err := toEntry(d.row)
			if err != nil {
				t.Fatalf("Error while using toEntry function: %q", err)
			}
			if !reflect.DeepEqual(d.want, got) {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}

func TestToEntry_Error(t *testing.T) {
	var testData = []struct {
		desc string
		row  []string
	}{
		{"StatusString", []string{"string", "0.019"}},
		{"DurationString", []string{"200", "string"}},
		{"StatusFloat", []string{"0.200", "0.200"}},
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			_, err := toEntry(d.row)
			if err == nil {
				t.Fatal("Error expected")
			}
		})
	}
}

func TestBuildEntryArray_Success(t *testing.T) {
	var testData = []struct {
		desc string
		row  [][]string
		want []sim.InputEntry
	}{
		{"OneEntry", [][]string{{"1", "503", "0.250", "body", "0", "0.250"}},
			[]sim.InputEntry{{503, 0.250, "body", 0, 0.250}}},
		{"ManyEntries", [][]string{{"2", "200", "0.019", "body", "0", "0.019"}, {"3", "503", "0.250", "body", "0", "0.250"}},
			[]sim.InputEntry{{200, 0.019, "body", 0, 0.019}, {503, 0.250, "body", 0, 0.250}}},
	}
	for _, d := range testData {
		t.Run(d.desc, func(t *testing.T) {
			got, err := buildEntryArray(d.row)
			if err != nil {
				t.Fatalf("Error while using toEntry function: %q", err)
			}
			if !reflect.DeepEqual(d.want, got) {
				t.Fatalf("Want: %v, got: %v", d.want, got)
			}
		})
	}
}
