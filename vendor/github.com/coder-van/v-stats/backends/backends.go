package backends

import (
	"github.com/coder-van/v-stats/metrics"
	"time"
	"fmt"
	"sync"
	gb "github.com/coder-van/v-stats/backends/graghite"
	"strings"
)

type InterfaceBackend interface {
	Flush(batch []byte) error
}

const MetricBatchSize = 128

func NewBackendManger() *BackendManger {
	
	b := &BackendManger{
		RegisteredBackends: make(map[string]InterfaceBackend, 10),
		metricsBuffer:      NewBuffer(MetricBatchSize),
	}
	return b
}


type BackendManger struct {
	RegisteredBackends map[string]InterfaceBackend
	metricsBuffer      *Buffer
}


func (b *BackendManger) RegisterBackend(name string, backend InterfaceBackend)  {
	if _, ok:= b.RegisteredBackends[name]; ok {
		return
	}
	b.RegisteredBackends[name] = backend
}

func (b *BackendManger) RegisterGraphite(addr string)  {
	// address, _ := net.ResolveTCPAddr("net", addr)
	g := gb.NewGraphite(addr)
	b.RegisterBackend("graghite:"+addr, g)
}

func (b *BackendManger) Run(shutdown chan struct{}, datapointSourceCh chan metrics.MetricDataPoint,
	interval time.Duration) error {
	// wait the collect threads to run, so that
	// emitter will emitted by metrics flush action.
	//time.Sleep(200 * time.Millisecond)
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	//length = len(e.bm.RegisteredBackends)
	//chs = make([]chan int)
	for {
		select {
		case <-shutdown:
			fmt.Println("Hang on, emitting any cached metrics before shutdown")
			b.Flush()
			return nil
		case <-ticker.C:
			b.Flush()
		case m := <- datapointSourceCh:
			b.add(m)
		}
	}
}

func (b *BackendManger) add(dp metrics.MetricDataPoint) {
	// fmt.Printf("add datapoint %s", dp.String())
	d := dp.String()
	if strings.Index(d, "cpu.cpu-total.idle") >= 0 {
		fmt.Println(d)
	}
	b.metricsBuffer.Add(dp)
	// 当缓存到了80%的时候就刷新缓存了
	if b.metricsBuffer.Len() >= int(MetricBatchSize*0.5) {
		b.Flush()
	}
}



func (b *BackendManger) Flush() {
	var wg sync.WaitGroup
	batch := b.metricsBuffer.Batch(MetricBatchSize)
	
	wg.Add(len(b.RegisteredBackends))
	for _, bm := range b.RegisteredBackends{
		go func() {
			defer wg.Done()
			err := bm.Flush(batch)
			if err != nil {
				fmt.Printf("Error occurred when posting to Forwarder API: %s \n", err.Error())
			}
			
		}()
	}
	
	wg.Wait()
}