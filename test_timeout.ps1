# PowerShell timeout test script
Write-Host "Starting test - if script doesn't complete in 5 seconds, it's stuck" -ForegroundColor Yellow

$job = Start-Job -ScriptBlock {
    Set-Location $using:PWD
    .\gobash.exe test_break_simple.sh 2>&1
}

$result = Wait-Job $job -Timeout 5

if ($result) {
    $output = Receive-Job $job
    Write-Host "Script output:" -ForegroundColor Green
    $output | Select-Object -Last 20
    Write-Host "`nScript completed" -ForegroundColor Green
    Remove-Job $job
    exit 0
} else {
    Write-Host "`nScript timeout - confirmed stuck!" -ForegroundColor Red
    Stop-Job $job
    Remove-Job $job
    exit 1
}
