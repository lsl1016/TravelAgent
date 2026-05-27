param(
    [string]$Addr = ""
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location (Join-Path $Root "backend")

if ($Addr -ne "") {
    $env:HTTP_ADDR = $Addr
}

go run ./cmd/server
