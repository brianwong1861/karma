package alertmanager

import "github.com/prometheus/client_golang/prometheus"

type karmaCollector struct {
	collectedAlerts *prometheus.Desc
	collectedGroups *prometheus.Desc
	cyclesTotal     *prometheus.Desc
	errorsTotal     *prometheus.Desc
}

func newKarmaCollector() *karmaCollector {
	return &karmaCollector{
		collectedAlerts: prometheus.NewDesc(
			"karma_collected_alerts_count",
			"Total number of alerts collected from Alertmanager API",
			[]string{"alertmanager", "state", "receiver"},
			prometheus.Labels{},
		),
		collectedGroups: prometheus.NewDesc(
			"karma_collected_groups_count",
			"Total number of alert groups collected from Alertmanager API",
			[]string{"alertmanager", "receiver"},
			prometheus.Labels{},
		),
		cyclesTotal: prometheus.NewDesc(
			"karma_collect_cycles_total",
			"Total number of alert collection cycles run",
			[]string{"alertmanager"},
			prometheus.Labels{},
		),
		errorsTotal: prometheus.NewDesc(
			"karma_alertmanager_errors_total",
			"Total number of errors encounter when requesting data from Alertmanager API",
			[]string{"alertmanager", "endpoint"},
			prometheus.Labels{},
		),
	}
}

func (c *karmaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.collectedAlerts
	ch <- c.collectedGroups
	ch <- c.cyclesTotal
	ch <- c.errorsTotal
}

func (c *karmaCollector) Collect(ch chan<- prometheus.Metric) {
	upstreams := GetAlertmanagers()

	for _, am := range upstreams {

		ch <- prometheus.MustNewConstMetric(
			c.cyclesTotal,
			prometheus.CounterValue,
			am.metrics.cycles,
			am.Name,
		)
		for key, val := range am.metrics.errors {
			ch <- prometheus.MustNewConstMetric(
				c.errorsTotal,
				prometheus.CounterValue,
				val,
				am.Name,
				key,
			)
		}

		// receiver name -> count
		groupsByReceiver := map[string]float64{}
		// receiver name -> state -> count
		alertsByReceiverByState := map[string]map[string]float64{}

		// iterate all alert groups this instance stores
		for _, group := range am.Alerts() {
			// count all groups per receiver
			if _, found := groupsByReceiver[group.Receiver]; !found {
				groupsByReceiver[group.Receiver] = 0
			}
			groupsByReceiver[group.Receiver]++

			// count all alerts per receiver & state
			for _, alert := range group.Alerts {
				if _, found := alertsByReceiverByState[alert.Receiver]; !found {
					alertsByReceiverByState[alert.Receiver] = map[string]float64{}
				}
				if _, found := alertsByReceiverByState[alert.Receiver][alert.State]; !found {
					alertsByReceiverByState[alert.Receiver][alert.State] = 0
				}
				alertsByReceiverByState[alert.Receiver][alert.State]++
			}
		}

		// publish metrics using calculated values
		for reciver, count := range groupsByReceiver {
			ch <- prometheus.MustNewConstMetric(
				c.collectedGroups,
				prometheus.GaugeValue,
				count,
				am.Name,
				reciver,
			)
		}
		for reciver, byState := range alertsByReceiverByState {
			for state, count := range byState {
				ch <- prometheus.MustNewConstMetric(
					c.collectedAlerts,
					prometheus.GaugeValue,
					count,
					am.Name,
					state,
					reciver,
				)
			}
		}
	}
}

func init() {
	prometheus.MustRegister(newKarmaCollector())
}
