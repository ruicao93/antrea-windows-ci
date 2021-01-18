package installovs

import (
	"fmt"
	"github.com/masterzen/winrm"
	"github.com/ruicao93/antrea-windows-ci/pkg/config"
	"github.com/ruicao93/antrea-windows-ci/pkg/util"
	"path"
	"strings"
)
const (
	BaseDir = `C:\antrea-windows-ci\ovs-install`
	OVSInstallationFileUrl = "https://raw.githubusercontent.com/ruicao93/antrea/nsx_ovs_install_disableHV/hack/windows/Install-OVS.ps1"
	OVSUninstallationFileUrl = "https://raw.githubusercontent.com/ruicao93/antrea/nsx_ovs_install_disableHV/hack/windows/Uninstall-OVS.ps1"
	GetNSXOVSFileUrl = "https://raw.githubusercontent.com/ruicao93/antrea/nsx_ovs_get/hack/windows/Get-NSXOVS.ps1"

	OVSDir = `c:\openvswitch`
	OVSDriverProvider = `The Linux Foundation (R)`
)

var (
	OVSInstallationFilePath = path.Join(BaseDir, "Install-OVS.ps1")
	OVSUninstallationFilePath = path.Join(BaseDir, "Uninstall-OVS.ps1")
	GetNSXOVSFilePath = path.Join(BaseDir, "Get-NSXOVS.ps1")
	NSXOVSFilePath = path.Join(BaseDir, "nsx-ovs.zip")
)

func GetOVSVersion(host *config.Host) (string, error) {
	//return util.CallPSCommand(host.Client,"ovs-vsctl.exe show")
	return "", nil
}

func OVSInstalled(host *config.Host) (bool, error) {
	return util.ServiceExists(host.Client, "ovs-vswitchd")
}

func VersionCheck(host *config.Host, expectedVersion string) (bool, error) {
	curVersion, err := GetOVSVersion(host)
	if err != nil {
		return false, err
	}
	return strings.Contains(curVersion, expectedVersion), nil
}

func GetNSXOVS(client *winrm.Client) error {
	cmd := fmt.Sprintf("%s -OutPutFile %s", GetNSXOVSFilePath, NSXOVSFilePath)
	if err := util.InvokePSCommand(client, cmd); err != nil {
		return nil
	}
	return util.PathExists(client, NSXOVSFilePath)
}


func installOVSInternal(host *config.Host, nsxOVS bool) error {
	client := host.Client
	// RM dir
	if err := util.RemoveDir(client, OVSDir); err != nil {
		return err
	}
	drivers, err := getOVSDriverNames(client)
	if err != nil {
		return err
	}
	if len(drivers) > 0 {
		return fmt.Errorf("found existed OVS drivers: %v", drivers)
	}
	// Download script
	if err := util.CreateDir(client, BaseDir); err != nil {
		return err
	}
	if err := util.DownloadFile(client, OVSInstallationFileUrl, OVSInstallationFilePath, true); err != nil {
		return err
	}
	if nsxOVS {
		if err := util.DownloadFile(client, GetNSXOVSFileUrl, GetNSXOVSFilePath, true); err != nil {
			return err
		}
		if err := GetNSXOVS(client); err != nil {
			return err
		}
	}
	cmd := OVSInstallationFilePath
	if nsxOVS {
		cmd = fmt.Sprintf("%s -LocalFile %s", cmd, NSXOVSFilePath)
	}
	return util.InvokePSCommand(client, cmd)
}

func deleteDriver(client *winrm.Client, driverName string) error {
	cmd := fmt.Sprintf("pnputil.exe /delete-driver %s", driverName)
	return util.InvokePSCommand(client, cmd)
}

func getOVSDriverNames(client *winrm.Client) ([]string, error) {
	var drivers []string
	out, err := util.CallPSCommand(client, "pnputil.exe -e")
	if err != nil {
		return drivers, err
	}
	lines := strings.Split(out, "\n")
	for index, line := range lines {
		if strings.Contains(line, OVSDriverProvider) {
			words := strings.Fields(lines[index - 1])
			drivers = append(drivers, words[len(words) - 1])
		}
	}
	return drivers, err
}

