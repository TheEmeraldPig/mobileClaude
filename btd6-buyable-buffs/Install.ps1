# Builds Buyable Buffs and puts BuyableBuffs.dll into BTD6's MelonLoader Mods folder.
# Run it by double-clicking Install.bat.

$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path

function Fail($msg) {
    Write-Host ""
    Write-Host "ERROR: $msg" -ForegroundColor Red
    Write-Host ""
    Read-Host "Press Enter to close"
    exit 1
}

# ---- 1. find BTD6 ---------------------------------------------------------------------
function Find-Btd6 {
    $candidates = @()
    $steam = $null
    foreach ($key in 'HKCU:\Software\Valve\Steam', 'HKLM:\SOFTWARE\WOW6432Node\Valve\Steam') {
        try {
            $p = (Get-ItemProperty $key -ErrorAction Stop)
            if ($p.SteamPath) { $steam = $p.SteamPath }
            elseif ($p.InstallPath) { $steam = $p.InstallPath }
            if ($steam) { break }
        } catch {}
    }
    if (-not $steam) { $steam = 'C:\Program Files (x86)\Steam' }
    $candidates += Join-Path $steam 'steamapps\common\BloonsTD6'

    $vdf = Join-Path $steam 'steamapps\libraryfolders.vdf'
    if (Test-Path $vdf) {
        foreach ($m in [regex]::Matches((Get-Content $vdf -Raw), '"path"\s+"([^"]+)"')) {
            $lib = $m.Groups[1].Value -replace '\\\\', '\'
            $candidates += Join-Path $lib 'steamapps\common\BloonsTD6'
        }
    }
    foreach ($c in $candidates) {
        if (Test-Path (Join-Path $c 'BloonsTD6.exe')) { return (Resolve-Path $c).Path }
    }
    return $null
}

$game = Find-Btd6
while (-not $game) {
    Write-Host "Couldn't find BTD6 automatically."
    $game = (Read-Host "Paste the BTD6 folder (the one with BloonsTD6.exe in it)").Trim('"', ' ')
    if (-not (Test-Path (Join-Path $game 'BloonsTD6.exe'))) {
        Write-Host "BloonsTD6.exe isn't in that folder, try again." -ForegroundColor Yellow
        $game = $null
    }
}
Write-Host "BTD6 found: $game" -ForegroundColor Green

# ---- 2. check MelonLoader + Mod Helper --------------------------------------------------
if (-not (Test-Path (Join-Path $game 'MelonLoader'))) {
    Fail "MelonLoader isn't installed. Install it from https://github.com/LavaGang/MelonLoader/releases, start BTD6 once, close it, then run this again."
}
if (-not (Test-Path (Join-Path $game 'MelonLoader\Il2CppAssemblies\Assembly-CSharp.dll'))) {
    Fail "MelonLoader is installed but hasn't set itself up yet. Start BTD6 once, wait for the main menu, close it, then run this again."
}
if (-not (Test-Path (Join-Path $game 'Mods\Btd6ModHelper.dll'))) {
    Fail "BTD6 Mod Helper is missing. Download Btd6ModHelper.dll from https://github.com/gurrenm3/BTD-Mod-Helper/releases, put it in '$game\Mods', start BTD6 once, close it, then run this again."
}
Write-Host "MelonLoader and Mod Helper found." -ForegroundColor Green

# ---- 3. get the .NET SDK (installed just for this user if missing) ----------------------
$dotnet = $null
try {
    $sdks = & dotnet --list-sdks 2>$null
    if ($LASTEXITCODE -eq 0 -and $sdks) { $dotnet = 'dotnet' }
} catch {}
if (-not $dotnet) {
    $local = Join-Path $env:LOCALAPPDATA 'Microsoft\dotnet'
    if (-not (Test-Path (Join-Path $local 'dotnet.exe')) -or -not (& (Join-Path $local 'dotnet.exe') --list-sdks)) {
        Write-Host "Downloading the .NET SDK (one time, about 200 MB)..."
        $script = Join-Path $env:TEMP 'dotnet-install.ps1'
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest 'https://dot.net/v1/dotnet-install.ps1' -OutFile $script -UseBasicParsing
        & $script -Channel 8.0 -InstallDir $local
    }
    $dotnet = Join-Path $local 'dotnet.exe'
}
Write-Host ".NET SDK ready." -ForegroundColor Green

# ---- 4. build (the project copies the dll into Mods) ------------------------------------
Write-Host "Building the mod..."
& $dotnet build (Join-Path $here 'BuyableBuffs.csproj') -c Release "-p:BloonsTD6=$game" -nologo
if ($LASTEXITCODE -ne 0) {
    Fail "The build failed. Copy the red error lines above and send them to Claude to fix."
}

$dll = Join-Path $game 'Mods\BuyableBuffs.dll'
if (-not (Test-Path $dll)) { Fail "Build finished but BuyableBuffs.dll didn't end up in the Mods folder." }

Write-Host ""
Write-Host "Done! Installed: $dll" -ForegroundColor Green
Write-Host "Start BTD6, click a placed tower, then press BUFFS at the top of the screen."
Write-Host ""
Read-Host "Press Enter to close"
