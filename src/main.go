package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/coder-van/v-collect/src/collector"
	"github.com/coder-van/v-stats"
	"github.com/coder-van/v-stats/metrics"
	"github.com/coder-van/v-util/log"
)

func NewAgent() *agent {

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

	sd := statsd.NewStatsD(&conf.StatsdConfig)
	sd.SetRegistry(r)

	a := &agent{
		sd: sd,
		cm: collector.NewCollectorManager(conf.CollectSeconds, 1, r),
	}
	
	sysCollector := collector.NewSysCollector(r)
	a.cm.RegisterCollector(sysCollector)
	
	if conf.CollectorConf.NginxConf.Enable {
		nginxCollector := collector.NewNginx(r, conf.CollectorConf.NginxConf)
		a.cm.RegisterCollectors(nginxCollector)
	}
	if conf.CollectorConf.ProcConf.Enable {
		procCollector := collector.NewProcCollector(r, conf.CollectorConf.ProcConf)
		a.cm.RegisterCollector(procCollector)
	}
	return a
}

type agent struct {
	cm *collector.CollectorManager
	sd *statsd.StatsD
}

func main() {
	agent := NewAgent()

	signCh := make(chan os.Signal, 1)
	exitCh := make(chan bool, 1)

	signal.Notify(signCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signCh
		fmt.Println(sig)
		exitCh <- true
		agent.sd.StopAll()
		agent.cm.Stop()
	}()

	agent.sd.StartAll()
	agent.cm.Start()

	<-exitCh
	time.Sleep(time.Second)
}
