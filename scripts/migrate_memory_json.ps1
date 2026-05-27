param(
    [string]$BaseUrl = "http://127.0.0.1:8080",
    [string]$Path = "../data/memory",
    [switch]$Import,
    [string]$AdminToken = ""
)

$ErrorActionPreference = "Stop"

$Headers = @{}
if ($AdminToken -ne "") {
    $Headers["X-Admin-Token"] = $AdminToken
}

$Body = @{
    path = $Path
    dry_run = -not $Import
} | ConvertTo-Json

Invoke-RestMethod `
    -Method Post `
    -Uri "$BaseUrl/api/v1/admin/migrate-memory-json" `
    -ContentType "application/json" `
    -Headers $Headers `
    -Body $Body
