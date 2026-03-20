package main

import "context"

type cronScheduler interface {
	Start()
	Stop() context.Context
}

func startWorkerScheduler(s cronScheduler) func() {
	if s == nil {
		return func() {}
	}
	s.Start()
	return func() {
		<-s.Stop().Done()
	}
}
