package sim

type Listener interface {
	RequestFinished(r *Request)
}
