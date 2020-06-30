package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo/edgeapp"

	"github.com/tskaard/sensibo/utils"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/tskaard/sensibo/handler"
	"github.com/tskaard/sensibo/model"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// SetupLog comment
func SetupLog(logfile string, level string, logFormat string) {
	if logFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"})
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"})
	}

	logLevel, err := log.ParseLevel(level)
	if err == nil {
		log.SetLevel(logLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	if logfile != "" {
		l := lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    5, // megabytes
			MaxBackups: 2,
		}
		log.SetOutput(&l)
	}

}

func main() {
	var workDir string
	flag.StringVar(&workDir, "c", "", "Work dir")
	flag.Parse()
	if workDir == "" {
		workDir = "./"
	} else {
		fmt.Println("Work dir", workDir)
	}
	appLifecycle := edgeapp.NewAppLifecycle()

	configs := model.NewConfigs(workDir)
	states := model.NewStates(workDir)
	err := configs.LoadFromFile()
	if err != nil {
		appLifecycle.SetAppState(edgeapp.AppStateStartupError, nil)
		fmt.Print(err)
		panic("Can't load config file")
	}
	err = states.LoadFromFile()
	if err != nil {
		appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
		log.Debug("Not able to load state")
		log.Debug(err)
	}
	utils.SetupLog(configs.LogFile, configs.LogLevel, configs.LogFormat)
	log.Info("--------------Starting sensibo----------------")
	appLifecycle.SetAppState(edgeapp.AppStateStarting, nil)
	appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
	appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
	appLifecycle.SetConnectionState(edgeapp.ConnStateDisconnected)

	mqtt := fimpgo.NewMqttTransport(configs.MqttServerURI, configs.MqttClientIdPrefix, configs.MqttUsername, configs.MqttPassword, true, 1, 1)
	err = mqtt.Start()

	if err != nil {
		log.Error("Can't connect to broker. Error:", err.Error())
	} else {
		log.Info("--------------Connected----------------")
	}
	defer mqtt.Stop()
	appLifecycle.SetAppState(edgeapp.AppStateRunning, nil)

	if err := edgeapp.NewSystemCheck().WaitForInternet(5 * time.Minute); err == nil {
		log.Info("<main> Internet connection - OK")
	} else {
		log.Error("<main> Internet connection - ERROR")
	}
	if states.IsConfigured() && err == nil {
		log.Debug(states.APIkey)
		appLifecycle.SetConfigState(edgeapp.ConfigStateConfigured)
		appLifecycle.SetConnectionState(edgeapp.ConnStateConnected)
		appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated)
	} else {
		appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
		appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
	}

	fimpHandler := handler.NewFimpSensiboHandler(mqtt, configs.StateDir, appLifecycle)
	fimpHandler.Start(configs.PollTimeSec)
	log.Info("--------------Started handler----------")

	mqtt.Subscribe("pt:j1/mt:cmd/rt:ad/rn:sensibo/ad:1")
	mqtt.Subscribe("pt:j1/mt:cmd/rt:dev/rn:sensibo/ad:1/#")
	mqtt.Subscribe("pt:j1/mt:cmd/rt:ad/rn:zigbee/ad:1")
	log.Info("Subscribing to topic: pt:j1/mt:cmd/rt:ad/rn:sensibo/ad:1")
	log.Info("Subscribing to topic: pt:j1/mt:cmd/rt:dev/rn:sensibo/ad:1/#")

	select {}

	mqtt.Stop()

}
