Write-Output "Installing Windows version of values-downloader plugin for Helm..."

$ErrorActionPreference = "Stop"

Copy-Item -Path "$env:HELM_PLUGIN_DIR\bin\windows\rockvalues.exe" -Destination "$env:HELM_PLUGIN_DIR\rockvalues.exe"
