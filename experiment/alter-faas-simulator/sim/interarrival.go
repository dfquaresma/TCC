package sim

import (
	"time"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

type InterArrival interface {
	next() float64
}

type poissonInterArrival struct {
	p *distuv.Poisson
}

func (pia *poissonInterArrival) next() float64 {
	return pia.p.Rand() / 1000
}

func NewPoissonInterArrival(lambda float64) InterArrival {
	return &poissonInterArrival{
		&distuv.Poisson{
			Lambda: lambda,
			Src:    rand.NewSource(uint64(time.Now().Nanosecond())),
		}}
}

type constantInterArrival struct {
	value float64
}

func (cia *constantInterArrival) next() float64 {
	return cia.value
}

func NewConstantInterArrival(value float64) InterArrival {
	return &constantInterArrival{value: value}
}
