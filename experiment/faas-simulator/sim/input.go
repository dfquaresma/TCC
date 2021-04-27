package sim

type iInputReproducer interface {
	next() (int, float64, string, float64, float64)
}

type inputReproducer struct {
	index   int
	warmed  bool
	entries []InputEntry
}

type warmedinputReproducer struct {
	index   int
	entries []InputEntry
}

func newInputReproducer(input []InputEntry, warmUp int) iInputReproducer {
	input = append([]InputEntry{input[0]}, input[warmUp + 1:]...) // remove warmUp, but leave the coldstart entry
	return &inputReproducer{entries: input}
}

func newWarmedInputReproducer(input []InputEntry, warmUp int) iInputReproducer {
	if len(input) > 1 {
		input = input[warmUp + 1:] // remove warmUp and coldstart
	}
	return &warmedinputReproducer{entries: input}
}

func (r *inputReproducer) next() (int, float64, string, float64, float64) {
	e := r.entries[r.index]
	r.index = (r.index + 1) % len(r.entries)
	r.setWarm()
	return e.Status, e.ResponseTime, e.Body, e.TsBefore, e.TsAfter
}

func (r *inputReproducer) setWarm() {
	if !r.warmed {
		r.warmed = true
		if len(r.entries) > 1 {
			r.entries = r.entries[1:] // remove first entry
			r.index = 0
		}
	}
}

func (r *warmedinputReproducer) next() (int, float64, string, float64, float64) {
	e := r.entries[r.index]
	r.index = (r.index + 1) % len(r.entries)
	return e.Status, e.ResponseTime, e.Body, e.TsBefore, e.TsAfter
}

// InputEntry packs information about one response.
type InputEntry struct {
	Status       int
	ResponseTime float64
	Body         string
	TsBefore     float64
	TsAfter      float64
}
