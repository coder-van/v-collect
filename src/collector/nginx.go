/*
 exe curl http://localhost/nginx_status/ get :
 
 Active connections: 1
 server accepts handled requests
  4959543 4959543 4958930
 Reading: 0 Writing: 1 Waiting: 0
 
 detail in http://nginx.org/en/docs/http/ngx_http_stub_status_module.html
 */
package collector

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder-van/v-stats/metrics"
)

const (
	// time limit seconds for requests made
	// http://docscn.studygolang.com/pkg/net/http/#Client
	RequestMadeTimeOutSec = 3
	
	// time limit seconds to wait for a server's response headers return
	// http://docscn.studygolang.com/pkg/net/http/#Transport
	ResponseTimeoutSec  = 5
)

type NginxConfig struct {
	Enable     bool    `toml:"enable"`
	Url        string  `toml:"url"`
}

// NewNginx XXX
func NewNginx(registry metrics.Registry, conf NginxConfig) *Nginx {
	bc := metrics.NewBaseStat("nginx", registry)
	return &Nginx{
		BaseStat: bc,
		StatusURL: conf.Url,
	}
}

func (n *Nginx) GetPrefix() string {
	return n.Prefix
}

// Nginx XXX
type Nginx struct {
	*metrics.BaseStat
	StatusURL string
}

var tr = &http.Transport{
	ResponseHeaderTimeout: time.Duration(ResponseTimeoutSec * time.Second),
}

var client = &http.Client{
	Transport: tr,
	Timeout:   time.Duration(RequestMadeTimeOutSec * time.Second),
}

func (n *Nginx) Register() {
	
	gItems := []string{"active", "reading", "writing", "waiting"}
	for _, k := range gItems {
		m := n.GetMemMetric(k)
		n.Registry.Register(m, metrics.NewGauge())
	}
	
	cItems := []string{"accepts", "handled", "requests"}
	for _, k := range cItems {
		m:= n.GetMemMetric(k)
		n.Registry.Register(m, metrics.NewCounter())
	}
}


// Check XXX
func (n *Nginx) Collect() {
	addr, err := url.Parse(n.StatusURL)
	if err != nil {
		n.OnErr("error_parse_address",
			fmt.Errorf("Error parse address '%s': %s", n.StatusURL, err))
	}

	resp, err := client.Get(addr.String())
	if err != nil {
		n.OnErr("error_making_request",
			 fmt.Errorf("error making HTTP request to %s: %s", addr.String(), err))
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		n.OnErr("error_response",
			fmt.Errorf("response from %s : %s", addr.String(), resp.Status))
	}
	r := bufio.NewReader(resp.Body)

	// Active connections
	buf := make([]byte, 256)
	_, err = r.Read(buf)
	if err != nil{
		n.OnErr("error_read_string", err)
		return
	}
	
	lines := strings.Split(string(buf), "\n")
	if len(lines) < 4 {
		n.OnErr("error_read_lines", fmt.Errorf("error_read_lines"))
		return
	}
	
	active   := strings.TrimSpace(strings.Split(lines[0], ":")[1])
	
	line2 := strings.TrimSpace(lines[2])
	accepts  :=strings.TrimSpace(strings.Split(line2, " ")[0])
	handled  :=strings.TrimSpace(strings.Split(line2, " ")[1])
	requests :=strings.TrimSpace(strings.Split(line2, " ")[2])
	
	data := strings.Fields(lines[3])
	reading := data[1]
	writing := data[3]
	waiting := data[5]
	
	n.GaugeUpdate("active", active)
	n.GaugeUpdate("reading", reading)
	n.GaugeUpdate("writing", writing)
	n.GaugeUpdate("waiting", waiting)
	
	n.CounterIncTotal("accepts", accepts)
	n.CounterIncTotal("handled", handled)
	n.CounterIncTotal("requests", requests)
}


