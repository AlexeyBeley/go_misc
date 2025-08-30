package aws_api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AlexeyBeley/go_misc/logger"
	clients "github.com/AlexeyBeley/go_misc/aws_api/clients"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

var lg = &(logger.Logger{AddDateTime: true})

type CleanerConfig struct {
	Region  *string `json:"Region"`
	Profile *string `json:"Profile"`
}

type Cleaner struct {
	Config *CleanerConfig
}

func CleanerNew(config *CleanerConfig) (*Cleaner, error) {
	new := &Cleaner{Config: config}
	return new, nil
}

func (cleaner *Cleaner) StartLogGroupStreamCleanerTask(asyncOrchestrator *AsyncOrchestrator, logs_api *clients.CloudwatchLogsAPI, logGroup *cloudwatchlogs.LogGroup, stream *cloudwatchlogs.LogStream) error {
	objects := []any{}
	err := logs_api.YieldStreamEvents(&cloudwatchlogs.GetLogEventsInput{
		StartFromHead: clients.BoolPtr(false),
		LogGroupName:  logGroup.LogGroupName,
		LogStreamName: stream.LogStreamName,
	}, clients.AggregatorInitializerNG(&objects))

	if err != nil {
		return err
	}
	if len(objects) == 0 {

		epochLimitRetention := (time.Now().UTC().Unix() - *logGroup.RetentionInDays*24*60*60) * 1000
		if *stream.LastEventTimestamp > epochLimitRetention {
			return fmt.Errorf("was not able to fetch events from stream inside retention range: %s", *stream.LogStreamName)
		}

		out, err := logs_api.DisposeStream(&cloudwatchlogs.DeleteLogStreamInput{LogGroupName: logGroup.LogGroupName, LogStreamName: stream.LogStreamName})
		if err != nil {
			lg.WarningF("Disposing log stream failed: %v, %v", out, err)
			return err
		}
	}
	return nil
}

