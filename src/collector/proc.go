package collector

import (
	"github.com/coder-van/v-stats/metrics"
	"bytes"
	"os/exec"
	"strings"
	"sync"
)

type ProcConfig struct {
	Enable     bool    `toml:"enable"`
	ProcNames  string  `toml:"process_names"`
}

func NewProcCollector(registry metrics.Registry, conf ProcConfig) *ProcCollector {
	bc := metrics.NewBaseStat("proc", registry)
	pns := make(map[string]int64)
	procs := strings.Split(conf.ProcNames, ",")
	for _, p := range procs {
		pn := strings.Trim(p, " ")
		pns[pn] = 0
	}
	return &ProcCollector{
		BaseStat: bc,
		Conf:          conf,
		procNames:     pns,
	}
}

type ProcCollector struct {
	sync.Mutex
	*metrics.BaseStat
	Conf      ProcConfig
	procNames map[string]int64
}

func (p *ProcCollector) GetPrefix() string {
	return p.Prefix
}

func (p *ProcCollector) Collect()  {
	if len(p.procNames) == 0 {
		return
	}
	p.Lock()
	defer p.Unlock()
	str , err := execRun("ps", "-ef")
	if err == nil {
		lines := strings.Split(str, "\n")
		for _, line := range lines {
			for pn := range p.procNames {
				if strings.Index(line, pn) > 0 {
					p.procNames[pn] += 1
				}
			}
		}
		for n, v := range p.procNames {
			p.GaugeUpdate(n, v)
			p.procNames[n] = 0
		}
	}else{
		p.OnErr("error-proc-collect", err)
	}
}

func execRun(cmd string, args ...string) (string, error) {
	ecmd := exec.Command(cmd, args...)
	bs, err := ecmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	
	return string(bytes.TrimSpace(bs)), nil
}