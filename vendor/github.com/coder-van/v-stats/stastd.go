package statsd

import (
	"net"
	"sync"
	"fmt"
	"time"
	
	"github.com/coder-van/v-stats/metrics"
	"github.com/coder-van/v-stats/backends"
)

const (
	//  means UDP packet limit, see https://en.wikipedia.org/wiki/User_Datagram_Protocol#Packet_structure
	UDPMaxPacketSize int = 64 * 1024
	
	// number of UDP messages allowed to queue up, once filled, the statsd server will start dropping packets
	AllowedPendingMessages = 10000
)

func NewStatsD() *StatsD {
	b := backends.NewBackendManger()
	
	return &StatsD{
		config: NewConfig(),
		PacketInChannel: make(chan []byte, AllowedPendingMessages),
		datapointChannel: make(chan metrics.MetricDataPoint, 10000),
		backendManger:   b,
	}
}

func (s *StatsD) SetConfig(c *Config){
	s.config = c
}

func (s *StatsD) LoadConfig(fp string){
	s.config.LoadConfig(fp)
}

type StatsD struct {
	config           *Config
	backendManger    *backends.BackendManger
	metricRegistry   metrics.Registry
	PacketInChannel  chan []byte  // Channel for all incoming statsd packets
	datapointChannel chan metrics.MetricDataPoint  // channel for backends to read
	drops int  // drops tracks the number of dropped metrics.
	
}


func (s *StatsD) SetRegistry(registry metrics.Registry){
	s.metricRegistry = registry
}

func (s *StatsD) Run(shutdown chan struct{}, flashInterval time.Duration) error {
	var wg sync.WaitGroup
	// interval := 30 * time.Second
	
	wg.Add(3)
	go func() {
		// 监听UDP
		defer wg.Done()
		if !s.config.IsLocal {
			return
		}
		if err := s.listenUDP(shutdown); err != nil {
			//log.Error(err)
			fmt.Println("listenUDP err:", err)
		}
	}()
	
	go func() {
		defer wg.Done()
		s.backendManger.RegisterGraphite(s.config.GraphiteAddr)
		if err := s.backendManger.Run(shutdown, s.datapointChannel, time.Second); err != nil {
			fmt.Errorf("Reporter routine failed, exiting: %s", err.Error())
			close(shutdown)
		}
	}()
	
	go func() {
		defer wg.Done()
		if err := s.RunAggregator(shutdown, s.datapointChannel, s.metricRegistry, flashInterval); err != nil {
			fmt.Println(err)
		}
	}()
	
	
	
	wg.Wait()
	close(s.PacketInChannel)
	return nil
}

func (s *StatsD) listenUDP(shutdown chan struct{}) error {
	addr, err := net.ResolveUDPAddr("udp", s.config.ReceiverAddr)
	if err != nil {
		fmt.Println("Can't resolve address:", err)
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
	}

	fmt.Println("Statsd listening on:", addr)
	for {
		select {
		case <-shutdown:
			
			fmt.Println("Statsd server thread exit")
			if err := conn.Close(); err != nil {
				return err
			}
			return nil
		default:
			s.deliverPacketToChannelIn(conn)
		}
	}
}


func (s *StatsD) deliverPacketToChannelIn(conn *net.UDPConn) {
	/*
	 * 对UDP收到的内容给StatsD in channel
	 */
	// TODO 不用每次都初始化一个数组
	buf := make([]byte, UDPMaxPacketSize)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		fmt.Println("failed to read UDP msg because of ", err.Error())
		return
	}
	
	bufCopy := make([]byte, n)
	copy(bufCopy, buf[:n])
	
	// channel in has length limit defined in AllowedPendingMessages, if messages count out limit, drop it and count
	select {
	case s.PacketInChannel <- bufCopy:
	default:
		s.drops++
		if s.drops == 1 || s.drops%AllowedPendingMessages == 0 {
			fmt.Printf("ERROR: statsd message queue full. We have dropped %d messages so far. ", s.drops)
		}
	}
}


func (s *StatsD) RunAggregator(
	shutdown chan struct{}, ch chan metrics.MetricDataPoint, r metrics.Registry, interval time.Duration) error {
	/* parser monitors the s.in channel, if there is a packet ready, it parses the
	 packet into statsd strings and then calls parseStatsdLine, which parses a
	 single statsd metric.
	*/
	ticker := time.NewTicker(time.Second*5)
	defer ticker.Stop()
	
	agg := NewAggregator(ch, r)
	
	var packet []byte
	for {
		select {
		case <-shutdown:
			return nil
		case <-ticker.C:
			agg.Flush()
			fmt.Printf("flush. %d %s \n", len(agg.chanToB), time.Now().Local())
		case packet = <-s.PacketInChannel:
			fmt.Printf("Received packet: %s", string(packet))
			agg.HandlePackets(string(packet))
		}
	}
}