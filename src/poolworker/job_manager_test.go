package poolworker

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kaspanet/kaspad/app/appmessage"
)

func TestJobBuffer(t *testing.T) {
	jm := NewJobManager()
	header := loadExampleHeader()
	for i := 0; i < 10000; i++ {
		jm.AddJob(&appmessage.RPCBlock{
			Header: header,
		})
	}
	_, exists := jm.GetJob(100) // shouldn't exist
	if exists {
		t.Fatal("job 100 should have been evicted long ago")
	}
	j, exists := jm.GetJob(9997) // should exist
	if !exists {
		t.Fatal("job 9997 should still exist")
	}
	if d := cmp.Diff(header, j.Header); d != "" {
		t.Fatalf("unexpected data: %s", d)
	}
}
