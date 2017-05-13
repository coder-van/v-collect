package system

import (
	
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"strings"
	"os"
	"github.com/coder-van/v-stats/metrics"
	"fmt"
)

/*
收集系统各指标  top 命令可以获取的内容
top 命令获取内容
	Processes: 331 total, 2 running, 10 stuck, 319 sleeping, 1957 threads                              10:44:12
	Load Avg: 2.31, 2.15, 2.02
	CPU usage: 1.1% user, 12.12% sys, 86.86% idle
	SharedLibs: 90M resident, 0B data, 6796K linkedit.
	MemRegions: 147134 total, 6175M resident, 99M private, 1992M shared.
	PhysMem: 16G used (2354M wired), 103M unused.
	VM: 927G vsize, 1351M framework vsize, 38326173(0) swapins, 40034566(0) swapouts.
	Networks: packets: 9934944/6545M in, 9567092/1901M out.
	Disks: 2593558/193G read, 3261477/244G written.

	PID    COMMAND      %CPU TIME     #TH   #WQ  #PORT MEM    PURG   CMPRS  PGRP  PPID  STATE    BOOSTS      %CPU_ME %CPU_OTHRS UID
	99345  Google Chrom 0.0  03:36.61 10    0    106+  30M+   0B     71M+   251   251   sleeping *0[69+]     0.00000 0.00000    501

此插件收集的内容
	1. upTime 系统启动时长
	2. sys.load
	3. cpu 使用率
	4. mem 内存
	5. net 网络
	6. diskIo 硬盘读写
	
	另外包含硬盘使用情况
 */

// KB, MB, GB, TB, PB...human friendly
const (
	_          = iota  // ignore first value by assigning to blank identifier
	KB float64 = 1 << (10 * iota)   // 1024
	MB
	GB
	TB
	PB
	Prefix     = "system."
)

func NewSysCollector() *SysCollector{
	var perCPU, totalCPU bool = true, true
	
	return &SysCollector{
		Prefix: Prefix,
		cpu: &CPUStats{
			PerCPU:   perCPU,
			TotalCPU: totalCPU,
		},
		io: &DiskIOStats{},
		net: &NetStats{},
		diskUsage: &DiskStats{[]string{}, []string{}},
	}
}

type SysCollector struct {
	Prefix      string
	cpu       *CPUStats
	io        *DiskIOStats
	net       *NetStats
	diskUsage *DiskStats
}


type CPUStats struct {
	lastStats []cpu.TimesStat
	PerCPU    bool
	TotalCPU  bool
}

type DiskIOStats struct {
	lastIOStats        map[string]disk.IOCountersStat
	lastCollectionTime int64
	Devices            []string
}

type DiskStats struct {
	MountPoints []string
	IgnoreFS    []string
}


// metric
var (
	Uptime         metrics.Gauge
	Load_1         metrics.Gauge
	Load_5         metrics.Gauge
	Load_15        metrics.Gauge
	Cpu            map[string]metrics.Gauge
	Mem            map[string]metrics.GaugeFloat64
	Swap           map[string]metrics.Gauge
	Net            map[string]metrics.Counter
	NetProtos	   map[string]metrics.Gauge
	Disk	       map[string]metrics.Gauge
)


type NetStats struct {
	lastIOStats    []net.IOCountersStat
}

func (s *SysCollector) GetPrefix() string{
	return s.Prefix
}

func (s *SysCollector) Register(r metrics.Registry)  {
	Uptime = metrics.NewGauge()
	Load_1 = metrics.NewGauge()
	Load_5 = metrics.NewGauge()
	Load_15 = metrics.NewGauge()

	r.Register("uptime", Uptime)
	r.Register("load.1", Load_1)
	r.Register("load.5", Load_5)
	r.Register("load.15", Load_15)

	s.RegisterCpu(r)
	
	s.RegisterMem(r)
	s.RegisterMemSwap(r)

	s.RegisterNetIO(r)
	s.RegisterNetProto(r)

	s.RegisterDiskUsage(r)
}


func (s *SysCollector) Collect()  error{
	
	if err := s.collectSysUpTime(); err != nil {
		return err
	}

	if err := s.collectSysLoad(); err != nil {
		return err
	}

	if err := s.collectSysCPU(); err != nil {
		return err
	}
	
	if err := s.collectMem(); err != nil {
		return err
	}
	
	if err := s.collectMemSwap(); err != nil {
		return err
	}

	if err := s.collectSysNet(); err != nil {
		return err
	}

	if err := s.collectNetProto(); err != nil {
		return err
	}

	if err := s.collectDiskUsage(); err != nil {
		return err
	}
	return nil
}

