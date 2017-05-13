package main


import (
	"github.com/coder-van/v-collect/server/collector/system"
	"github.com/coder-van/v-collect/server/collector"
	"github.com/coder-van/v-stats"
	"github.com/coder-van/v-stats/metrics"
	"github.com/coder-van/v-util/log"
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"time"
)

func NewAgent()  *agent{
	
	conf, err := LoadConfig("")
	
	if err != nil {
		fmt.Println(err)
		panic("Load config failed")
		return nil
	}
	if conf.Debug {
		log.Debug = true
	}
	
	log.SetLogDir(conf.LoggingConfig.LogDir)
	
	prefix := fmt.Sprintf("%s.%s.", conf.HostConfig.HostGroup, conf.HostConfig.HostSign)
	r := metrics.NewPrefixedChildRegistry(metrics.GlobRegistry, prefix)
	
	sd := statsd.NewStatsD(nil)
	sd.SetRegistry(r)
	
	return &agent{
		sd : sd,
		cm: collector.NewCollectorManager(conf.CollectSeconds,1, r),
	}
}

type agent struct {
	cm *collector.CollectorManager
	sd *statsd.StatsD
}


func main() {
	agent := NewAgent()
	
	signCh := make(chan os.Signal,1)
	exitCh := make(chan bool, 1)
	
	signal.Notify(signCh, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		sig := <- signCh
		fmt.Println(sig)
		exitCh <- true
		agent.sd.StopAll()
		agent.cm.Stop()
	}()
	
	sysCollector := system.NewSysCollector()
	agent.cm.RegisterCollector(sysCollector)
	
	agent.sd.StartAll()
	agent.cm.Start()
	
	<- exitCh
	time.Sleep(time.Second)
}
