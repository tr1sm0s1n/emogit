# Define repository details
$REPO_OWNER = "tr1sm0s1n"
$REPO_NAME = "emogit"

# Detect system architecture
$ARCH = "amd64"
$OS_ARCH = [System.Environment]::Is64BitOperatingSystem

if (-not $OS_ARCH) { $ARCH = "386" }
if ($ENV:PROCESSOR_ARCHITECTURE -match "ARM") { $ARCH = "arm64" }

# Get the latest release tag from GitHub and remove "v" prefix
$LATEST_TAG = (Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest").tag_name -replace '^v', ''

# Construct the ZIP filename
$ZIP_NAME = "$REPO_NAME`_$LATEST_TAG`_windows_$ARCH.zip"

# Find the correct download URL for the ZIP
$DOWNLOAD_URL = (Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest").assets | Where-Object { $_.name -eq $ZIP_NAME } | Select-Object -ExpandProperty browser_download_url

if (-not $DOWNLOAD_URL) {
    Write-Host "ERROR: Could not find a matching release ZIP file." -ForegroundColor Red
    exit 1
}

# Set the download path in the current directory
$DOWNLOAD_PATH = "$PWD\$ZIP_NAME"

# Download the ZIP file
Write-Host "Downloading from $DOWNLOAD_URL..."
Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile $DOWNLOAD_PATH

# Extract the ZIP file
Write-Host "Extracting $ZIP_NAME..."
Expand-Archive -Path $DOWNLOAD_PATH -DestinationPath $PWD -Force

# Remove the ZIP file after extraction
Remove-Item $DOWNLOAD_PATH -Force

Write-Host "Done!"
