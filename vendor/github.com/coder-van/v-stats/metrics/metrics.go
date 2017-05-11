
package metrics

// UseNilMetrics is checked by the constructor functions for all of the
// standard metrics.  If it is true, the metric returned is a stub.
//
// This global kill-switch helps quantify the observer effect and makes
// for less cluttered pprof profiles.

// no use
// var UseNilMetrics bool = false

var GlobRegistry Registry = DefaultRegistry
var registryMap map[string]Registry = make(map[string]Registry, 0)

func GetRegistry(key string) Registry {
	if m, ok := registryMap[key]; ok{
		return m
	}
	return NewRegistry()
}

func SetGlobRegistry(r Registry)  {
	if GlobRegistry.Len() > 0 {
		panic("GlobRegistry metrics not empty")
	}
	GlobRegistry = r
}