func (s *SysCollector) collectSysUpTime() error {
	hostInfo, err := host.Info()
	if err != nil {
		return err
	}
	Uptime.Update(int64(hostInfo.Uptime))
	return nil
}

func (s *SysCollector) collectSysLoad() error {
	/*
	 data like:
		type: gauge
		key: system.load
		fields: map[15:1.72 1:1.73 5:1.65]
		tags: []
	 */
	loadAvg, err := load.Avg()
	if err != nil {
		return err
	}
	Load_1.Update(int64(loadAvg.Load1))
	Load_5.Update(int64(loadAvg.Load5))
	Load_15.Update(int64(loadAvg.Load15))
	return nil
}


func (s *SysCollector) RegisterCpu(r metrics.Registry) error {
	times, err := cpuTimes(s.cpu.PerCPU, s.cpu.TotalCPU)
	if err != nil {
		return err
	}
	CpuItems := []string{"user", "system", "idle", "nice", "iowait", "irq", "softirq", "stolen", "guest", "guest_nice"}
	Cpu = make(map[string]metrics.Gauge, len(times)*len(CpuItems))
	
	for _, cts := range times {
		keyPrefix := "cpu." + cts.CPU + "."
		for _, k := range CpuItems{
			g := metrics.NewGauge()
			Cpu[keyPrefix+k] = g
			r.Register(keyPrefix+k, g)
		}
	}
	
	return nil
}

func (s *SysCollector) collectSysCPU() error {
	/*
	data to handler cpu0-cpuN append cpu-total
	here is one cpu data:
		type: gauge
		key: system.cpu
		fields: map[
			user:1.8610421836228288
			idle:96.6501240694789
			iowait:0
			softirq:0 guest:0
			guest_nice:0 system:1.488833746898263
			nice:0 irq:0 stolen:0
		]
		tags: [cpu:cpu-total]
	 */
	
	times, err := cpuTimes(s.cpu.PerCPU, s.cpu.TotalCPU)
	
	if err != nil {
		return fmt.Errorf("error getting CPU info: %s", err)
	}
	
	for i, cts := range times {
		
		keyPrefix := "cpu."+cts.CPU+"."
		
		total := cpuTotalTime(cts)
		
		// Add in percentage
		if len(s.cpu.lastStats) == 0 {
			// If it's the 1st check, can't get CPU Usage stats yet
			break
		}
		lastCts := s.cpu.lastStats[i]
		lastTotal := cpuTotalTime(lastCts)
		totalDelta := total - lastTotal
		
		if totalDelta < 0 {
			s.cpu.lastStats = times
			return fmt.Errorf("Error: current total CPU time is less than previous total CPU time")
		}
		
		if totalDelta == 0 {
			continue
		}
		
		fields := map[string]float64{
			"user":       100 * (cts.User - lastCts.User) / totalDelta,
			"system":     100 * (cts.System - lastCts.System) / totalDelta,
			"idle":       100 * (cts.Idle - lastCts.Idle) / totalDelta,
			"nice":       100 * (cts.Nice - lastCts.Nice) / totalDelta,
			"iowait":     100 * (cts.Iowait - lastCts.Iowait) / totalDelta,
			"irq":        100 * (cts.Irq - lastCts.Irq) / totalDelta,
			"softirq":    100 * (cts.Softirq - lastCts.Softirq) / totalDelta,
			"stolen":     100 * (cts.Steal - lastCts.Steal) / totalDelta,
			"guest":      100 * (cts.Guest - lastCts.Guest) / totalDelta,
			"guest_nice": 100 * (cts.GuestNice - lastCts.GuestNice) / totalDelta,
		}
		for k, v := range fields {
			Cpu[keyPrefix+k].Update(int64(v))
		}
	}
	
	s.cpu.lastStats = times
	
	return nil
}


func (s *SysCollector) getMemMetric(k string) string {
	prefix := "mem"
	return fmt.Sprintf("%s.%s", prefix, k)
}

