package pool

import (
	"log"
)

type JobFunc func(id int64, data interface{})

type Job struct {
	Data    interface{}
	JobFunc JobFunc
}

type Worker struct {
	ID            int64
	WorkerChannel chan chan *Job // used to communicate between dispatcher and workers
	Channel       chan *Job
	End           chan struct{}
	jobFinished   chan bool
}

// start worker
func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerChannel <- w.Channel // when the worker is available place channel in queue
			select {
			case job := <-w.Channel: // worker has received job
				if job != nil {
					job.JobFunc(w.ID, job.Data) // do work
					w.jobFinished <- true
				}

			case <-w.End:
				return
			}
		}
	}()
}

// end worker
func (w *Worker) Stop() {
	log.Printf("worker [%d] is stopping", w.ID)
	w.End <- struct{}{}
}
