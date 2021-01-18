package main

import (
	"flag"
	"fmt"
	"github.com/ruicao93/antrea-windows-ci/pkg/config"
	"github.com/ruicao93/antrea-windows-ci/pkg/features"
	"io/ioutil"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
	"k8s.io/klog"
)

var configFile = flag.String("configFile", "config.yaml", "Hosts config file")
var dryRun = flag.Bool("dryRun", false, "Dry run")

func main() {
	flag.Parse()
	configData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		klog.Errorf("Failed to load config file: %v", err)
		os.Exit(1)
	}
	ciConfig := config.CIConfig{}
	err = yaml.Unmarshal(configData, &ciConfig)
	if err != nil {
		klog.Errorf("Failed to parse config file: %v", err)
		os.Exit(1)
	}
	ciConfig.SetDefaults()
	if len(ciConfig.Hosts) == 0 {
		klog.Warningf("Ho host found in config, exit")
		os.Exit(0)
	}
	taskMap, err := config.NewTasks(&ciConfig)
	if err != nil {
		klog.Errorf("Failed to init tasks: %v", err)
		os.Exit(1)
	}
	hosts, err := config.NewHosts(&ciConfig, taskMap)
	if err != nil {
		klog.Errorf("Failed to init hosts: %v", err)
		os.Exit(1)
	}
	config.DumpTasks(taskMap)
	config.DumpHosts(hosts)

	if *dryRun || ciConfig.DryRun {
		os.Exit(0)
	}

	var wg sync.WaitGroup
	klog.Infof("******** Start works ********")
	for _, host := range hosts {
		wg.Add(1)
		go func(host *config.Host) {
			if err := features.ApplyHost(host); err != nil {
				host.Success = false
				host.Error = fmt.Errorf("failed to apply host %s: %v", host.HostConfig.Host, err)
			} else {
				host.Success = true
			}
			wg.Done()
		}(host)
	}
	wg.Wait()
	klog.Infof("******** Works complete ********")
	DumpResults(hosts)
	for _, host := range hosts {
		host.SSHClient.Close()
	}
}

func DumpResults(hosts []*config.Host) {
	var successfulHosts []*config.Host
	var failureHosts []*config.Host
	result := "success!"
	for _, host := range hosts {
		if host.Success {
			successfulHosts = append(successfulHosts, host)
		} else {
			failureHosts = append(failureHosts, host)
		}
	}
	if len(failureHosts) > 0 {
		result = "fail!"
	}
	klog.Infof("Result: %s", result)
	klog.Infof("Success: %d", len(successfulHosts))
	klog.Infof("Fail: %d", len(failureHosts))
	for index, host := range failureHosts {
		klog.Infof("====== %d. Failure host: %s", index+1, host.HostConfig.Host)
		klog.Info(host.Error)
	}
}
