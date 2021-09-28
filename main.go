package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "The number of received http request",
		},
		[]string{"handler", "method"},
	)
	loadGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "load_total",
			Help: "The load of the server",
		},
	)
	errorCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "The total number of errors",
		},
	)

	listenAddr string
	display    string

	ma = map[int]string{
		0:    "zero\t",
		1:    "one\t",
		10:   "ten\t",
		100:  "hundred",
		1000: "thousand",
		1e6:  "million",
		1e9:  "billion",
		//		1e12: "trillion",
	}
	help string
)

func init() {
	flag.StringVar(&listenAddr, "listen-addr", ":9090", "The addres the server will listen to.")
	flag.StringVar(&display, "display", "nothing", "The message the server will respond with.")

	v := make([]int, len(ma))
	i := 0
	for key := range ma {
		v[i] = key
		i++
	}
	sort.Ints(v)
	help = `Following endpoints are available (zero will reset the counter):
endpoint	weigth
`
	for _, i := range v {
		help = fmt.Sprintf("%s/%s\t%d\n", help, ma[i], i)
	}
	help = fmt.Sprintf("%s/%s\t%d\n", help, "metrics", 0)

}

func metricsMiddleWare(path string, weight int, next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestCounter.With(prometheus.Labels{"method": r.Method, "handler": path}).Inc()
		if weight == 0 {
			loadGauge.Set(0)
		} else if weight > 0 {
			loadGauge.Add(float64(weight))
		}
		next(w, r)
	}
}

func helper(str string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(str + "\n"))
		if err != nil {
			errorCounter.Inc()
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("hi"))
	if err != nil {
		errorCounter.Inc()
	}
}

func main() {
	flag.Parse()

	r := prometheus.NewRegistry()
	r.MustRegister(
		errorCounter,
		requestCounter,
		loadGauge,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	for k, v := range ma {
		s := "/" + strings.Trim(v, "\t")
		m.HandleFunc(s, metricsMiddleWare(s, k, helper(display)))
	}
	m.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
		return
	})
	m.HandleFunc("/", metricsMiddleWare("/", -1, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(help))
		return
	}))
	log.Fatal(http.ListenAndServe(listenAddr, m))
}
