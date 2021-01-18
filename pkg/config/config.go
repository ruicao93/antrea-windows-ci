package config

import (
	"fmt"
	"github.com/masterzen/winrm"
	"golang.org/x/crypto/ssh"
	"k8s.io/klog"
	"time"
)

const (
	DefaultUser = "Administrator"
)

type Feature struct {
	Name      string            `yaml:"name"`
	Args      []string          `yaml:"args,omitempty"`
	KeyValues map[string]string `yaml:"keyValues,omitempty"`
}

type Task struct {
	Name    string  `yaml:"name"`
	Feature Feature `yaml:"feature"`
}

type HostConfig struct {
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	User     string   `yaml:"user,omitempty"`
	DryRun   bool     `yaml:"dryRun,omitempty"`
	Password string   `yaml:"password"`
	Tasks    []string `yaml:"tasks"`
}

type CIConfig struct {
	Hosts  []HostConfig `yaml:"hosts"`
	Tasks  []Task       `yaml:"tasks"`
	DryRun bool         `yaml:"dryRun,omitempty"`
}

type Host struct {
	HostConfig *HostConfig
	Tasks      []*Task
	Success    bool
	Error      error
	Client     *winrm.Client
	SSHClient  *ssh.Client
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

func NewSSHClient(hostConfig *HostConfig) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		Timeout:         time.Second, //ssh 连接time out 时间一秒钟, 如果ssh验证错误 会在一秒内返回
		User:            hostConfig.User,
		Auth:            []ssh.AuthMethod{ssh.Password(hostConfig.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以， 但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}
	addr := fmt.Sprintf("%s:%d", hostConfig.Host, 22)
	return ssh.Dial("tcp", addr, sshConfig)
}

func NewTasks(ciConfig *CIConfig) (map[string]*Task, error) {
	taskMap := make(map[string]*Task)
	for i := 0; i < len(ciConfig.Tasks); i++ {
		task := &ciConfig.Tasks[i]
		taskMap[task.Name] = task
	}
	return taskMap, nil
}

func NewHosts(ciConfig *CIConfig, taskMap map[string]*Task) ([]*Host, error) {
	var err error
	hosts := make([]*Host, 0, len(ciConfig.Hosts))
	for i := 0; i < len(ciConfig.Hosts); i++ {
		hostConfig := &ciConfig.Hosts[i]
		host := Host{HostConfig: hostConfig}
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

		if host.SSHClient, err = NewSSHClient(host.HostConfig); err != nil {
			return hosts, fmt.Errorf("failed to init ssh client for host %s: %v", hostConfig.Host, err)
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