func uninstallOVSInternal(host *config.Host) error {
	client := host.Client
	// Download script
	if err := util.CreateDir(client, BaseDir); err != nil {
		return err
	}
	if err := util.DownloadFile(client, OVSUninstallationFileUrl, OVSUninstallationFilePath, true); err != nil {
		return err
	}
	// Call Script
	if err := util.InvokePSCommand(client, OVSUninstallationFilePath); err != nil {
		return err
	}
	if ovsInstalled, err := OVSInstalled(host); err != nil || ovsInstalled {
		return fmt.Errorf("OVS sill exists after uninstall")
	}
	drivers, err := getOVSDriverNames(client)
	if err != nil {
		return err
	}
	if len(drivers) == 0 {
		return fmt.Errorf("cannot find driver OVS name")
	}
	for _, driver := range drivers {
		if err := deleteDriver(client, driver); err != nil {
			return err
		}
	}
	return util.RemoveDir(client, OVSDir)
}

func InstallOVS(host *config.Host, expectedVersion string, nsxOVS bool) error {
	client := host.Client
	ovsInstalled, err := OVSInstalled(host)
	if err != nil {
		return err
	}
	if ovsInstalled {
		if expectedVersion == "" {
			return nil
		}
	}

	versionCheck, err := VersionCheck(host, expectedVersion)
	if err != nil {
		return  err
	}
	if versionCheck {
		return nil
	}

	// Remove existing OVS
	if ovsInstalled {
		if err := uninstallOVSInternal(host); err != nil {
			return err
		}

		// Restart Computer
		if err := util.RestartComputer(host, true); err != nil {
			return err
		}
	} else {
		drivers, err := getOVSDriverNames(client)
		if err != nil {
			return err
		}
		if len(drivers) > 0 {
			for _, driver := range drivers {
				if err := deleteDriver(client, driver); err != nil {
					return err
				}
			}
		}
	}

	// Install new OVS
	if err := installOVSInternal(host, nsxOVS); err != nil {
		return err
	}
	// Restart computer
	if err := util.RestartComputer(host, true); err != nil {
		return err
	}
	return nil
}

func PostInstallOVS(host *config.Host, expectedVersion string) error {
	ovsInstalled, err := OVSInstalled(host)
	if err != nil {
		return err
	}
	if !ovsInstalled {
		return fmt.Errorf("ovs not found")
	}
	versionCheck, err := VersionCheck(host, expectedVersion)
	if err != nil {
		return  err
	}
	if !versionCheck {
		return fmt.Errorf("unexpected OVS version")
	}
	return nil
}

func PostUnInstallOVS(host *config.Host, expectedVersion string) error {
	ovsInstalled, err := OVSInstalled(host)
	if err != nil {
		return err
	}
	if ovsInstalled {
		return fmt.Errorf("ovs found")
	}
	drivers, err := getOVSDriverNames(host.Client)
	if err != nil {
		return err
	}
	if len(drivers) > 0 {
		return fmt.Errorf("found OVS drivers: %v", drivers)
	}
	return nil
}


func ApplyFeature(host *config.Host, feature *config.Feature) error {
	expectedVersion := feature.GetValue(KeyOVSVersion)
	nsxOVS := false
	ovsType := feature.GetValue(KeyOVSType)
	if ovsType == ValueOVSTypeNSX {
		nsxOVS = true
	}
	if err := InstallOVS(host, expectedVersion, nsxOVS); err != nil {
		return fmt.Errorf("failed to install OVS on host %s: %v", host.HostConfig.Host, err)
	}
	if err := PostInstallOVS(host, expectedVersion); err != nil {
		return fmt.Errorf("failed to check OVS after installation on host %s: %v", host.HostConfig.Host, err)
	}
	return nil
}


