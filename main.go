package main


import (
	"sync"
	
	"github.com/coder-van/v-collect/collector/system"
	"github.com/coder-van/v-collect/collector"
	"github.com/coder-van/v-stats"
	"time"
	"fmt"
	"github.com/coder-van/v-stats/metrics"
	"github.com/coder-van/v-util/log"
)

func NewAgent()  *agent{
	
	conf, err := LoadConfig("")
	
	if err != nil {
		fmt.Printf("Load config failed %s \n", err)
		return nil
	}
	if conf.Debug {
		log.Debug = true
	}
	
	log.SetLogDir(conf.LoggingConfig.LogDir)
	
	prefix := fmt.Sprintf("%s.%s.", conf.HostConfig.HostGroup, conf.HostConfig.HostSign)
	r := metrics.NewPrefixedChildRegistry(metrics.GlobRegistry, prefix)
	
	sd := statsd.NewStatsD()
	sd.SetConfig(conf.StatsdConfig)
	sd.SetRegistry(r)
	
	return &agent{
		sd : sd,
		cm: collector.NewCollectorManager(1, r),
	}
}

type agent struct {
	cm *collector.CollectorManager
	sd *statsd.StatsD
}


func main() {
	//i, err := cpu.Counts(false)
	//fmt.Println(i, err)
	
	// config, err := util.NewConfig("")
	// fmt.Println(config, err)
	a := NewAgent()
	
	if a == nil {
		return
	}
	
	sysCollector := system.NewSysCollector()
	a.cm.RegisterCollector(sysCollector)
	
	var wg sync.WaitGroup
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.cm.Run(time.Second)
	}()
	
	shutdown := make(chan struct{})
	a.sd.Run(shutdown, 5*time.Second)
	
	wg.Wait()
}
