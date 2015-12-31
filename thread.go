package pool

//just for test

type Runnable interface {
	Run()
}

type FuncRun struct {
	f func()
}

func (this *FuncRun) Run() {
	this.f()
}

type Thread struct {
	target Runnable
	done   chan int
}

func NewThread(f func()) *Thread {
	return NewThreadWithRunnable(&FuncRun{f: f})
}

func NewThreadWithRunnable(run Runnable) *Thread {
	return &Thread{target: run, done: make(chan int, 1)}
}

func (this *Thread) Start() {
	go func() {
		this.target.Run()
		this.done <- 0
	}()
}

func (this *Thread) Join() {
	<-this.done
}
