function Get-SHA256Hash {
    param (
        [Parameter(ValueFromPipeline = $true)]
        [string]$Target = $null
    )

    # Handle pipeline input (equivalent to /dev/stdin)
    if (-not $Target) {
        $input_content = $input | Out-String
        if ($input_content) {
            $temp_file = [System.IO.Path]::GetTempFileName()
            $input_content | Out-File -FilePath $temp_file -Encoding ascii
            $Target = $temp_file
            $cleanup_temp = $true
        }
        else {
            Write-Error "No input provided"
            return $null
        }
    }

    try {
        # Try Get-FileHash (PowerShell native)
        try {
            $hash = (Get-FileHash -Path $Target -Algorithm SHA256).Hash.ToLower()
            return $hash
        }
        catch {
            Write-Verbose "Get-FileHash failed, trying alternatives"
        }

        # Try CertUtil (Windows native)
        try {
            $result = certutil -hashfile $Target SHA256 | Select-Object -Skip 1 | Select-Object -First 1
            $hash = $result -replace " ", ""
            return $hash.ToLower()
        }
        catch {
            Write-Verbose "CertUtil failed, trying alternatives"
        }

        # Try .NET implementation
        try {
            $hasher = [System.Security.Cryptography.SHA256]::Create()
            $stream = [System.IO.File]::OpenRead($Target)
            $hash_bytes = $hasher.ComputeHash($stream)
            $stream.Close()
            $hash = [BitConverter]::ToString($hash_bytes) -replace "-", ""
            return $hash.ToLower()
        }
        catch {
            Write-Error "Unable to compute SHA-256 hash using any available method"
            return $null
        }
    }
    finally {
        # Clean up temp file if one was created
        if ($cleanup_temp -and (Test-Path $temp_file)) {
            Remove-Item -Path $temp_file -Force
        }
    }
}

function Test-SHA256Hash {
    param (
        [Parameter(Mandatory = $true)]
        [string]$Target,
        
        [Parameter(Mandatory = $true)]
        [string]$ChecksumFile
    )

    if (-not (Test-Path $ChecksumFile)) {
        Write-Error "Checksum file '$ChecksumFile' not found"
        return $false
    }

    $basename = Split-Path -Leaf $Target
    
    # Read checksums file and find the entry for our target
    $want = $null
    foreach ($line in Get-Content -Path $ChecksumFile) {
        if ($line -match $basename) {
            # Extract the hash from the line (first field)
            $want = ($line -split '\s+')[0].ToLower()
            break
        }
    }

    if (-not $want) {
        Write-Error "Unable to find checksum for '$Target' in '$ChecksumFile'"
        return $false
    }

    $got = Get-SHA256Hash -Target $Target
    
    if ($want -ne $got) {
        Write-Error "Checksum for '$Target' did not verify: expected $want, got $got"
        return $false
    }

    return $true
}

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

# Construct the filenames
$ZIP_NAME = "$REPO_NAME`_$LATEST_TAG`_windows_$ARCH.zip"
$CKS_NAME = "$REPO_NAME`_$LATEST_TAG`_checksums.txt"

# Find the correct download URL for the ZIP and checksum file
$RELEASE_ASSETS = (Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest").assets
$DOWNLOAD_URL = $RELEASE_ASSETS | Where-Object { $_.name -eq $ZIP_NAME } | Select-Object -ExpandProperty browser_download_url
$CHECKSUM_URL = $RELEASE_ASSETS | Where-Object { $_.name -eq $CKS_NAME } | Select-Object -ExpandProperty browser_download_url

if (-not $DOWNLOAD_URL) {
    Write-Host "ERROR: Could not find a matching release ZIP file." -ForegroundColor Red
    exit 1
}

if (-not $CHECKSUM_URL) {
    Write-Host "WARNING: Could not find checksums.txt file. Skipping integrity verification." -ForegroundColor Yellow
}

# Set the download paths in the current directory
$DOWNLOAD_PATH = "$PWD\$ZIP_NAME"
$CHECKSUM_PATH = "$PWD\$CKS_NAME"

# Download the ZIP file
Write-Host "Downloading from $DOWNLOAD_URL..."
Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile $DOWNLOAD_PATH

# Download the checksum file if available
if ($CHECKSUM_URL) {
    Write-Host "Downloading checksums from $CHECKSUM_URL..."
    Invoke-WebRequest -Uri $CHECKSUM_URL -OutFile $CHECKSUM_PATH
    
    # Verify the checksum
    Write-Host "Verifying file integrity..."
    $verified = Test-SHA256Hash -Target $DOWNLOAD_PATH -ChecksumFile $CHECKSUM_PATH
    
    if (-not $verified) {
        Write-Host "ERROR: Checksum verification failed. The downloaded file may be corrupted or tampered with." -ForegroundColor Red
        Remove-Item $DOWNLOAD_PATH -Force -ErrorAction SilentlyContinue
        Remove-Item $CHECKSUM_PATH -Force -ErrorAction SilentlyContinue
        exit 1
    }
    
    Write-Host "Checksum verification successful." -ForegroundColor Green
    
    # Remove the checksum file after verification
    Remove-Item $CHECKSUM_PATH -Force
}

# Extract the ZIP file
Write-Host "Extracting $ZIP_NAME..."
Expand-Archive -Path $DOWNLOAD_PATH -DestinationPath $PWD -Force

# Remove the ZIP file after extraction
Remove-Item $DOWNLOAD_PATH -Force

Write-Host "Installation completed successfully!" -ForegroundColor Green
