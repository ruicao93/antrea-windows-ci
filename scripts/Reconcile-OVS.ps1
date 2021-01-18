Param(
    [parameter(Mandatory = $false)] [String] $Operation="install",
    [parameter(Mandatory = $false)] [String] $OVSType,
    [parameter(Mandatory = $false)] [String] $ExpectedVersion
)
$ErrorActionPreference = "Stop"

$OP_INSTALL = "install"
$OP_UNINSTALL = "uninstall"
$TYPE_NSX = "nsx"
$OVSDir = "c:\openvswitch"
$OVSDriverProvider = "The Linux Foundation (R)"

$BaseDir = "C:\antrea-windows-ci\ovs-install"
$OVSInstallationFileUrl = "https://raw.githubusercontent.com/ruicao93/antrea/nsx_ovs_install_disableHV/hack/windows/Install-OVS.ps1"
$OVSUninstallationFileUrl = "https://raw.githubusercontent.com/ruicao93/antrea/nsx_ovs_install_disableHV/hack/windows/Uninstall-OVS.ps1"
$GetNSXOVSFileUrl = "https://raw.githubusercontent.com/ruicao93/antrea/nsx_ovs_get/hack/windows/Get-NSXOVS.ps1"

$OVSInstallationFilePath = Join-Path -Path $BaseDir -ChildPath "Install-OVS.ps1"
$OVSUninstallationFilePath = Join-Path -Path $BaseDir -ChildPath "Uninstall-OVS.ps1"
$GetNSXOVSFilePath = Join-Path -Path $BaseDir -ChildPath "Get-NSXOVS.ps1"
$NSXOVSFilePath = Join-Path -Path $BaseDir -ChildPath "win-ovs.zip"


function New-DirectoryIfNotExist($Path)
{
    if (!(Test-Path $Path))
    {
        mkdir -p $Path
    }
}

function Get-WebFileIfNotExist($Path, $URL) {
    if (Test-Path $Path) {
        return
    }
    Write-Host "Downloading $URL to $PATH"
    curl.exe -sLo $Path $URL
}

function Remove-DirIfExist($Path) {
    if (Test-Path $Path) {
        rm -r -Force $Path
    }
}

function InstallOVSInternal($nsxOVS) {
    if (ServiceExists("ovs-vswitchd")) {
        Write-Host "found existing OVS, exiting..."
        return $false
    }
    if (DriverExists) {
        Write-Host "found existing driver, exiting..."
        return $false
    }
    Remove-DirIfExist $OVSDir
    New-DirectoryIfNotExist $BaseDir
    Get-WebFileIfNotExist $OVSInstallationFilePath $OVSInstallationFileUrl
    if ($nsxOVS) {
        Get-WebFileIfNotExist $GetNSXOVSFilePath $GetNSXOVSFileUrl
        & $GetNSXOVSFilePath -OutPutFile $NSXOVSFilePath
        & $OVSInstallationFilePath -LocalFile $NSXOVSFilePath
    } else {
        & $OVSInstallationFilePath
    }
    return $true
}

function DriverExists() {
    $driversInfo = pnputil.exe -e
    $driversInfo = $driversInfo.Split([Environment]::NewLine)
    for ($index = 0; $index -lt $driversInfo.Length; $index++) {
        if ($driversInfo[$index].Contains($OVSDriverProvider)) {
            $driverNameLine = $driversInfo[$index - 1] -split '\s+'
            $driverName = $driverNameLine[$driverNameLine.Length - 1]
            Write-Host "Found driver $driverName"
            return $true
        }
    }
    return $false
}

function DeleteDrivers() {
    $driversInfo = pnputil.exe -e
    $driversInfo = $driversInfo.Split([Environment]::NewLine)
    for ($index = 0; $index -lt $driversInfo.Length; $index++) {
        if ($driversInfo[$index].Contains($OVSDriverProvider)) {
            $driverNameLine = $driversInfo[$index - 1] -split '\s+'
            $driverName = $driverNameLine[$driverNameLine.Length - 1]
            Write-Host "deleting driver $driverName"
            pnputil.exe /delete-driver $driverName
        }
    }
}

function UninstallOVSInternal() {
    New-DirectoryIfNotExist $BaseDir
    Get-WebFileIfNotExist $OVSUninstallationFilePath $OVSUninstallationFileUrl
    & $OVSUninstallationFilePath
    Remove-DirIfExist $OVSDir
    DeleteDrivers
    return $true
}

function CheckOVSVersion($ExpectedVersion) {
    $OVSInfo = ovs-vsctl.exe show
    return $OVSInfo.Contains($ExpectedVersion)
}

function ServiceExists($svcName) {
    $SVC = $(Get-Service $svcName -ErrorAction SilentlyContinue).Name
    return $SVC -contains $svcName
}

function InstallOVS() {
    Param(
        [parameter(Mandatory = $false)] [String] $OVSType,
        [parameter(Mandatory = $false)] [String] $ExpectedVersion
    )
    if (ServiceExists("ovs-vswitchd")) {
        if ($ExpectedVersion -and (CheckOVSVersion $ExpectedVersion)) {
            Write-Host "OVS installed"
            return $true
        } else {
            Write-Host "OVS version is not as expected"
            $res = UninstallOVSInternal
            if (-not $res) {
                exit 1
            }
        }
    }
    Write-Host "OVS not found"
    if (DriverExists) {
        DeleteDrivers
    }
    $NSX_OVS = $false
    if ($OVSType -eq $TYPE_NSX) {
        $NSX_OVS = $true
    }
    $res = InstallOVSInternal $NSX_OVS
    if ($res) {
        exit 0
    } else {
        exit 1
    }
}

function UninstallOVS() {
    if (-not (ServiceExists("ovs-vswitchd"))) {
        exit 0
    }
    $res = UninstallOVSInternal
    if ($res) {
        exit 0
    } else {
        exit 1
    }
}

if ($OVSType -like $TYPE_NSX) {
    $OVSType = $TYPE_NSX
}

if ($Operation -like $OP_INSTALL) {
    $Operation = $OP_INSTALL
    InstallOVS -OVSType $OVSType -ExpectedVersion $ExpectedVersion
} elseif ($Operation -like $OP_UNINSTALL) {
    $Operation = $OP_UNINSTALL
    UninstallOVS
}
