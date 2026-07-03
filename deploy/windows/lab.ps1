<#
.SYNOPSIS
  PZ Platform — single-Windows-OS test lab.

  Everything runs on one Windows machine: the backend (Windows service),
  N Project Zomboid dedicated servers (steamcmd), one agent per server
  (Windows services), and a verify step that proves the full content loop:

    agent scans mods → pushes blobs + manifest → backend serves join →
    join-cli downloads and extracts a byte-identical profile.

.USAGE
  Run from an elevated PowerShell (Administrator):

    .\lab.ps1 install                 # build binaries, install PZ servers + services
    .\lab.ps1 install -SkipGameServers  # platform-only lab (no steamcmd / PZ download)
    .\lab.ps1 up                      # start backend, agents, game servers
    .\lab.ps1 verify                  # prove the end-to-end loop; exit 0 = PASS
    .\lab.ps1 status                  # one-screen health view
    .\lab.ps1 down                    # stop everything
    .\lab.ps1 clean                   # down + uninstall services (keeps $Root data)

.NOTES
  - Requires Go on PATH (builds lab binaries from this repo).
  - PZ Dedicated Server (app 380870) is ~4 GB via steamcmd; use
    -SkipGameServers for a fast platform-only lab.
#>
[CmdletBinding()]
param(
  [Parameter(Position = 0)]
  [ValidateSet('install', 'up', 'down', 'status', 'verify', 'clean')]
  [string]$Command = 'status',

  [string]$Root = 'C:\pz-lab',
  [int]$Servers = 2,
  [int]$BackendPort = 8080,
  [switch]$SkipGameServers
)

$ErrorActionPreference = 'Stop'
$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$BackendUrl = "http://localhost:$BackendPort"
$SteamCmdZip = 'https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip'
$PZAppId = 380870

function Log([string]$msg)  { Write-Host "[lab] $msg" -ForegroundColor Cyan }
function Ok([string]$msg)   { Write-Host "  PASS  $msg" -ForegroundColor Green }
function Fail([string]$msg) { Write-Host "  FAIL  $msg" -ForegroundColor Red }

function Assert-Admin {
  $id = [Security.Principal.WindowsIdentity]::GetCurrent()
  $p = New-Object Security.Principal.WindowsPrincipal($id)
  if (-not $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'lab.ps1 must run as Administrator (services + steamcmd install).'
  }
}

function ServerName([int]$i) { "pzlab$i" }
function AgentServiceName([int]$i) { "PZAgent$i" }
function ServerDataDir([int]$i) { Join-Path $Root "data\srv$i" }
function ServerModsDir([int]$i) { Join-Path (ServerDataDir $i) 'mods' }
function ServerGamePort([int]$i) { 16261 + (($i - 1) * 2) }

# ───────────────────────── install ─────────────────────────

function Build-Binaries {
  Log "building lab binaries from $RepoRoot"
  if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go not found on PATH. Install Go 1.23+ (choco install golang) and retry.'
  }
  $bin = Join-Path $Root 'bin'
  New-Item -ItemType Directory -Force -Path $bin | Out-Null
  Push-Location $RepoRoot
  try {
    go build -o (Join-Path $bin 'pz-backend.exe') ./apps/backend/cmd/backend
    if ($LASTEXITCODE -ne 0) { throw 'go build backend failed' }
    go build -o (Join-Path $bin 'pz-agent.exe') ./apps/pz-agent/cmd/agent
    if ($LASTEXITCODE -ne 0) { throw 'go build agent failed' }
    go build -o (Join-Path $bin 'join-cli.exe') ./apps/join-cli
    if ($LASTEXITCODE -ne 0) { throw 'go build join-cli failed' }
  }
  finally { Pop-Location }
}

function Install-SteamCmd {
  $dir = Join-Path $Root 'steamcmd'
  $exe = Join-Path $dir 'steamcmd.exe'
  if (Test-Path $exe) { Log 'steamcmd already installed'; return $exe }
  Log 'downloading steamcmd'
  New-Item -ItemType Directory -Force -Path $dir | Out-Null
  $zip = Join-Path $dir 'steamcmd.zip'
  Invoke-WebRequest -Uri $SteamCmdZip -OutFile $zip
  Expand-Archive -Path $zip -DestinationPath $dir -Force
  Remove-Item $zip
  return $exe
}