func (s *SysCollector) RegisterMem(r metrics.Registry) error {
	MemItems := []string{"total", "usable", "used", "free", "cached", "buffered", "pct_usable"}
	Mem = make(map[string]metrics.GaugeFloat64, len(MemItems))
	
	for _, k := range MemItems{
		metric := s.getMemMetric(k)
		g := metrics.NewGaugeFloat64()
		Mem[metric] = g
		r.Register(metric, g)
	}
	return nil
}

func (s *SysCollector) collectMem() error {
	/*
	data to handle
		type: gauge
		key: system.mem
		fields: map[
			used:16245.65234375
			free:138.34765625
			cached:0 buffered:0
			pct_usable:36.37368679046631
			total:16384
			usable:5959.46484375
		]
		tags: []
	 */
	vm, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error getting virtual memory info: %s", err)
	}
	
	fields := map[string]float64{
		"total":      float64(vm.Total) / MB,
		"usable":     float64(vm.Available) / MB,
		"used":       float64(vm.Total-vm.Free) / MB,
		"free":       float64(vm.Free) / MB,
		"cached":     float64(vm.Cached) / MB,
		"buffered":   float64(vm.Buffers) / MB,
		"pct_usable": 100 * float64(vm.Available) / float64(vm.Total),
	}
	
	for k, v := range fields {
		metric := s.getMemMetric(k)
		Mem[metric].Update(v)
	}
	return nil
}


func (s *SysCollector) RegisterMemSwap(r metrics.Registry) error {
	swapItems := []string{"total", "used", "free", "pct_free", "swapped_in", "swapped_out"}
	Swap = make(map[string]metrics.Gauge, len(swapItems))
	
	keyPrefix := "swap."
	for _, k := range swapItems{
		name := keyPrefix+k
		g := metrics.NewGauge()
		Swap[name] = g
		r.Register(name, g)
		
	}
	return nil
}

func (s *SysCollector) collectMemSwap() error {
	/*
		type: gauge
		key: system.swap
		fields: map[
			swapped_out:0
			total:6144
			used:4433.5
			free:1710.5
			pct_free:27.840169270833343
			swapped_in:0
		]
		tags: []
	 */
	swap, err := mem.SwapMemory()
	if err != nil {
		return fmt.Errorf("error getting swap memory info: %s", err)
	}
	
	fields := map[string]float64{
		"total":       float64(swap.Total) / MB,
		"used":        float64(swap.Used) / MB,
		"free":        float64(swap.Free) / MB,
		"pct_free":    100 - swap.UsedPercent,
		"swapped_in":  float64(swap.Sin),
		"swapped_out": float64(swap.Sout),
	}
	keyPrefix := "swap."
	for k, v := range fields {
		Swap[keyPrefix+k].Update(int64(v))
	}
	
	// agg.AddMetrics("gauge", "system.swap", fields, nil, "")
	return nil
}


func (s *SysCollector) RegisterNetIO(r metrics.Registry) error {
	netIOs, err := net.IOCounters(true)
	if err != nil {
		return err
	}
	
	NetItems := []string{"bytes_sent", "bytes_rcvd", "packets_in.count", "packets_in.error",
		"packets_out.count", "packets_out.error"}
	Net = make(map[string]metrics.Counter, len(netIOs)*len(NetItems))
	
	for _, io := range netIOs {
		keyPrefix := "net."+ io.Name+"."
		for _, k := range NetItems{
			name := keyPrefix+k
			Net[name] = metrics.NewCounter()
			r.Register(name, Net[name])
		}
	}
	return nil
}

func (s *SysCollector) collectSysNet() error {
	/* lo0 | stf0 | gif0 | en0 | en1 | en2 | bridg | p2p0 | awdl0
		type: rate
		key: system.net
		fields: map[
			packets_out.count: 4000223
			packets_out.error: 0
			bytes_sent: 1358838325
			bytes_rcvd: 1358838325
			packets_in.count: 4000223
			packets_in.error: 0
		]
		tags: [interface: lo0]
	 */
	netIOs, err := net.IOCounters(true)
	if err != nil {
		return fmt.Errorf("error getting net io info: %s", err)
	}
	
	for i, io := range netIOs {
		
		if len(s.net.lastIOStats) == 0 {
			// If it's the 1st
			break
		}
		lastIO := s.net.lastIOStats[i]
		fields := map[string]float64{
			"bytes_sent":        float64(io.BytesSent-lastIO.BytesSent),
			"bytes_rcvd":        float64(io.BytesRecv-lastIO.BytesRecv),
			"packets_in.count":  float64(io.PacketsRecv-lastIO.PacketsRecv),
			"packets_in.error":  float64(io.Errin + io.Dropin-(lastIO.Errin+lastIO.Dropin)),
			"packets_out.count": float64(io.PacketsSent-lastIO.PacketsSent),
			"packets_out.error": float64(io.Errout + io.Dropout - (lastIO.Errout+lastIO.Dropout)),
		}
		// agg.AddMetrics("rate", "system.net", fields, tags, "")
		keyPrefix := "net."+ io.Name+"."
		for k, v := range fields {
			Net[keyPrefix+k].Inc(int64(v))
		}
		
	}
	s.net.lastIOStats = netIOs
	return nil
}


