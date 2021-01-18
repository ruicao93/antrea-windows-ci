package features

import (
	"fmt"
	"github.com/ruicao93/antrea-windows-ci/pkg/config"
	"github.com/ruicao93/antrea-windows-ci/pkg/features/installovs"
	"github.com/ruicao93/antrea-windows-ci/pkg/features/windowscontainer"
	"k8s.io/klog"
)

const (
	InternalFeatureWindowsContainer = "WindowsContainer"
	InternalFeatureOVSInstall = "InstallOVS"
)

var FeaturesMap map[string]func(*config.Host, *config.Feature) error

func init() {
	FeaturesMap = make(map[string]func(*config.Host, *config.Feature) error)
	FeaturesMap[InternalFeatureWindowsContainer] = windowscontainer.ApplyFeature
	FeaturesMap[InternalFeatureOVSInstall] = installovs.ApplyFeature
}

func ApplyFeature(host *config.Host, feature *config.Feature) error {
	klog.Infof("Start feature %s for host: %s", feature.Name, host.HostConfig.Host)
	if applyFunc, ok :=FeaturesMap[feature.Name]; !ok {
		return fmt.Errorf("unsupported feature: %s", feature.Name)
	} else {
		return applyFunc(host, feature)
	}
}

func ApplyHost(host *config.Host) error {
	klog.Infof("Start tasks for host: %s", host.HostConfig.Host)
	if !host.HostConfig.DryRun {
		for _, task := range host.Tasks {
			if err := ApplyFeature(host, &task.Feature); err != nil {
				return fmt.Errorf("failed to apply task %s, feature: %s for host %s: %v", task.Name, task.Feature.Name, host.HostConfig.Host, err)
			}
		}
	} else {
		klog.Infof("Skip tasks for host: %s", host.HostConfig.Host)
	}
	klog.Infof("Complete tasks for host: %s", host.HostConfig.Host)
	return nil
}