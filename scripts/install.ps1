Write-Output "Installing Windows version of values-downloader plugin for Helm..."

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Path "$env:HELM_PLUGIN_DIR\bin\windows" -Force
Copy-Item -Path "$env:HELM_PLUGIN_DIR\bin\windows\values-downloader.exe" -Destination "$env:HELM_PLUGIN_DIR\values-downloader.exe"
