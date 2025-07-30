# Équivalent de "set -e" - arrêter le script en cas d'erreur
$ErrorActionPreference = "Stop"

# Lire la version depuis le fichier plugin.yaml
$pluginYamlPath = "${env:HELM_PLUGIN_DIR}/plugin.yaml"
$yamlContent = Get-Content -Path $pluginYamlPath -Raw
$VERSION = ($yamlContent | Select-String -Pattern "version:\s*(.+)" | ForEach-Object { $_.Matches[0].Groups[1].Value }).Trim()

$rockvaluesPath = "${env:HELM_PLUGIN_DIR}/bin/linux/rockvalues"

if (-not (Test-Path -Path $rockvaluesPath -PathType Leaf)) {
    try {
        $url = "https://github.com/rockops/rockvalues/releases/download/$VERSION/rockvalues"
        
        # Créer le répertoire s'il n'existe pas
        $directory = Split-Path -Path $rockvaluesPath -Parent
        if (-not (Test-Path -Path $directory)) {
            New-Item -Path $directory -ItemType Directory -Force | Out-Null
        }
        
        # Télécharger le fichier avec Invoke-WebRequest
        Invoke-WebRequest -Uri $url -OutFile $rockvaluesPath -ErrorAction Stop
    }
    catch {
        Write-Error "Error: rockvalues binary not found in ${env:HELM_PLUGIN_DIR}/bin/linux/"
        exit 1
    }
}

# Copier le fichier
$sourcePath = "${env:HELM_PLUGIN_DIR}/bin/linux/rockvalues"
$destinationPath = "${env:HELM_PLUGIN_DIR}/rockvalues"
Copy-Item -Path $sourcePath -Destination $destinationPath -Force
