package windowscontainer

import (
	"fmt"
	"github.com/masterzen/winrm"
	"github.com/ruicao93/antrea-windows-ci/pkg/config"
	"github.com/ruicao93/antrea-windows-ci/pkg/util"
	"k8s.io/klog"
	"strings"
)

const (
	windowsFeatureHyperV      = "Hyper-V"
	windowsFeatureContainers      = "Containers"
	optionalFeatureHyperV     = "Microsoft-Hyper-V"
	optionalFeatureHypervisor = "Microsoft-Hyper-V-Online"

	windowsFeatureStateInstalled = "Installed"
	optionalFeatureStateEnabled  = "Enabled"
)

func GetWindowsOptionalFeatureState(client *winrm.Client, featureName string) (string, error) {
	cmd := fmt.Sprintf("$(Get-WindowsOptionalFeature -Online -FeatureName %s -ErrorAction SilentlyContinue).State", featureName)
	return util.CallPSCommand(client, cmd)
}

func EnableOptionalFeatureHyperV(client *winrm.Client) (string, error) {
	cmd := fmt.Sprintf("dism /online /enable-feature /featurename:%s /all /NoRestart", optionalFeatureHyperV)
	return util.CallPSCommand(client, cmd)
}

func DisableOptionalFeature(client *winrm.Client, featureName string) (string, error) {
	cmd := fmt.Sprintf("dism /online /disable-feature /featurename:%s /NoRestart", featureName)
	return util.CallPSCommand(client, cmd)
}

func GetWindowsFeatureInstallState(client *winrm.Client, featureName string) (string, error) {
	cmd := fmt.Sprintf("$(Get-WindowsFeature -Name %s -ErrorAction SilentlyContinue).InstallState", featureName)
	return util.CallPSCommand(client, cmd)
}

func InstallWindowsFeature(client *winrm.Client, featureName string) (string, error) {
	cmd := fmt.Sprintf("Install-WindowsFeature -Name %s", featureName)
	return util.CallPSCommand(client, cmd)
}

func RemoveWindowsFeature(client *winrm.Client, featureName string) (string, error) {
	cmd := fmt.Sprintf("Remove-WindowsFeature -Name %s", featureName)
	return util.CallPSCommand(client, cmd)
}

func WindowsFeatureInstalled(client *winrm.Client, featureName string) (bool, error) {
	state, err := GetWindowsFeatureInstallState(client, featureName)
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(state, windowsFeatureStateInstalled) {
		return true, nil
	} else {
		return false, nil
	}
}

func WindowsOptionalFeatureEnabled(client *winrm.Client, featureName string) (bool, error) {
	state, err := GetWindowsOptionalFeatureState(client, featureName)
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(state, optionalFeatureStateEnabled) {
		return true, nil
	} else {
		return false, nil
	}
}

