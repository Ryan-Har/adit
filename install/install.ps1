$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'AMD64') { 'amd64' } else { 'arm64' }
$url = "https://github.com/Ryan-Har/adit/releases/latest/download/client-windows-$arch.exe"
$output = "C:\Windows\System32\adit.exe"

Invoke-WebRequest -Uri $url -OutFile $output
Write-Output "Adit installed successfully!"