function Install-PZServer {
  $steamcmd = Install-SteamCmd
  $dest = Join-Path $Root 'pzserver'
  Log "installing PZ Dedicated Server (app $PZAppId) into $dest — several GB, be patient"
  # steamcmd exits non-zero on first self-update; retry once.
  & $steamcmd +force_install_dir $dest +login anonymous +app_update $PZAppId validate +quit
  if (-not (Test-Path (Join-Path $dest 'StartServer64.bat'))) {
    & $steamcmd +force_install_dir $dest +login anonymous +app_update $PZAppId validate +quit
  }
  if (-not (Test-Path (Join-Path $dest 'StartServer64.bat'))) {
    throw "PZ server install failed: StartServer64.bat not found in $dest"
  }
}

function Seed-SampleMods([int]$i) {
  $mods = ServerModsDir $i
  if (Test-Path $mods) {
    if ((Get-ChildItem $mods | Measure-Object).Count -gt 0) { return }
  }
  Log "seeding sample mods for $(ServerName $i)"
  $modDir = Join-Path $mods "LabMod$i"
  New-Item -ItemType Directory -Force -Path (Join-Path $modDir 'media\lua') | Out-Null
  Set-Content -Path (Join-Path $modDir 'mod.info') -Value "name=LabMod$i`nid=LabMod$i"
  Set-Content -Path (Join-Path $modDir 'media\lua\init.lua') -Value "print('lab mod $i for $(ServerName $i)')"
}

function Install-Services {
  $bin = Join-Path $Root 'bin'
  $backendData = Join-Path $Root 'data\backend'
  New-Item -ItemType Directory -Force -Path (Join-Path $backendData 'store') | Out-Null
  if (-not (Test-Path (Join-Path $backendData 'registry.json'))) {
    Set-Content -Path (Join-Path $backendData 'registry.json') -Value '{"servers":[]}'
  }

  Log 'installing PZBackend service'
  & (Join-Path $bin 'pz-backend.exe') -service install `
    -addr ":$BackendPort" `
    -registry (Join-Path $backendData 'registry.json') `
    -store (Join-Path $backendData 'store') `
    -content-registry (Join-Path $backendData 'content-registry.json') `
    -fixtures (Join-Path $backendData 'fixtures') `
    -logfile (Join-Path $Root 'logs\pz-backend.log')

  foreach ($i in 1..$Servers) {
    Log "installing $(AgentServiceName $i) service for $(ServerName $i)"
    & (Join-Path $bin 'pz-agent.exe') -service install `
      -service-name (AgentServiceName $i) `
      -server (ServerName $i) `
      -mods (ServerModsDir $i) `
      -backend $BackendUrl `
      -interval 60s `
      -logfile (Join-Path $Root "logs\pz-agent$i.log")
  }
}

function Invoke-Install {
  Assert-Admin
  New-Item -ItemType Directory -Force -Path $Root, (Join-Path $Root 'logs'), (Join-Path $Root 'run') | Out-Null
  Build-Binaries
  foreach ($i in 1..$Servers) { Seed-SampleMods $i }
  if (-not $SkipGameServers) { Install-PZServer }
  Install-Services
  Log "install complete. Next: .\lab.ps1 up"
}

# ───────────────────────── up / down ─────────────────────────

function Start-GameServer([int]$i) {
  $bat = Join-Path $Root 'pzserver\StartServer64.bat'
  if (-not (Test-Path $bat)) { Log "PZ server not installed — skipping game server $(ServerName $i)"; return }
  $pidFile = Join-Path $Root "run\srv$i.pid"
  if (Test-Path $pidFile) {
    $oldPid = Get-Content $pidFile
    if (Get-Process -Id $oldPid -ErrorAction SilentlyContinue) { Log "$(ServerName $i) already running (pid $oldPid)"; return }
  }
  Log "starting PZ server $(ServerName $i) on port $(ServerGamePort $i)"
  $args = @(
    "-cachedir=$(ServerDataDir $i)",
    '-servername', (ServerName $i),
    '-port', (ServerGamePort $i),
    '-adminusername', 'admin',
    '-adminpassword', 'pzlab-admin-1!'
  )
  $proc = Start-Process -FilePath $bat -ArgumentList $args `
    -WorkingDirectory (Join-Path $Root 'pzserver') `
    -RedirectStandardOutput (Join-Path $Root "logs\srv$i.out.log") `
    -RedirectStandardError (Join-Path $Root "logs\srv$i.err.log") `
    -PassThru -WindowStyle Hidden
  Set-Content -Path $pidFile -Value $proc.Id
}

