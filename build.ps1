$env:CGO_ENABLED = "0"
$env:GOFLAGS = ""
$ldflags = "-s -w -X github.com/wnddd839/codebuddy-proxy/internal/version.Version=dev"

$targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Out = "releases/codebuddy-proxy-windows-x64.exe" },
    @{ GOOS = "linux";   GOARCH = "amd64"; Out = "releases/codebuddy-proxy-linux-amd64" },
    @{ GOOS = "darwin";  GOARCH = "arm64"; Out = "releases/codebuddy-proxy-darwin-arm64" },
    @{ GOOS = "darwin";  GOARCH = "amd64"; Out = "releases/codebuddy-proxy-darwin-amd64" }
)

foreach ($t in $targets) {
    $env:GOOS = $t.GOOS
    $env:GOARCH = $t.GOARCH
    Write-Host "Building $($t.GOOS)/$($t.GOARCH) -> $($t.Out)"
    & go build -trimpath -ldflags $ldflags -o $t.Out ./cmd/codebuddy-proxy
    if ($LASTEXITCODE -ne 0) { Write-Error "build failed for $($t.Out)"; exit 1 }
}

Copy-Item bin/codebuddy-proxy-gui.exe releases/codebuddy-proxy-gui-windows-x64.exe -Force
cp .env.example releases/.env.example -Force

Write-Host "All builds done."
cd releases
Get-ChildItem codebuddy-proxy-* | ForEach-Object {
    $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower()
    "$hash  $($_.Name)"
} | Out-File -Encoding ASCII SHA256SUMS.txt
Get-Content SHA256SUMS.txt
