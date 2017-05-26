package collector

import (
	"github.com/coder-van/v-stats/metrics"
	"github.com/coder-van/v-util/log"
	"time"
)

func NewCollectorManager(seconds int, size int, r metrics.Registry) *CollectorManager {
	return &CollectorManager{
		exit:            make(chan bool),
		collectInterval: time.Duration(1e9 * seconds),
		logger:          log.GetLogger("collector", log.RotateModeMonth),
		collectors:      make([]ICollector, 0, size),
		BaseRegistry:    r,
	}
}

type CollectorManager struct {
	exit            chan bool
	collectInterval time.Duration
	logger          *log.Vlogger
	collectors      []ICollector
	BaseRegistry    metrics.Registry
}

type ICollector interface {
	GetPrefix() string
	Collect()
}

func (cm *CollectorManager) RegisterCollector(c ICollector) {
	cm.logger.Printf("RegisterCollector %s", c.GetPrefix())
	cm.collectors = append(cm.collectors, c)
}

func (cm *CollectorManager) RegisterCollectors(cs... ICollector) {
	for _, c := range cs {
		cm.logger.Printf("RegisterCollector %s", c.GetPrefix())
		cm.collectors = append(cm.collectors, c)
	}
	
}

func (cm *CollectorManager) run(shutdown chan bool, interval time.Duration) {
	defer close(cm.exit)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	cm.logger.Println("CollectorManager started")
	for {
		select {
		case <-shutdown:
			cm.logger.Println("CollectorManager stoped")
			return
		case <-ticker.C:
			for _, c := range cm.collectors {
				// go func() {
					c.Collect()
				//}()
			}
		}
	}
}

func (cm *CollectorManager) Start() {
	cm.logger.Println("CollectorManager starting")
	go cm.run(cm.exit, cm.collectInterval)
}

func (cm *CollectorManager) Stop() {
	cm.logger.Println("CollectorManager stoping")
	cm.exit <- true
}