func (cleaner *Cleaner) StartLogGroupCleanerTask(asyncOrchestrator *AsyncOrchestrator, logs_api *clients.CloudwatchLogsAPI, logGroup *cloudwatchlogs.LogGroup) error {
	err := logs_api.YieldCloudwatchLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: logGroup.LogGroupName,
		OrderBy:      clients.StrPtr("LastEventTime"),
		Descending:   clients.BoolPtr(false),
	}, func(streamAny any) (bool, error) {

		stream, ok := streamAny.(*cloudwatchlogs.LogStream)
		if !ok {
			return false, fmt.Errorf("cast error: %v", streamAny)
		}

		work := func() (any, error) {
			err := cleaner.StartLogGroupStreamCleanerTask(asyncOrchestrator, logs_api, logGroup, stream)
			return nil, err
		}
		task := &Task{Work: work}
		asyncOrchestrator.AddTask(task)

		return true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (cleaner *Cleaner) WorkLogGroupCleanerGenerator(asyncOrchestrator *AsyncOrchestrator) func() (any, error) {
	return func() (any, error) {

		logs_api := clients.CloudwatchLogsAPINew(cleaner.Config.Region, cleaner.Config.Profile)
		logGroups, err := logs_api.GetLogGroups(nil)
		if err != nil {
			return nil, err
		}
		for _, logGroup := range logGroups {
			//if *logGroup.LogGroupName != "" {
			//		continue
			//	}
			var RetentionInDays string
			if logGroup.RetentionInDays == nil {
				RetentionInDays = "Nil"
			} else {
				RetentionInDays = fmt.Sprintf("%d", *logGroup.RetentionInDays)
			}
			lg.InfoF("Log group %s, retention %s", *logGroup.LogGroupName, RetentionInDays)

			work := func() (any, error) {
				err := cleaner.StartLogGroupCleanerTask(asyncOrchestrator, logs_api, logGroup)
				return nil, err
			}
			task := &Task{Work: work}
			asyncOrchestrator.AddTask(task)

		}
		return nil, nil
	}
}

func (cleaner *Cleaner) CleanLogGroupsExpired() error {
	asyncOrchestrator, err := AsyncOrchestratorNew(5)
	if err != nil {
		return err
	}
	task := &Task{Work: cleaner.WorkLogGroupCleanerGenerator(asyncOrchestrator)}
	asyncOrchestrator.AddTask(task)
	asyncOrchestrator.Wait()

	return nil
}

type AsyncOrchestrator struct {
	TaskId        *int
	WorkerPool    *WorkerPool
	Tasks         map[int]*Task
	SyncSemaphore chan bool
}

type Task struct {
	ID       int
	Work     func() (any, error)
	Result   any
	Error    error
	Started  bool
	Finished bool
}

type WorkerPool struct {
	NumWorkers    int
	WG            *sync.WaitGroup
	Tasks         chan *Task
	WorkerContext context.Context
	CancelFunc    context.CancelFunc
}

// Generator
func AsyncOrchestratorNew(NumWorkers int) (*AsyncOrchestrator, error) {
	asyncOrchestrator := &AsyncOrchestrator{}
	pool, err := WorkerPoolNew(NumWorkers)
	if err != nil {
		return nil, err
	}
	asyncOrchestrator.WorkerPool = pool
	asyncOrchestrator.Tasks = map[int]*Task{}
	asyncOrchestrator.TaskId = clients.IntPtr(0)
	asyncOrchestrator.SyncSemaphore = make(chan bool, 1)
	go asyncOrchestrator.TaskStarter()

	return asyncOrchestrator, nil
}
func (asyncOrchestrator *AsyncOrchestrator) AddTask(task *Task) error {
	asyncOrchestrator.SyncSemaphore <- true

	task.ID = *asyncOrchestrator.TaskId
	asyncOrchestrator.Tasks[task.ID] = task
	*asyncOrchestrator.TaskId = *asyncOrchestrator.TaskId + 1
	lg.InfoF("Added task. Tasks count: %d", len(asyncOrchestrator.Tasks))

	<-asyncOrchestrator.SyncSemaphore

	return nil
}

func (asyncOrchestrator *AsyncOrchestrator) Wait() error {
	for {
		finishedTasks := 0
		asyncOrchestrator.SyncSemaphore <- true
		allTasksLen := len(asyncOrchestrator.Tasks)
		for _, task := range asyncOrchestrator.Tasks {
			if task.Finished {
				finishedTasks++
			}

		}

		<-asyncOrchestrator.SyncSemaphore

		if finishedTasks == allTasksLen {
			return nil
		}

		select {
		case <-asyncOrchestrator.WorkerPool.WorkerContext.Done():
			return nil
		case <-time.After(time.Second * 5):
			lg.InfoF("Waiting for all tasks to finish: %d/%d", finishedTasks, allTasksLen)
		}

	}
}

func (asyncOrchestrator *AsyncOrchestrator) TaskStarter() error {
	for {

		var taskToStart *Task

		asyncOrchestrator.SyncSemaphore <- true

		for _, task := range asyncOrchestrator.Tasks {
			if task.Started {
				continue
			}
			taskToStart = task
			break
		}

		<-asyncOrchestrator.SyncSemaphore

		//blocked until the WorkerPool can not receive task
		if taskToStart != nil {
			asyncOrchestrator.WorkerPool.AddTask(taskToStart)
		}

		select {
		case <-asyncOrchestrator.WorkerPool.WorkerContext.Done():
			return nil
		case <-time.After(time.Millisecond * 10):
			continue
		}
	}
}

// Generator
func WorkerPoolNew(NumWorkers int) (*WorkerPool, error) {
	workerCtx, cancelFunc := context.WithCancel(context.Background())
	WG := &sync.WaitGroup{}
	WP := &WorkerPool{NumWorkers: NumWorkers,
		WG:            WG,
		WorkerContext: workerCtx,
		CancelFunc:    cancelFunc}

	for i := range NumWorkers {
		WG.Add(1)
		go WP.Work(i + 1)
	}

	WP.Tasks = make(chan *Task, NumWorkers)
	return WP, nil
}

func (workerPool *WorkerPool) AddTask(task *Task) error {
	task.Started = true
	workerPool.Tasks <- task
	return nil
}

func (workerPool *WorkerPool) Work(wid int) error {
	for {
		select {
		case task, ok := <-workerPool.Tasks:
			if !ok {
				lg.ErrorF("channel closed")
			}
			lg.InfoF("Async worker %d starting doing work %v", wid, task.ID)
			task.Result, task.Error = task.Work()
			task.Finished = true
		case <-workerPool.WorkerContext.Done():
			return nil
		}

	}

}
