$ErrorActionPrefernce = "Stop"

# Check for required env vars
foreach ($var in @("CLUSTER_CIDR", "KUBE_NETWORK", "NODE_IP")) {
    if ([System.Environment]::GetEnvironmentVariable($var) -EQ $null) {
        throw "Required envionment variable '$var' is not set!"
    }
}

# copy token for client-go incluster config
# C:\hpc\<container_id>\var\run\secrets\kubernetes.io\serviceaccount\token -> C:\var\run\secrets\kubernetes.io\serviceaccount\token
if (-not(Test-Path -Path C:\var\run)) {
    Write-Host "Copying secrets"
    Copy-Item -Recurse .\var\run  C:\var\run
}


#Write-Host "Going to sleep"
#Start-Sleep -Seconds 36000

Write-Host "Importing hns.psm1"
Import-Module ./hns.psm1


$NetworkName = $env:KUBE_NETWORK
Write-Host "Waiting for network '$NetworkName' to be available..."
while (-Not (Get-HnsNetwork | ? Name -EQ $NetworkName)) {
    Write-Debug "waiting for HNS network..."
    Start-Sleep 5
}
Write-Host "Found HNS network '$NetworkName'"

$argList = @(`
    "kube", `
    "to-local", `
#    "to-winkernel", `
    "to-winkernelfs", `
    "-v=4", `
    "--cluster-cidr=${env:CLUSTER_CIDR}", `
    "--bind-address=${env:NODE_IP}" `
    )

$network = (Get-HnsNetwork | ? Name -EQ $NetworkName)
Write-Host "Network type: $network.Type"

if ($network.Type -EQ "overlay") {
    WRite-Host "Overlay / VXLAN network detected. Waiting for host endpoint to be created..."
    while( -not (Get-HnsEndpoint | ? Name -EQ "${NetworkName}_ep")) {
        Start-Sleep 1
    }
    $sourceVip = (Get-HnsEndpoint | ? Name -EQ "${NetworkName}_ep").IpAddress
    Write-Host "Host endpoint found. Source CIP: $sourceVip"
    $argList += "--source-vip=$sourceVip"
}

Write-Host "Running ./kpng.exe $argList"
Invoke-Expression "./kpng.exe $argList"
