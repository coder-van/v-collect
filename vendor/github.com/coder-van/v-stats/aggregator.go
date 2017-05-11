package statsd

import (
	"strings"
	"fmt"
	"strconv"
	"time"
	
	"github.com/coder-van/v-stats/metrics"
)

/*
 数据以固定周期刷新，上次刷新据此次刷新为一个周期，在每次刷新时对不同类型数据做不同的聚合运算处理
 */
const Prefix  = ""
var Percentiles  = []float64{0.5, 0.75, 0.95, 0.99, 0.999}

func NewAggregator(ch chan metrics.MetricDataPoint, r metrics.Registry) *aggregator{
	return &aggregator{
		metricsRegister: r,
		chanToB: ch,
	}
}

type aggregator struct {
	metricsRegister        metrics.Registry
	chanToB chan           metrics.MetricDataPoint
}

func (agg *aggregator) HandlePackets(packet string) {
	/*
	 收到的 packets格式 由\n符分隔的多项数据
	 <metric>\n[metric...]
	 metric 内容如下
	 <key>:<section1>:[section2]
	 section:
	 <value>|<type>|@<rate>[|#<tag1_name=tag1_value>,[tag2_name=tag2_value]]
	 */
	_metricLines := strings.Split(packet, "\n")
	for _, metricLine := range _metricLines {
		metricLineTmp := strings.TrimSpace(metricLine)
		if metricLineTmp != "" {
			err := agg.parseMetricLine(metricLine)
			if err != nil {
				//log.Error("Error occurred when parsing packet:", err)
				continue
			}
		}
	}
}

func (agg *aggregator) parseMetricLine(packet string)  error {
	/*
	 parse metric to
	 */
	bits := strings.SplitN(packet, ":", 2)
	if len(bits) != 2 {
		//log.Infof("Error: splitting ':', Unable to parse metric: %s", packet)
		return fmt.Errorf("Error Parsing statsd packet")
	}
	
	key := bits[0]
	metadata := bits[1]
	
	data := []string{}
	sections := strings.Split(metadata, ":")
	for _, token := range sections {
		data = append(data, token)
	}
	
	agg.parseSections(key, data)
	return nil
}

func (agg *aggregator) parseSections(key string, data []string)  error {
	// 将同一个key下的数据整理成数据数组
	//_metrics := []Metric{}
	
	//for _, datum := range data {
	//	m := baseMetric{}
	//	m.Key = key
	//	// Validate splitting the bit on "|"
	//	fields := strings.Split(datum, "|")
	//	if len(fields) < 2 {
	//		return fmt.Errorf("Error parsing statsd packet")
	//	}
	//
	//	// Set allows value of strings.
	//	metric_type := fields[1]
	//	if metric_type != "s" {
	//		value, err := strconv.ParseFloat(fields[0], 64)
	//		if err != nil {
	//			return fmt.Errorf("Error parsing value from packet %s: %s", datum, err.Error())
	//		}
	//		m.Value = value
	//	}else{
	//		value, err := strconv.ParseFloat(fields[0], 64)
	//		if err != nil {
	//			return  fmt.Errorf("")
	//		}
	//		m.Value = value
	//	}
	//	for _, segment := range fields {
	//		if strings.Contains(segment, "@") && len(segment) > 1 {
	//			rate, err := strconv.ParseFloat(segment[1:], 64)
	//			if err != nil || (rate < 0 || rate > 1) {
	//				return  fmt.Errorf("Error: parsing sample rate, %s, it must be in format like: "+
	//					"@0.1, @0.5, etc. Ignoring sample rate for packet", err.Error())
	//			}
	//
	//			// sample rate successfully parsed
	//			m.Rate = rate
	//		} else if len(segment) > 0 && segment[0] == '#' {
	//			tags := strings.Split(segment[1:], ",")
	//
	//			m.Tags = make(map[string]string, len(tags))
	//			for _, t := range tags{
	//				ts := strings.Split(t, "=")
	//				name := ts[0]
	//				value := ts[1]
	//				m.Tags[name] = value
	//			}
	//		}
	//	}
	//	e :=agg.metricCreate(metric_type, m)
	//	fmt.Println(e)
	//}
	return nil
}
//func (agg *aggregator) metricCreate(metric_type string, m baseMetric) error{
//	switch metric_type {
//	case "c":
//		m.Type = TypeCounter
//		agg.counter(m)
//	case "r":
//		m.Type = "rate"
//		agg.counter(m)
//	case "g":
//		m.Type = TypeGauge
//		agg.gauge(m)
//	case "s":
//		m.Type = TypeSet
//		agg.set(m)
//	case "ms", "h":
//		m.Type = TypeTimer
//		agg.timer(m)
//	default:
//		return fmt.Errorf("Error Parsing statsd line")
//	}
//	return nil
//}


func (agg *aggregator) Flush() {
	du := float64(time.Nanosecond)
	now := time.Now().Unix()
	agg.metricsRegister.Each(func(name string, i interface{}) {
		
		switch metric := i.(type) {
		case metrics.Counter:
			agg.chanToB <- metrics.NewMetricDataPoint(name+".counter", metric.Count(), now)
		case metrics.Gauge:
			agg.chanToB <- metrics.NewMetricDataPoint(name+".value", metric.Value(), now)
		case metrics.GaugeFloat64:
			agg.chanToB <- metrics.NewMetricDataPoint(name+".value", metric.Value(), now)
		case metrics.Histogram:
			h := metric.Snapshot()
			ps := h.Percentiles(Percentiles)
			agg.chanToB <- metrics.NewMetricDataPoint(name+".count", h.Count(), now)
			agg.chanToB <- metrics.NewMetricDataPoint(name+".min", h.Min(), now)
			agg.chanToB <- metrics.NewMetricDataPoint(name+".max", h.Max(), now)
			agg.chanToB <- metrics.NewMetricDataPoint(name+".mean", h.Mean(), now)
			agg.chanToB <- metrics.NewMetricDataPoint(name+".std-dev",h.StdDev(), now)
			
			for j, key := range Percentiles {
				key := strings.Replace(strconv.FormatFloat(key*100.0, 'f', -1, 64), ".", "", 1)
				agg.chanToB <- metrics.NewMetricDataPoint(name+"-percentile"+key,  ps[j], now)
			}
		case metrics.Timer:
			t := metric.Snapshot()
			ps := t.Percentiles(Percentiles)
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.count", Prefix, name), t.Count(), now)
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.min", Prefix, name), t.Min()/int64(du), now)
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.max", Prefix, name), t.Max()/int64(du), now)
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.mean", Prefix, name), t.Mean()/du, now)
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.std-dev", Prefix, name), t.StdDev()/du, now)
			
			
			agg.chanToB <- metrics.NewMetricDataPoint(name+"count", t.Count(), now)
			
			for j, key := range Percentiles {
				key := strings.Replace(strconv.FormatFloat(key*100.0, 'f', -1, 64), ".", "", 1)
				k := fmt.Sprintf("%s.%s-percentile %.2f", Prefix, name, key)
				agg.chanToB <- metrics.NewMetricDataPoint(k, ps[j], now)
			}
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.1-minute", Prefix, name), t.Rate1(), now)
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.5-minute", Prefix, name), t.Rate5(), now)
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.15-minute", Prefix, name), t.Rate15(), now)
			agg.chanToB <- metrics.NewMetricDataPoint(fmt.Sprintf("%s.%s.mean-rate", Prefix, name), t.RateMean(), now)
		}
		
	})
	
}
