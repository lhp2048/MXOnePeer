$ErrorActionPreference = 'Stop'

if ($env:OS -ne 'Windows_NT') {
    Write-Error 'this installer is for Windows'
    exit 1
}

$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if ($principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error 'run as the regular user, not an elevated administrator'
    exit 1
}

if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') {
    Write-Error 'this installer is for 64-bit Windows'
    exit 1
}

$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$bin = Join-Path $root 'dist\mxonepeer-windows-amd64.exe'
if (-not (Test-Path $bin)) {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        Write-Error "missing $bin; run .\go\deploy\build.sh on a machine with Go, then install again"
        exit 1
    }
    Push-Location $root
    $env:CGO_ENABLED = '0'
    & go build -trimpath -o $bin .\cmd\mxonepeer
    $code = $LASTEXITCODE
    Pop-Location
    if ($code -ne 0) {
        Write-Error 'go build failed'
        exit 1
    }
}

$dest = Join-Path $env:LOCALAPPDATA 'mxonepeer'
New-Item -ItemType Directory -Force -Path $dest | Out-Null

Get-Process -Name 'mxonepeer' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Stop-ScheduledTask -TaskName 'MXOnePeer' -ErrorAction SilentlyContinue

$installed = Join-Path $dest 'mxonepeer.exe'
Copy-Item -Force $bin $installed

$envFile = Join-Path $dest 'env'
if (-not (Test-Path $envFile)) {
    Set-Content -Path $envFile -Value 'TS_AUTHKEY=' -Encoding ascii
    & icacls $envFile /inheritance:r /grant:r "${env:USERNAME}:(R,W)" | Out-Null
}

$taskArgs = "--hostname mxonepeer --state-dir `"$dest`" --port 18020 --admin-host 127.0.0.1 --admin-port 8788"
$action = New-ScheduledTaskAction -Execute $installed -Argument $taskArgs -WorkingDirectory $dest
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -MultipleInstances IgnoreNew `
    -RestartCount 999 `
    -RestartInterval (New-TimeSpan -Minutes 1) `
    -ExecutionTimeLimit ([TimeSpan]::Zero)
Register-ScheduledTask `
    -TaskName 'MXOnePeer' `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -User $identity.Name `
    -Force | Out-Null
Start-ScheduledTask -TaskName 'MXOnePeer'
Write-Host 'on this Windows machine open http://127.0.0.1:8788/ to save an auth key or start Tailscale login'
