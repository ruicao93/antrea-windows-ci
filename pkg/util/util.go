package util

import (
	"fmt"
	"github.com/masterzen/winrm"
	"github.com/ruicao93/antrea-windows-ci/pkg/config"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"strings"
	"time"
)

func CallPSCommand(client *winrm.Client, cmd string) (string, error) {
	stdout, stderr, rc, err := client.RunPSWithString(cmd, "")
	if err != nil {
		return stdout, err
	}
	//if stderr != "" {
	//	return stdout, fmt.Errorf("%s", stderr)
	//}
	if rc != 0 {
		return stderr, fmt.Errorf("exit code: %d, error: %s", rc, stderr)
	}
	return stdout, nil
}

func InvokePSCommand(client *winrm.Client, cmd string) error {
	_, err := CallPSCommand(client, cmd)
	return err
}

func RestartComputer(host *config.Host, waitReboot bool) error {
	client := host.Client
	cmd := "Restart-Computer -Force"
	_, err := CallPSCommand(client, cmd)
	if err != nil {
		return nil
	}
	if !waitReboot {
		return nil
	}
	// Wait down
	err = wait.PollImmediate(10 * time.Second, 3 * time.Minute, func() (done bool, err error) {
		if _, err := CallPSCommand(client, "ls"); err != nil {
			klog.Infof("host %s is down now", host.HostConfig.Host)
			return true, nil
		}
		klog.Infof("Waiting for host %s down", host.HostConfig.Host)
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("timeout to wait host %s down: %v", host.HostConfig.Host, err)
	}

	// Wait up
	err = wait.PollImmediate(10 * time.Second, 10 * time.Minute, func() (done bool, err error) {
		if _, err := CallPSCommand(client, "ls"); err != nil {
			klog.Infof("Waiting for host %s up", host.HostConfig.Host)
			return false, nil
		}
		klog.Infof("host %s is up now", host.HostConfig.Host)
		return true, nil
	})
	return nil
}

func CreateDir(client *winrm.Client, path string) error {
	cmd := fmt.Sprintf(`mkdir -Force "%s"`, path)
	return InvokePSCommand(client, cmd)
}

func RemoveFile(client *winrm.Client, path string) error {
	cmd := fmt.Sprintf(`rm -Force "%s"`, path)
	return InvokePSCommand(client, cmd)
}

func PathExists(client *winrm.Client, path string) error {
	cmd := fmt.Sprintf("Get-Item %s", path)
	return InvokePSCommand(client, cmd)
}

func RemoveDir(client *winrm.Client, path string) error {
	cmd := fmt.Sprintf(`rm -r -Force "%s"`, path)
	return InvokePSCommand(client, cmd)
}

func DownloadFile(client *winrm.Client, url, dstPath string, removeOnExist bool) error {
	cmd := fmt.Sprintf("curl.exe -sLo %s %s", dstPath, url)
	//if removeOnExist {
	//	cmd = fmt.Sprintf("rm -Force %s && %s", dstPath, cmd)
	//}
	return InvokePSCommand(client, cmd)
}

func GetService(client *winrm.Client, svcName string) (string, error) {
	cmd := fmt.Sprintf(`$(Get-Service "%s" -ErrorAction SilentlyContinue).Name`, svcName)
	return CallPSCommand(client, cmd)
}

func ServiceExists(client *winrm.Client, svcName string) (bool, error) {
	existedSvc, err := GetService(client, svcName)
	if err != nil {
		return false, err
	}
	return strings.Contains(existedSvc, svcName), nil
}