func InstallHyperV(host *config.Host) (bool, error) {
	klog.Infof("Working on install Hyper-V on host: %s", host.HostConfig.Host)
	client := host.Client
	// 1. Check Hyper-V Windows feature installation state
	installed, err := WindowsFeatureInstalled(client, windowsFeatureHyperV)
	if err != nil {
		return false, fmt.Errorf("failed to check Windows feature %s installation state on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host,err)
	}
	if installed {
		klog.Infof("Windows feature %s already installed on host %s", windowsFeatureHyperV, host.HostConfig.Host)
		return false, nil
	}

	enabled, err := WindowsOptionalFeatureEnabled(client, optionalFeatureHyperV)
	if err != nil {
		return false, fmt.Errorf("failed to check WindowsOptionalfeature %s enable state on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host, err)
	}
	if enabled {
		return false, fmt.Errorf("WindowsOptionalfeature %s already enabled", windowsFeatureHyperV)
	}

	// 2. Install Hyper-V
	_, err = InstallWindowsFeature(client, windowsFeatureHyperV)
	if err != nil {
		return false, fmt.Errorf("failed to install Windows feature %s on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host, err)
	}

	return true, nil
}

func DisableHyperV(host *config.Host) (bool, error) {
	klog.Info("Working on disable Hyper-V")
	client := host.Client
	// 1. Check Hyper-V Windows feature installation state
	installed, err := WindowsFeatureInstalled(client, windowsFeatureHyperV)
	if err != nil {
		return false, fmt.Errorf("failed to check Windows feature %s installation state on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host,err)
	}
	if !installed {
		klog.Infof("Windows feature %s not installed on host %s", windowsFeatureHyperV, host.HostConfig.Host)
	} else {
		_, err = RemoveWindowsFeature(client, windowsFeatureHyperV)
		if err != nil {
			return false, fmt.Errorf("failed to remove Windows feature %s on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host, err)
		}
		return true, nil
	}

	requireBoot := false
	enabled, err := WindowsOptionalFeatureEnabled(client, optionalFeatureHypervisor)
	if err != nil {
		return false, fmt.Errorf("failed to check WindowsOptionalfeature %s enable state on host %s: %v", optionalFeatureHypervisor, host.HostConfig.Host, err)
	}
	if !enabled {
		klog.Infof("WindowsOptionalfeature %s not enabled on host %s", optionalFeatureHypervisor, host.HostConfig.Host)
	} else {
		_, err = DisableOptionalFeature(client, optionalFeatureHypervisor)
		if err != nil {
			return false, fmt.Errorf("failed to remove Windows feature %s on host %s: %v", optionalFeatureHypervisor, host.HostConfig.Host, err)
		}
		requireBoot = true
	}

	enabled, err = WindowsOptionalFeatureEnabled(client, optionalFeatureHyperV)
	if err != nil {
		return false, fmt.Errorf("failed to check WindowsOptionalfeature %s enable state on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host, err)
	}
	if !enabled {
		klog.Infof("WindowsOptionalfeature %s not enabled on host %s", windowsFeatureHyperV, host.HostConfig.Host)
	} else {
		_, err = DisableOptionalFeature(client, optionalFeatureHyperV)
		if err != nil {
			return false, fmt.Errorf("failed to remove Windows feature %s on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host, err)
		}
		requireBoot = true
	}

	return requireBoot, nil
}

func InstallHyperVWithoutCPUCheck(host *config.Host) (bool, error) {
	klog.Infof("Working on install Hyper-V without CPU check on host: %s", host.HostConfig.Host)
	client := host.Client
	// 1. Check Hyper-V Windows feature installation state
	installed, err := WindowsFeatureInstalled(client, windowsFeatureHyperV)
	if err != nil {
		return false, fmt.Errorf("failed to check Windows feature %s installation state on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host, err)
	}
	if installed {
		klog.Infof("Windows feature %s already installed on host", windowsFeatureHyperV, host.HostConfig.Host)
		return false, nil
	}

	enabled, err := WindowsOptionalFeatureEnabled(client, optionalFeatureHyperV)
	if err != nil {
		return false, fmt.Errorf("failed to check WindowsOptionalfeature %s enable state on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host, err)
	}
	if enabled {
		klog.Infof("WindowsOptionalfeature %s already enabled", windowsFeatureHyperV)
		return false, nil
	}

	// 2. Install features
	_, err = EnableOptionalFeatureHyperV(client)
	if err != nil {
		return false, fmt.Errorf("failed to install Windows feature %s on host %s: %v", optionalFeatureHyperV, host.HostConfig.Host, err)
	}

	return true, nil
}

func AssertWindowsFeatureInstalledState(host *config.Host, featureName string, expectedState bool) error {
	client := host.Client
	installed, err := WindowsFeatureInstalled(client, featureName)
	if err != nil {
		return fmt.Errorf("failed to check Windows feature %s installation state on host %s: %v", featureName, host.HostConfig.Host, err)
	}
	if installed != expectedState {
		return fmt.Errorf("unexpected Windows feature %s installation state after installation on host %s", featureName, host.HostConfig.Host)
	}
	return nil
}

func AssertWindowsOptionalFeatureState(host *config.Host, featureName string, expectedState bool) error {
	client := host.Client
	enabled, err := WindowsOptionalFeatureEnabled(client, optionalFeatureHyperV)
	if err != nil {
		return fmt.Errorf("failed to check WindowsOptionalfeature %s enable state on host %s: %v", windowsFeatureHyperV, host.HostConfig.Host, err)
	}
	if enabled != expectedState {
		return fmt.Errorf("WindowsOptionalfeature %s state is not as expected on host %s", windowsFeatureHyperV, host.HostConfig.Host)
	}
	return nil
}

func PostInstallHyperV(host *config.Host) error {
	return AssertWindowsFeatureInstalledState(host, windowsFeatureHyperV, true)
}

func PostInstallHyperVWithoutCPUCheck(host *config.Host) error {
	return AssertWindowsOptionalFeatureState(host, windowsFeatureHyperV, true)
}

func PostInstallContainers(host *config.Host) error {
	return AssertWindowsFeatureInstalledState(host, windowsFeatureContainers, true)
}

func PostDisableHyperV(host *config.Host) error {
	if err := AssertWindowsFeatureInstalledState(host, windowsFeatureHyperV, false); err != nil {
		return err
	}

	if err := AssertWindowsOptionalFeatureState(host, optionalFeatureHypervisor, false); err != nil {
		return err
	}

	if err := AssertWindowsOptionalFeatureState(host, optionalFeatureHyperV, false); err != nil {
		return err
	}
	return nil
}

func InstallContainers(host *config.Host) (bool, error){
	klog.Infof("Working on install Containers on host: %s", host.HostConfig.Host)
	client := host.Client
	// 1. Check Windows feature Containers installation state
	installed, err := WindowsFeatureInstalled(client, windowsFeatureContainers)
	if err != nil {
		return false, fmt.Errorf("failed to check Windows feature %s installation state on host %s: %v", windowsFeatureContainers, host.HostConfig.Host,err)
	}
	if installed {
		klog.Infof("Windows feature %s already installed on host %s", windowsFeatureContainers, host.HostConfig.Host)
		return false, nil
	}

	// 2. Install containers
	_, err = InstallWindowsFeature(client, windowsFeatureContainers)
	if err != nil {
		return false, fmt.Errorf("failed to install Windows feature %s on host %s: %v", windowsFeatureContainers, host.HostConfig.Host, err)
	}
	return true, nil
}

func ApplyFeature(host *config.Host, feature *config.Feature) error {
	requireBoot := false
	if boot, err := InstallContainers(host); err != nil {
		return err
	} else {
		requireBoot = boot
	}

	installHyperVFunc := InstallHyperV
	postInstallHypervFunc := PostInstallHyperV

	for _, arg := range feature.Args {
		if arg == ParamSkipCPUCheck {
			installHyperVFunc = InstallHyperVWithoutCPUCheck
			postInstallHypervFunc = PostInstallHyperVWithoutCPUCheck
			break
		} else if arg == ParamDisableHyperV {
			installHyperVFunc = DisableHyperV
			postInstallHypervFunc = PostDisableHyperV
			break
		}
	}

	if boot, err := installHyperVFunc(host); err != nil {
		return err
	} else {
		requireBoot = requireBoot || boot
	}

	if !requireBoot {
		return nil
	}


	if err := util.RestartComputer(host, true); err != nil {
		return fmt.Errorf("failed to restart computer %s: %v", host.HostConfig.Host, err)
	}

	if err := PostInstallContainers(host); err != nil {
		return err
	}
	if err := postInstallHypervFunc(host); err != nil {
		return err
	}
	return nil
}
