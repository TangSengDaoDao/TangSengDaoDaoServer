package pool

import (
	"sync/atomic"
)

var WorkerChannel = make(chan chan *Job)

type JobStatistics struct {
	Executing int64 // Total number of jobs executed
	Total     int64
}

type Collector struct {
	Work  chan *Job // receives jobs to send to workers
	End   chan bool // when receives bool stops workers
	jobS  *JobStatistics
	queue *Queue
}

func StartDispatcher(workerCount int64) Collector {
	var i int64
	var workers []Worker
	input := make(chan *Job) // channel to recieve work
	end := make(chan bool)   // channel to spin down workers
	jobFinished := make(chan bool)
	collector := Collector{
		Work:  input,
		End:   end,
		jobS:  &JobStatistics{},
		queue: NewQueue(),
	}

	go collector.loopPop()

	for i < workerCount {
		i++
		worker := Worker{
			ID:            i,
			Channel:       make(chan *Job),
			WorkerChannel: WorkerChannel,
			End:           make(chan struct{}),
			jobFinished:   jobFinished,
		}
		worker.Start()
		workers = append(workers, worker) // stores worker
	}
	go func() {
		for {
			select {
			case <-jobFinished: // job finished
				atomic.AddInt64(&collector.jobS.Executing, -1)
			}
		}
	}()

	// start collector
	go func() {
		for {
			select {
			case <-end:
				for _, w := range workers {
					w.Stop() // stop worker
				}
				return
			case job := <-input:
				collector.queue.Push(job)
			}
		}
	}()

	return collector
}

func (c Collector) loopPop() {
	for {
		jobObj := c.queue.Pop()
		atomic.AddInt64(&c.jobS.Total, 1)
		worker := <-WorkerChannel // wait for available channel
		atomic.AddInt64(&c.jobS.Executing, 1)
		worker <- jobObj.(*Job) // dispatch work to worker
	}

}

func (c Collector) GetStatistics() *JobStatistics {
	return c.jobS
}

func (c Collector) Waiting() int {
	return c.queue.Len()
}
