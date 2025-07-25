function Show-Error {
    Write-Host ""
    Write-Host "** ERROR ** : $args" -ForegroundColor Red
    Write-Host ""
}

function Show-Debug {
    if ($env:HELM_DEBUG -eq "true") {
        Write-Host "local-plugin [debug] $args" -ForegroundColor Yellow
    }
}

function Show-Info {
    Write-Host "local-plugin [info ] $args" -ForegroundColor Cyan
}

function Show-Usage {
    Show-Error "Usage: helm values chart://valuefile.yaml[@chart] [chart]"
    exit 1
}

Show-Debug "Helm get values tester"
Show-Debug "$args"

$argList = $args
while ($argList.Count -gt 1) {
    if ($argList[0] -eq "-f") {
        $argList = $argList[1..($argList.Count - 1)]
        $file = $argList[0]
        Write-Host "# Source: $file"
        if ($file -match "^chart://") {
            & "$env:HELM_PLUGIN_DIR/go/values-downloader.exe" certFile keyFile caFile $file
            if ($LASTEXITCODE -ne 0) { exit 1 }
        } else {
            if (Test-Path $file) {
                Get-Content $file
            } else {
                Write-Host "# File does not exist"
            }
        }
        Write-Host ""
        Write-Host "---"
    } else {
        Show-Debug "Param $($argList[0]) ignored (does not represent a file to load)"
    }
    $argList = $argList[1..($argList.Count - 1)]
}
