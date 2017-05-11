package collector

import (
	"time"
	"github.com/coder-van/v-stats/metrics"
	"github.com/coder-van/v-util/log"
)

func NewCollectorManager(length int, r metrics.Registry) *CollectorManager{
	return &CollectorManager{
		logger: log.GetLogger("collector", log.RotateModeMonth),
		collectors: make([]ICollector, 0, length),
		baseRegistry: r,
	}
}

type CollectorManager struct {
	logger *log.Vlogger
	collectors []ICollector
	baseRegistry metrics.Registry
}

type ICollector interface {
	GetPrefix() string
	Collect() error
	Register(r metrics.Registry)
}

func (cm *CollectorManager) RegisterCollector(c ICollector)  {
	cm.logger.Printf("RegisterCollector %s", c.GetPrefix())
	r := metrics.NewPrefixedChildRegistry(cm.baseRegistry, c.GetPrefix())
	c.Register(r)
	cm.collectors = append(cm.collectors, c)
}

func (cm *CollectorManager) Run(interval time.Duration)  {
	cm.logger.Printf("Run CollectorManager, duration: %d", interval)
	for {
		time.Sleep(interval)
		for _, c := range cm.collectors{
			go func() {
				c.Collect()
			}()
		}
	}
}