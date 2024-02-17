$ErrorActionPreference = "Stop";

#TODO: support other CNI's

ipmo -Force .\hns.psm1
$NetworkName = "Calico"
Write-Host "Waiting for HNS network $NetworkName to be created..."
while (-Not (Get-HnsNetwork | ? Name -EQ $NetworkName)) {
    Write-Debug "Still waiting for HNS network..."
    Start-Sleep 1
}
Write-Host "HNS network $NetworkName found."

$network = (Get-HnsNetwork | ? Name -EQ $NetworkName)
if ($network.Type -EQ "Overlay") {
    # This is a VXLAN network, kube-proxy needs to know the source IP to use for SNAT operations.
    Write-Host "Detected VXLAN network, waiting for Calico host endpoint to be created..."
    while (-Not (Get-HnsEndpoint | ? Name -EQ "Calico_ep")) {
        Start-Sleep 1
    }
    Write-Host "Host endpoint found."
    $sourceVip = (Get-HnsEndpoint | ? Name -EQ "Calico_ep").IpAddress
    $argList += "--source-vip=$sourceVip"

    Write-Host "Found sourceip: $sourceVip"
    [Environment]::SetEnvironmentVariable('SOURCE_VIP', $sourceVip, 'Machine')
}else {
    Write-Host "Not a VXLAN network, skipping sourceip"
}