func (s *SysCollector) RegisterNetProto(r metrics.Registry) error {
	protos, _ := net.ProtoCounters(nil)
	NetProtos = make(map[string]metrics.Gauge)
	
	tags := map[string]string{"interface": "all", }
	prefix := "net."
	for _, _proto := range protos {
		
		for stat:= range _proto.Stats {
			name := fmt.Sprintf("%s%s_%s", prefix, strings.ToLower(_proto.Protocol), strings.ToLower(stat))
			key := metrics.MakeMetric(name, tags)
			NetProtos[key] = metrics.NewGauge()
		}
	}
	return nil
}

func (s *SysCollector) collectNetProto() error {
	/* Get system wide stats for different network protocols
	     (ignore these stats if the call fails)
		
	 */
	
	netProtos, _ := net.ProtoCounters(nil)
	tags := map[string]string{"interface": "all", }
	prefix := "net."
	for _, _proto := range netProtos {
		
		for stat, value := range _proto.Stats {
			name := fmt.Sprintf("%s%s_%s", prefix, strings.ToLower(_proto.Protocol), strings.ToLower(stat))
			key := metrics.MakeMetric(name, tags)
			NetProtos[key].Update(int64(value))
		}
	}
	return nil
}


func (s *SysCollector) RegisterDiskUsage(r metrics.Registry) error {
	disks, err := diskUsage(s.diskUsage.MountPoints, s.diskUsage.IgnoreFS)
	if err != nil {
		return err
	}
	// fmt.Println(disks)
	
	DiskItems := []string{"disk.total", "disk.free", "disk.used", "disk.in_use",
						  "fs.inodes.total", "fs.inodes.free", "fs.inodes.used", "fs.inodes.in_use"}
	Disk = make(map[string]metrics.Gauge, len(disks)*len(DiskItems))
	
	for _, du := range disks {
		tags := map[string]string{"path": du.Path, "fstype": du.Fstype}
		
		for _, k := range DiskItems {
			key := metrics.MakeMetric(k, tags)
			g := metrics.NewGauge()
			Disk[key] = g
			r.Register(key, g)
		}
	}
	return nil
}

func (s *SysCollector) collectDiskUsage() error {
	/*  [path:/ fstype:hfs]  [path:/dev fstype:devfs] [path:/Volumes/Gogland 1.0 EAP fstype:hfs]
		type: gauge
		key: system.disk
		fields: map[
			disk.total:2.43924992e+08
			disk.free:1.5257208e+08
			disk.used:9.1096912e+07
			disk.in_use:0.37385516824397586
			fs.inodes.total:60981246
			fs.inodes.free:38143020
			fs.inodes.used:22838226
			fs.inodes.in_use:0.37451228858131236
		]
		tags: [path:/ fstype:hfs]

	 */
	disks, err := diskUsage(s.diskUsage.MountPoints, s.diskUsage.IgnoreFS)
	if err != nil {
		return fmt.Errorf("error getting disk usage info: %s", err)
	}
	// fmt.Println(disks)
	for _, du := range disks {
		if du.Total == 0 {
			// Skip dummy filesystem (procfs, cgroupfs, ...)
			continue
		}
		
		var usedPercent float64
		if du.Used+du.Free > 0 {
			usedPercent = float64(du.Used) / (float64(du.Used) + float64(du.Free))
		}
		
		fields := map[string]float64{
			"disk.total":       float64(du.Total) / KB,
			"disk.free":        float64(du.Free) / KB,
			"disk.used":        float64(du.Used) / KB,
			"disk.in_use":      usedPercent,
			"fs.inodes.total":  float64(du.InodesTotal),
			"fs.inodes.free":   float64(du.InodesFree),
			"fs.inodes.used":   float64(du.InodesUsed),
			"fs.inodes.in_use": du.InodesUsedPercent / 100,
		}
		tags := map[string]string{"path": du.Path, "fstype": du.Fstype }
		
		for k, v := range fields{
			key := metrics.MakeMetric(k, tags)
			if d, ok := Disk[key]; !ok{
				panic(key)
			}else{
				d.Update(int64(v))
			}
			
		}
	}
	
	return nil
}


