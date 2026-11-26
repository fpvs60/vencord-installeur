# Reconstruit InstalleurVencord.exe à partir de TON Vencord actuel :
#   - le build dans %APPDATA%\Vencord\dist (avec ton fix PermissionsViewer)
#   - les thèmes activés
#   - les plugins activés/désactivés et leurs réglages
# Relance ce script après avoir changé tes plugins ou ton thème.
# Avec -Publish, publie aussi l'exe en nouvelle release GitHub (la commande des potes prend toujours la dernière).
param([switch]$Publish)
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$vc = Join-Path $env:APPDATA 'Vencord'
$payload = Join-Path $root 'payload'
$utf8 = New-Object System.Text.UTF8Encoding $false

if (Test-Path $payload) { Remove-Item $payload -Recurse -Force }
New-Item -ItemType Directory -Force "$payload\dist", "$payload\themes" | Out-Null

# Build Vencord (sans les .map, inutiles pour tes potes)
Get-ChildItem "$vc\dist" -File | Where-Object Extension -ne '.map' | Copy-Item -Destination "$payload\dist"

$s = [IO.File]::ReadAllText("$vc\settings\settings.json") | ConvertFrom-Json

foreach ($t in $s.enabledThemes) { Copy-Item (Join-Path "$vc\themes" $t) "$payload\themes" }

# Réglages perso à ne pas partager : plugin -> clés retirées
$strip = @{ PinDMs = @('userBasedCategoryList') }

$plugins = [ordered]@{}
foreach ($p in $s.plugins.PSObject.Properties) {
    if ($p.Value.enabled -eq $true) {
        $conf = [ordered]@{}
        foreach ($k in $p.Value.PSObject.Properties) {
            if ($strip[$p.Name] -contains $k.Name) { continue }
            $conf[$k.Name] = $k.Value
        }
        $plugins[$p.Name] = $conf
    } elseif ($p.Value.enabled -eq $false) {
        $plugins[$p.Name] = [ordered]@{ enabled = $false }
    }
}

$preset = [ordered]@{
    # Le build est figé : pas de mise à jour auto chez les potes (elle échouerait)
    autoUpdate             = $false
    autoUpdateNotification = $false
    useQuickCss            = $true
    enabledThemes          = @($s.enabledThemes)
    plugins                = $plugins
}
[IO.File]::WriteAllText("$payload\preset.json", ($preset | ConvertTo-Json -Depth 20), $utf8)

Push-Location $root
try {
    go build -trimpath -ldflags "-s -w" -o InstalleurVencord.exe .
    if ($LASTEXITCODE -ne 0) { throw "go build a échoué" }
} finally { Pop-Location }

$on = @($plugins.Values | Where-Object { $_.enabled -eq $true }).Count
Write-Host "OK : $root\InstalleurVencord.exe ($on plugins activés, thèmes : $($s.enabledThemes -join ', '))"

if ($Publish) {
    $tag = 'v' + (Get-Date -Format 'yyyy.MM.dd-HHmm')
    gh release create $tag "$root\InstalleurVencord.exe" --repo fpvs60/vencord-installeur `
        --title "Installeur Vencord $tag" --notes "$on plugins activés, thème : $($s.enabledThemes -join ', ')"
    if ($LASTEXITCODE -ne 0) { throw "gh release create a échoué" }
    Write-Host "Publié : $tag"
}
