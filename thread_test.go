package pool

import (
	"fmt"
	"testing"
	"time"
)

func TestThread(t *testing.T) {
	thread := NewThread(func() {
		fmt.Println("sleep start")
		time.Sleep(time.Duration(1) * time.Second)
		fmt.Println("sleep end")
	})
	thread.Start()
	thread.Join()
}

type TestRun struct {
}

func (this *TestRun) Run() {
	fmt.Println("sleep start")
	time.Sleep(time.Duration(1) * time.Second)
	fmt.Println("sleep end")
}

func TestThreadRunnable(t *testing.T) {
	thread := NewThreadWithRunnable(&TestRun{})
	thread.Start()
	thread.Join()
}
