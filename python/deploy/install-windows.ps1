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

function Test-Python {
    param(
        [string]$Executable,
        [string[]]$Prefix
    )
    if (-not (Get-Command $Executable -ErrorAction SilentlyContinue)) {
        return $false
    }
    & $Executable @($Prefix) -c "import platform, sys; raise SystemExit(0 if sys.version_info >= (3, 12) and platform.machine().lower() == 'amd64' else 1)"
    return $LASTEXITCODE -eq 0
}

$python = $null
foreach ($candidate in @(
    @{ Executable = 'py'; Prefix = @('-3.13') },
    @{ Executable = 'py'; Prefix = @('-3.12') },
    @{ Executable = 'py'; Prefix = @('-3.14') },
    @{ Executable = 'python'; Prefix = @() }
)) {
    if (Test-Python $candidate.Executable $candidate.Prefix) {
        $python = $candidate
        break
    }
}

if (-not $python) {
    Write-Error '64-bit Python 3.12 or newer is required; tailscale-py 0.7.1 has a Windows wheel for amd64 only'
    exit 1
}

$src = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$package = Join-Path $src 'mxonepeer'
if (-not (Test-Path (Join-Path $package '__main__.py'))) {
    Write-Error 'mxonepeer package was not found next to this script'
    exit 1
}

$dest = Join-Path $env:LOCALAPPDATA 'mxonepeer'
New-Item -ItemType Directory -Force -Path $dest | Out-Null

Get-CimInstance Win32_Process -Filter "Name = 'python.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.CommandLine -and $_.CommandLine -like '*mxonepeer*' } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
Stop-ScheduledTask -TaskName 'MXOnePeer' -ErrorAction SilentlyContinue

$installed = Join-Path $dest 'mxonepeer'
if (Test-Path $installed) {
    Remove-Item -Recurse -Force $installed
}
Copy-Item -Recurse -Force $package $installed
Get-ChildItem -Path $installed -Recurse -Directory -Filter '__pycache__' -ErrorAction SilentlyContinue |
    Remove-Item -Recurse -Force

$venvPython = Join-Path $dest 'venv\Scripts\python.exe'
if (-not (Test-Path $venvPython)) {
    & $python.Executable @($python.Prefix) -m venv (Join-Path $dest 'venv')
    if ($LASTEXITCODE -ne 0) {
        Write-Error 'python venv failed; install 64-bit Python 3.12 or newer and run this again'
        exit 1
    }
}

& (Join-Path $dest 'venv\Scripts\pip.exe') install --disable-pip-version-check 'tailscale-py==0.7.1'
if ($LASTEXITCODE -ne 0) {
    Write-Error 'pip install tailscale-py failed'
    exit 1
}

$envFile = Join-Path $dest 'env'
if (-not (Test-Path $envFile)) {
    Set-Content -Path $envFile -Value 'TS_AUTHKEY=' -Encoding ascii
    & icacls $envFile /inheritance:r /grant:r "${env:USERNAME}:(R,W)" | Out-Null
}

$runner = Join-Path $dest 'run.cmd'
@"
@echo off
setlocal
set "DEST=$dest"
set "PYTHONPATH=%DEST%"
set "TS_RS_EXPERIMENT=this_is_unstable_software"
if exist "%DEST%\env" (
  for /f "usebackq tokens=1,* delims==" %%A in ("%DEST%\env") do (
    if /I "%%A"=="TS_AUTHKEY" if not "%%B"=="" set "TS_AUTHKEY=%%B"
  )
)
:loop
"%DEST%\venv\Scripts\python.exe" -m mxonepeer --hostname mxonepeer --state-dir "%DEST%" --port 18020 --admin-host 127.0.0.1 --admin-port 8788 >> "%DEST%\mxonepeer.log" 2>&1
ping -n 6 127.0.0.1 >nul
goto loop
"@ | Set-Content -Path $runner -Encoding Default

$action = New-ScheduledTaskAction -Execute $runner -WorkingDirectory $dest
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