function Stop-GameServer([int]$i) {
  $pidFile = Join-Path $Root "run\srv$i.pid"
  if (-not (Test-Path $pidFile)) { return }
  $procId = Get-Content $pidFile
  $proc = Get-Process -Id $procId -ErrorAction SilentlyContinue
  if ($proc) {
    Log "stopping PZ server $(ServerName $i) (pid $procId)"
    # Kill the whole tree: the bat spawns java.
    & taskkill /PID $procId /T /F | Out-Null
  }
  Remove-Item $pidFile -ErrorAction SilentlyContinue
}

function Invoke-Up {
  Assert-Admin
  Log 'starting PZBackend'
  Start-Service PZBackend
  foreach ($i in 1..$Servers) {
    Log "starting $(AgentServiceName $i)"
    Start-Service (AgentServiceName $i)
    if (-not $SkipGameServers) { Start-GameServer $i }
  }
  Log "lab is up. Verify with: .\lab.ps1 verify"
}

function Invoke-Down {
  Assert-Admin
  foreach ($i in 1..$Servers) {
    Stop-GameServer $i
    Stop-Service (AgentServiceName $i) -ErrorAction SilentlyContinue
  }
  Stop-Service PZBackend -ErrorAction SilentlyContinue
  Log 'lab is down'
}

function Invoke-Clean {
  Assert-Admin
  Invoke-Down
  $bin = Join-Path $Root 'bin'
  $agentExe = Join-Path $bin 'pz-agent.exe'
  $backendExe = Join-Path $bin 'pz-backend.exe'
  foreach ($i in 1..$Servers) {
    if (Test-Path $agentExe) {
      try { & $agentExe -service uninstall -service-name (AgentServiceName $i) } catch { Log "uninstall $(AgentServiceName $i): $_" }
    }
  }
  if (Test-Path $backendExe) {
    try { & $backendExe -service uninstall } catch { Log "uninstall PZBackend: $_" }
  }
  Log "services removed. Lab data kept at $Root (delete manually if desired)"
}

# ───────────────────────── status / verify ─────────────────────────

function Invoke-Status {
  Write-Host "`n─── services ───"
  $svcNames = @('PZBackend') + (1..$Servers | ForEach-Object { AgentServiceName $_ })
  Get-Service -Name $svcNames -ErrorAction SilentlyContinue |
    Format-Table Name, Status, StartType -AutoSize

  Write-Host "─── game servers ───"
  foreach ($i in 1..$Servers) {
    $pidFile = Join-Path $Root "run\srv$i.pid"
    $state = 'stopped'
    if (Test-Path $pidFile) {
      $procId = Get-Content $pidFile
      if (Get-Process -Id $procId -ErrorAction SilentlyContinue) { $state = "running (pid $procId)" }
    }
    Write-Host ("  {0}  port {1}  {2}" -f (ServerName $i), (ServerGamePort $i), $state)
  }

  Write-Host "`n─── backend ───"
  try {
    $health = Invoke-RestMethod "$BackendUrl/api/v1/health" -TimeoutSec 3
    Write-Host "  $BackendUrl  status=$($health.status) version=$($health.version)"
    $agents = Invoke-RestMethod "$BackendUrl/api/v1/agents" -TimeoutSec 3
    foreach ($a in $agents.agents) {
      Write-Host ("  agent {0}  {1}  mods={2}  lastSeen={3}" -f $a.serverId, $a.status, $a.modCount, $a.lastSeen)
    }
  }
  catch { Write-Host "  backend unreachable: $_" -ForegroundColor Yellow }
  Write-Host ''
}

