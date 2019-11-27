package controller

// Controller ...
type Controller struct {
	//Tracer *tracer.Tracer
}

// New ...
func New() (*Controller, error) {
	// t, err := tracer.NewTracer(&tcpEventTracer{})
	// if err != nil {
	// 	return nil, err
	// }
	// t.Start()
	return &Controller{
		//	Tracer: t,
	}, nil
}

func (c *Controller) Start() {
	//c.Tracer.Start()
}

func (c *Controller) Stop() {
	//c.Tracer.Stop()
}