// 以下是对一些获取数据操作进行封装

func cpuTotalTime(t cpu.TimesStat) float64 {
	total := t.User + t.System + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal +
		t.Guest + t.GuestNice + t.Idle
	return total
}

func cpuTimes(perCPU, totalCPU bool) ([]cpu.TimesStat, error) {
	var cpuTimes []cpu.TimesStat
	if perCPU {
		if perCPUTimes, err := cpu.Times(true); err == nil {
			cpuTimes = append(cpuTimes, perCPUTimes...)
		} else {
			return nil, err
		}
	}
	if totalCPU {
		if totalCPUTimes, err := cpu.Times(false); err == nil {
			cpuTimes = append(cpuTimes, totalCPUTimes...)
		} else {
			return nil, err
		}
	}
	return cpuTimes, nil
}

func diskUsage(mountPointFilter []string, fsTypeExclude []string, ) ([]*disk.UsageStat, error) {
	parts, err := disk.Partitions(true)
	if err != nil {
		return nil, err
	}
	
	// Make a "set" out of the filter slice
	mountPointFilterSet := make(map[string]bool)
	for _, filter := range mountPointFilter {
		mountPointFilterSet[filter] = true
	}
	fsTypeExcludeSet := make(map[string]bool)
	for _, filter := range fsTypeExclude {
		fsTypeExcludeSet[filter] = true
	}
	
	var usage []*disk.UsageStat
	
	for _, p := range parts {
		if len(mountPointFilter) > 0 {
			// If the mount point is not a member of the filter set,
			// don't gather info on it.
			_, ok := mountPointFilterSet[p.Mountpoint]
			if !ok {
				continue
			}
		}
		mountPoint := os.Getenv("HOST_MOUNT_PREFIX") + p.Mountpoint
		if _, err := os.Stat(mountPoint); err == nil {
			du, err := disk.Usage(mountPoint)
			if err != nil {
				return nil, err
			}
			du.Path = p.Mountpoint
			// If the mount point is a member of the exclude set,
			// don't gather info on it.
			_, ok := fsTypeExcludeSet[p.Fstype]
			if ok {
				continue
			}
			du.Fstype = p.Fstype
			usage = append(usage, du)
		}
	}
	
	return usage, nil
}



//func diskUsage(mountPointFilter []string, fsTypeExclude []string, ) ([]*disk.UsageStat, error) {
//	parts, err := disk.Partitions(true)
//	if err != nil {
//		return nil, err
//	}
//
//	// Make a "set" out of the filter slice
//	mountPointFilterSet := make(map[string]bool)
//	for _, filter := range mountPointFilter {
//		mountPointFilterSet[filter] = true
//	}
//	fsTypeExcludeSet := make(map[string]bool)
//	for _, filter := range fsTypeExclude {
//		fsTypeExcludeSet[filter] = true
//	}
//
//	var usage []*disk.UsageStat
//
//	for _, p := range parts {
//		if len(mountPointFilter) > 0 {
//			// If the mount point is not a member of the filter set, don't gather info on it.
//			_, ok := mountPointFilterSet[p.Mountpoint]
//			if !ok {
//				continue
//			}
//		}
//		mountPoint := os.Getenv("HOST_MOUNT_PREFIX") + p.Mountpoint
//		if _, err := os.Stat(mountPoint); err == nil {
//			du, err := disk.Usage(mountPoint)
//			if err != nil {
//				return nil, err
//			}
//			du.Path = p.Mountpoint
//			// If the mount point is a member of the exclude set,
//			// don't gather info on it.
//			_, ok := fsTypeExcludeSet[p.Fstype]
//			if ok {
//				continue
//			}
//			du.Fstype = p.Fstype
//			usage = append(usage, du)
//		}
//	}
//
//	return usage, nil
//}

