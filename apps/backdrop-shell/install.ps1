# Installs backdrop-shell and turns it on for new terminals on Windows.
#
#   powershell -ExecutionPolicy Bypass -File .\install.ps1
#   powershell -ExecutionPolicy Bypass -File .\install.ps1 -Scene city -Opacity 0.3
#
# Needs Go 1.25 or newer. The named settings are passed to
# "backdrop-shell enable"; change them later with that command, or by editing
# %LOCALAPPDATA%\limoni\backdrop.conf (the path "backdrop-shell status" shows).
#
# Turn it off:  backdrop-shell disable     (stays installed; enable brings it back)
# Remove it:    backdrop-shell uninstall

param(
    [string]$Scene,
    [double]$Opacity,
    [double]$Fps,
    [switch]$Still,
    [string]$Image,
    [string]$Art,
    [string]$SceneCmd,
    [switch]$Select,
    [string]$Shells
)

$ErrorActionPreference = 'Stop'
function Say($text) { Write-Host $text }
function Fail($text) { Write-Error "backdrop-shell: $text"; exit 1 }

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Fail 'Go is not installed. Get it from https://go.dev/dl/ (1.25 or newer) and run this again.'
}
$version = (& go env GOVERSION) -replace '^go', ''
$parts = $version.Split('.')
$major = [int]$parts[0]
$minor = 0
if ($parts.Length -gt 1) { $minor = [int]($parts[1] -replace '\D.*$', '') }
if ($major -lt 1 -or ($major -eq 1 -and $minor -lt 25)) {
    Fail "Go $version is too old; 1.25 or newer is needed."
}

$bindir = & go env GOBIN
if (-not $bindir) { $bindir = Join-Path (& go env GOPATH) 'bin' }
New-Item -ItemType Directory -Force -Path $bindir | Out-Null
$bin = Join-Path $bindir 'backdrop-shell.exe'

# From a checkout, build what is in it, since the scenes may be newer than
# the last release. Otherwise install the released app, as install.sh does.
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
if (Test-Path -LiteralPath (Join-Path $here 'main.go')) {
    Say "Building from $here"
    Push-Location $here
    try { & go build -o $bin . } finally { Pop-Location }
    if ($LASTEXITCODE -ne 0) { Fail 'the build failed' }
} else {
    Say 'Installing github.com/thebanri/limoni/apps/backdrop-shell@latest'
    & go install github.com/thebanri/limoni/apps/backdrop-shell@latest
    if ($LASTEXITCODE -ne 0) { Fail 'the install failed' }
}
if (-not (Test-Path -LiteralPath $bin)) { Fail "the build finished but $bin is not there" }
Say "Installed $bin"

$enableArgs = @()
if ($PSBoundParameters.ContainsKey('Scene')) { $enableArgs += @('-scene', $Scene) }
if ($PSBoundParameters.ContainsKey('Opacity')) {
    $enableArgs += @('-opacity', $Opacity.ToString([Globalization.CultureInfo]::InvariantCulture))
}
if ($PSBoundParameters.ContainsKey('Fps')) {
    $enableArgs += @('-fps', $Fps.ToString([Globalization.CultureInfo]::InvariantCulture))
}
if ($PSBoundParameters.ContainsKey('Still')) { $enableArgs += "-still=$($Still.IsPresent)" }
if ($PSBoundParameters.ContainsKey('Image')) { $enableArgs += @('-image', $Image) }
if ($PSBoundParameters.ContainsKey('Art')) { $enableArgs += @('-art', $Art) }
if ($PSBoundParameters.ContainsKey('SceneCmd')) { $enableArgs += @('-scene-cmd', $SceneCmd) }
if ($PSBoundParameters.ContainsKey('Select')) { $enableArgs += "-select=$($Select.IsPresent)" }
if ($PSBoundParameters.ContainsKey('Shells')) { $enableArgs += @('-shells', $Shells) }

& $bin enable @enableArgs
if ($LASTEXITCODE -ne 0) { Fail 'enable failed' }

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$bindir*") {
    Say ''
    Say "Note: $bindir is not on your PATH. Add it so backdrop-shell works everywhere:"
    Say "    [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$bindir', 'User')"
}
