package poolworker

import (
	"fmt"
	"sync/atomic"

	"github.com/kaspanet/kaspad/app/appmessage"
)

type WorkJob struct {
	Block           *appmessage.RPCBlock
	Id              uint32
	Hash            string
	BigJobParams    []any
	NormalJobParams []any
}

const JobBufferSize = 256 // ~1 block a sec

type JobManager struct {
	Jobs    [256]*WorkJob
	counter uint32
}

func NewJobManager() *JobManager {
	return &JobManager{}
}

func (jm *JobManager) AddJob(blockTemplate *appmessage.RPCBlock) *WorkJob {
	id := atomic.AddUint32(&jm.counter, 1)
	header, _ := SerializeBlockHeader(blockTemplate)

	// 'Normal' job params
	jobParams := []any{
		fmt.Sprintf("%d", id),
		GenerateJobHeader(header),
		blockTemplate.Header.Timestamp,
	}
	// 'Big' job params
	bigJob := []any{
		fmt.Sprintf("%d", id),
		GenerateLargeJobParams(header, uint64(blockTemplate.Header.Timestamp)),
	}

	created := &WorkJob{
		Block:           blockTemplate,
		Id:              id,
		NormalJobParams: jobParams,
		BigJobParams:    bigJob,
	}

	jm.Jobs[id%JobBufferSize] = created
	return created
}

func (jm *JobManager) GetJob(id uint32) (*appmessage.RPCBlock, bool) {
	job := jm.Jobs[id%JobBufferSize]
	if job.Id != id {
		return nil, false
	}
	return job.Block, true
}
