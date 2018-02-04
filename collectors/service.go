package collectors

import (
	"log"
	
	"github.com/prometheus/client_golang/prometheus"
)

type ServiceCollector struct {
	Status		*prometheus.GaugeVec
}

func NewServiceCollector()*ServiceCollector{
	labels := make(prometheus.Labels)

	return &ServiceCollector{
		Status: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "default",
				Name:        "fast_service_status",
				Help:        "TEST FOR SERVICE STATUS",
				ConstLabels: labels,
			},
			[]string{"service"},
		),
	}
}

func (s *ServiceCollector) collectorList() []prometheus.Collector {
	return []prometheus.Collector{
		s.Status,
	}
}

func (s *ServiceCollector)collect()error{
	var status float64
	status = 1
	s.Status.WithLabelValues("node1234").Set(status)
	
	return nil
}

func (s *ServiceCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range s.collectorList() {
		metric.Describe(ch)
	}
}

func (s *ServiceCollector) Collect(ch chan<- prometheus.Metric) {
	if err := s.collect(); err != nil {
		log.Println("failed collecting service metrics:", err)
	}

	for _, metric := range s.collectorList() {
		metric.Collect(ch)
	}
}

