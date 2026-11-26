# Télécharge la dernière version de InstalleurVencord.exe et la lance.
# Usage : irm https://raw.githubusercontent.com/fpvs60/vencord-installeur/main/install.ps1 | iex
& {
    $ErrorActionPreference = 'Stop'
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $url = 'https://github.com/fpvs60/vencord-installeur/releases/latest/download/InstalleurVencord.exe'
    $exe = Join-Path $env:TEMP 'InstalleurVencord.exe'

    Write-Host 'Telechargement de l''installeur Vencord...'
    $ProgressPreference = 'SilentlyContinue'
    Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $exe

    try { & $exe } finally { Remove-Item $exe -ErrorAction SilentlyContinue }
}
