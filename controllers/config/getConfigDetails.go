//(C) Copyright [2022] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"time"

	constants "github.com/ODIM-Project/BMCOperator/config/constants"
	"github.com/ODIM-Project/BMCOperator/logs"
	l "github.com/ODIM-Project/BMCOperator/logs"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var podName = os.Getenv("POD_NAME")

const configFilePath = "/etc/bmc-operator-config/config.yaml"

// Data is a global variable to store the configmap
var Data ConfigModel
var confMutex = &sync.RWMutex{}

// ConfigModel contains config values (Mandatory values)
type ConfigModel struct {
	Reconciliation                 string      `yaml:"reconciliation"`
	ReconcileInterval              string      `yaml:"reconcileInterval"`
	SecretName                     string      `yaml:"secretName"`
	MetricPort                     string      `yaml:"metricsBindPort"`
	HealthPort                     string      `yaml:"healthProbeBindPort"`
	EventClientPort                string      `yaml:"eventClientPort"`
	LogLevel                       log.Level   `yaml:"logLevel"`
	LogFormat                      l.LogFormat `yaml:"logFormat"`
	KubeConfigPath                 string      `yaml:"kubeConfigPath"`
	EventSubReconciliation         string      `yaml:"eventSubReconciliation"`
	Namespace                      string      `yaml:"namespace"`
	EventSubscriptionMessageIds    []string    `yaml:"EventSubscriptionMessageIds"`
	EventSubscriptionEventTypes    []string    `yaml:"EventSubscriptionEventTypes"`
	EventSubscriptionResourceTypes []string    `yaml:"EventSubscriptionResourceTypes"`
}

var (
	// TickerTime: time to be set on hourly basis
	TickerTime int
	// Ticker holds a channel that delivers “ticks” of a clock at intervals
	Ticker *time.Ticker
)

// SetConfiguration will extract the config data from file
func SetConfiguration() error {

	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("Cannot read the config path %v", err)
	}
	err = yaml.Unmarshal(configData, &Data)
	if err != nil {
		return fmt.Errorf("error parsing config yaml file %v", err)
	}
	return nil
}

// TrackConfigListener listes to the chanel on the config file changes
func TrackConfigListener(errChan chan error) {
	eventChan := make(chan interface{})
	format := Data.LogFormat
	reconciliation := Data.Reconciliation
	eventsubreconciliation := Data.EventSubReconciliation
	reconciliationInterval := Data.ReconcileInterval
	transactionID := uuid.New()
	ctx := l.CreateContextForLogging(context.Background(), transactionID.String(), constants.BmcOperator, constants.TrackFileConfigActionID, constants.TrackFileConfigActionName, podName)
	go TrackConfigFileChanges(eventChan, errChan)
	for {
		select {
		case info := <-eventChan:
			l.LogWithFields(ctx).Info(info) // new data arrives through eventChan channel
			if l.Log.Level != Data.LogLevel {
				l.Log.Logger.SetLevel(Data.LogLevel)
				l.LogWithFields(ctx).Info("Log level is updated, new log level is ", Data.LogLevel)
			}
			if format != Data.LogFormat {
				logs.SetLogFormat(Data.LogFormat)
				format = Data.LogFormat
				l.LogWithFields(ctx).Info("Log format is updated, new log format is ", Data.LogFormat)
			}
			if reconciliation != Data.Reconciliation {
				if Data.Reconciliation == "Accommodate" || Data.Reconciliation == "Revert" {
					reconciliation = Data.Reconciliation
					l.LogWithFields(ctx).Info("Reconciliation action is updated, new action is ", Data.Reconciliation)
				} else {
					l.LogWithFields(ctx).Info("Please provide valid value for Reconciliation. Supported values are 'Accommodate' and 'Revert'")
				}
			}
			if eventsubreconciliation != Data.EventSubReconciliation {
				if Data.EventSubReconciliation == "Accommodate" || Data.EventSubReconciliation == "Revert" {
					eventsubreconciliation = Data.EventSubReconciliation
					l.LogWithFields(ctx).Info("EventSubReconciliation action is updated, new action is ", Data.EventSubReconciliation)
				} else {
					l.LogWithFields(ctx).Info("Please provide valid value for EventSubReconciliation. Supported values are 'Accommodate' and 'Revert'")
				}
			}
			if reconciliationInterval != Data.ReconcileInterval {
				reconciliation = Data.ReconcileInterval
				var err error
				if Data.ReconcileInterval != "" {
					if TickerTime, err = strconv.Atoi(Data.ReconcileInterval); err == nil {
						if Ticker != nil {
							Ticker.Reset(time.Duration(time.Duration(TickerTime) * time.Hour))
						}
						l.LogWithFields(ctx).Info("Reconciliation interval is updated, ticker is set to  ", TickerTime)
					} else {
						l.LogWithFields(ctx).Info("Please provide valid value for ReconcileInterval")
					}
				} else {
					l.LogWithFields(ctx).Info("Please provide valid value for ReconcileInterval")
				}
			}
		case err := <-errChan:
			l.LogWithFields(ctx).Error(err)
		}
	}
}

// TrackConfigFileChanges monitors the config changes using fsnotfiy
func TrackConfigFileChanges(eventChan chan<- interface{}, errChan chan<- error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errChan <- err
	}
	err = watcher.Add(configFilePath)
	if err != nil {
		errChan <- err
	}

	go func() {
		for {
			select {
			case fileEvent, ok := <-watcher.Events:
				if !ok {
					continue
				}
				if fileEvent.Op&fsnotify.Write == fsnotify.Write || fileEvent.Op&fsnotify.Remove == fsnotify.Remove {
					// update the config
					confMutex.Lock()
					if err := SetConfiguration(); err != nil {
						errChan <- fmt.Errorf("error while trying to set configuration: %s", err.Error())
					}
					confMutex.Unlock()
					eventChan <- "config file modified" + fileEvent.Name
				}
				//Reading file to continue the watch
				watcher.Add(configFilePath)
			case err, _ := <-watcher.Errors:
				if err != nil {
					errChan <- err
					defer watcher.Close()
				}
			}
		}
	}()
}
