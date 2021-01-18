package config

import (
	"fmt"
	"github.com/masterzen/winrm"
	"k8s.io/klog"
)

const (
	DefaultUser  = "Administrator"
)

type Feature struct {
	Name string `yaml:"name"`
	Args []string `yaml:"args,omitempty"`
	KeyValues map[string]string `yaml:"keyValues,omitempty"`
}

type Task struct {
	Name string `yaml:"name"`
	Feature Feature `yaml:"feature"`
}

type HostConfig struct {
	Host string `yaml:"host"`
	Port int  `yaml:"port"`
	User string `yaml:"user,omitempty"`
	DryRun bool `yaml:"dryRun,omitempty"`
	Password string `yaml:"password"`
	Tasks []string `yaml:"tasks"`
}

type CIConfig struct {
	Hosts []HostConfig  `yaml:"hosts"`
	Tasks []Task `yaml:"tasks"`
	DryRun bool `yaml:"dryRun,omitempty"`
}

type Host struct {
	HostConfig *HostConfig
	Tasks []*Task
	Success bool
	Error error
	Client *winrm.Client
}

func (hostConfig *HostConfig) SetDefaults() {
	if hostConfig.User == "" {
		hostConfig.User = DefaultUser
	}
}

func (ciConfig *CIConfig) SetDefaults() {
	for _, hostConfig := range ciConfig.Hosts {
		hostConfig.SetDefaults()
	}
}

func (feature *Feature) GetValue(key string) string {
	if value, ok := feature.KeyValues[key]; ok {
		return value
	} else {
		return ""
	}
}

func NewWinRMClient(hostConfig *HostConfig) (*winrm.Client, error) {
	endpoint := winrm.NewEndpoint(hostConfig.Host, hostConfig.Port, false, true, nil, nil, nil, 0)
	client, err := winrm.NewClient(endpoint, hostConfig.User, hostConfig.Password)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewTasks(ciConfig *CIConfig) (map[string]*Task, error) {
	taskMap := make(map[string]*Task)
	for _, task := range ciConfig.Tasks {
		taskMap[task.Name] = &task
	}
	return taskMap, nil
}

func NewHosts(ciConfig *CIConfig, taskMap map[string]*Task) ([]*Host, error) {
	var err error
	hosts := make([]*Host, 0, len(ciConfig.Hosts))
	for _, hostConfig := range ciConfig.Hosts {
		host := Host{HostConfig: &hostConfig}
		//host.Tasks = []*Task{}
		for _, taskName := range hostConfig.Tasks {
			if task, ok := taskMap[taskName]; !ok {
				return hosts, fmt.Errorf("host %s uses a undefiend task %s", hostConfig.Host, taskName)
			} else {
				host.Tasks = append(host.Tasks, task)
			}
		}
		if host.Client, err = NewWinRMClient(host.HostConfig); err != nil {
			return hosts, fmt.Errorf("failed to init winrm client for host %s: %v", hostConfig.Host, err)
		}
		hosts = append(hosts, &host)
	}
	return hosts, nil
}

func DumpTasks(taskMap map[string]*Task) {
	for taskName, task := range taskMap {
		klog.Infof("===========Task: %s=========", taskName)
		klog.Infof("%v", *task)
	}
}

func DumpHosts(hosts []*Host) {
	for _, host := range hosts {
		klog.Infof("===========Host: %s=========", host.HostConfig.Host)
		klog.Infof("%v", host.HostConfig)
	}
}