function Invoke-Verify {
  $failures = 0
  Log "verifying the end-to-end content loop against $BackendUrl"

  # 1. backend health
  try {
    $health = Invoke-RestMethod "$BackendUrl/api/v1/health" -TimeoutSec 5
    if ($health.status -eq 'ok') { Ok 'backend /health' } else { Fail "backend /health status=$($health.status)"; $failures++ }
  }
  catch { Fail "backend unreachable: $_"; $failures++ }

  # 2. all servers registered (agents auto-register them)
  try {
    $servers = (Invoke-RestMethod "$BackendUrl/api/v1/servers" -TimeoutSec 5).servers
    foreach ($i in 1..$Servers) {
      $name = ServerName $i
      if ($servers | Where-Object { $_.id -eq $name }) { Ok "server $name registered" }
      else { Fail "server $name not in registry (agent not synced yet?)"; $failures++ }
    }
  }
  catch { Fail "GET /servers: $_"; $failures++ }

  # 3. agents heartbeating
  try {
    $agents = (Invoke-RestMethod "$BackendUrl/api/v1/agents" -TimeoutSec 5).agents
    foreach ($i in 1..$Servers) {
      $name = ServerName $i
      $a = $agents | Where-Object { $_.serverId -eq $name }
      if ($a -and $a.status -eq 'online') { Ok "agent for $name online (mods=$($a.modCount))" }
      elseif ($a) { Fail "agent for $name is $($a.status)"; $failures++ }
      else { Fail "no agent state for $name"; $failures++ }
    }
  }
  catch { Fail "GET /agents: $_"; $failures++ }

  # 4. manifests published
  foreach ($i in 1..$Servers) {
    $name = ServerName $i
    try {
      $hist = Invoke-RestMethod "$BackendUrl/api/v1/manifests/$name/history" -TimeoutSec 5
      $latest = ($hist.versions | Measure-Object -Property version -Maximum).Maximum
      if ($latest -ge 1) { Ok "manifest for $name published (v$latest)" }
      else { Fail "no manifest versions for $name"; $failures++ }
    }
    catch { Fail "manifest history for ${name}: $_"; $failures++ }
  }

  # 5. join through the same pipeline the launcher UI uses
  $launcherRoot = Join-Path $Root 'launcher'
  New-Item -ItemType Directory -Force -Path $launcherRoot | Out-Null
  $joinCli = Join-Path $Root 'bin\join-cli.exe'
  $primary = ServerName 1
  $env:PZ_LAUNCHER_ROOT = $launcherRoot
  & $joinCli -server $primary -backend $BackendUrl 2>&1 | Out-File (Join-Path $Root 'logs\verify-join.log')
  if ($LASTEXITCODE -eq 0) { Ok "join-cli -backend completed for $primary" }
  else { Fail "join-cli failed (see logs\verify-join.log)"; $failures++ }

  # 6. extracted profile is byte-identical to the server's mods
  $srcMods = ServerModsDir 1
  $dstMods = Join-Path $launcherRoot "profiles\$primary\mods"
  if (Test-Path $dstMods) {
    $mismatch = 0
    Get-ChildItem -Recurse -File $srcMods | ForEach-Object {
      $rel = $_.FullName.Substring($srcMods.Length)
      $peer = Join-Path $dstMods $rel
      if (-not (Test-Path $peer)) { $mismatch++; return }
      if ((Get-FileHash $_.FullName).Hash -ne (Get-FileHash $peer).Hash) { $mismatch++ }
    }
    if ($mismatch -eq 0) { Ok "profile content byte-identical to $primary server mods" }
    else { Fail "$mismatch file(s) missing or different in extracted profile"; $failures++ }
  }
  else { Fail "profile mods dir not created: $dstMods"; $failures++ }

  Write-Host ''
  if ($failures -eq 0) { Log 'VERIFY PASSED — full loop proven on this machine'; exit 0 }
  else { Log "VERIFY FAILED — $failures check(s) failed"; exit 1 }
}

switch ($Command) {
  'install' { Invoke-Install }
  'up'      { Invoke-Up }
  'down'    { Invoke-Down }
  'status'  { Invoke-Status }
  'verify'  { Invoke-Verify }
  'clean'   { Invoke-Clean }
}
