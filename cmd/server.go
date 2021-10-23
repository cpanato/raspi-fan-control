package cmd

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/stianeikeland/go-rpio/v4"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var (
	tempGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "raspeberry_temp",
		Help: "The Raspebeery Pie temperature in celsius",
	})

	fanGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "raspeberry_external_fan",
		Help: "If the external Fan is active or not",
	})

	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start the fan control server",
		Run: func(cmd *cobra.Command, args []string) {
			startServer()
		},
	}
)

func startServer() {
	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
<head><title>Fan Control for RaspberryPies</title></head>
<body>
<h1>Fan Control for RaspberryPies</h1>
</body>
</html>
`))
	})
	r.HandleFunc("/healthz", ping).Methods(http.MethodGet)
	r.Path("/metrics").Handler(promhttp.Handler())

	server := &http.Server{
		Addr:         ":9001",
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	v := VersionInfo()
	log.Println(v.String())

	log.Printf("Listening on %s\n", ":9001")
	go func() {
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		log.Printf("Server exited with error %s\n", err.Error())
		os.Exit(1)
	}()

	err := handleFan()
	if err != nil {
		log.Printf("Server exited with error %s\n", err.Error())
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-sig
}

func ping(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte(`{"status": "ok"}`))
	if err != nil {
		log.Printf("Failed to write ping %s\n", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func handleFan() error {
	pin := rpio.Pin(pinPort)
	err := rpio.Open()
	if err != nil {
		return err
	}
	defer rpio.Close()
	pin.Output()

	outputChannel := make(chan error)
	go func() {
		for {
			temp, err := readTemp()
			if err != nil {
				outputChannel <- err
			}

			tempGauge.Set(temp)

			if temp >= 50 {
				log.Printf("Temp is above the threshold. Actual temperature: %f, turning on the fan\n", temp)
				pin.High()
				fanGauge.Set(1)
			} else {
				log.Printf("Temp is under the threshold. Actual temperature: %f, turning off the fan\n", temp)
				pin.Low()
				fanGauge.Set(0)
			}

			log.Println("Sleeping 30 seconds")
			time.Sleep(15 * time.Second)
		}
	}()

	return <-outputChannel
}

func readTemp() (float64, error) {
	file, err := os.Open("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return -9999, err
	}
	defer file.Close()

	dat, err := ioutil.ReadAll(file)
	if err != nil {
		return -9999, err
	}

	tempParsed, err := strconv.ParseFloat(strings.TrimSuffix(string(dat), "\n"), 64)
	if err != nil {
		return -9999, err
	}

	tempCelsius := tempParsed / 1000.0
	log.Printf("Temperature %f celsius\n", tempCelsius)

	return tempCelsius, nil
}
