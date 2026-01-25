# Verman Windows Docker Test Environment
# Requires: Pre-built verman.exe (run "go build -o verman.exe" first)
# Build: docker build -t verman-test .
# Run:   docker run --rm -it verman-test

# Windows Server Core LTSC2025 - compatible with Windows 11 24H2
# Has setx, registry access, full Windows CLI tools (~5GB)
FROM mcr.microsoft.com/windows/servercore:ltsc2025

WORKDIR C:/verman

# Copy pre-built binary and scripts
COPY verman.exe .
COPY scripts ./scripts

# Use Windows PowerShell for RUN commands
SHELL ["powershell", "-Command"]

# Verify binary runs
RUN .\verman.exe --help

# Run setup (copies to ~/.verman/bin)
RUN .\verman.exe setup

# Add verman bin and working dir to PATH (include essential Windows paths)
# Server Core runs as ContainerAdministrator by default
ENV PATH="C:\\verman;C:\\Users\\ContainerAdministrator\\.verman\\bin;C:\\Windows\\System32;C:\\Windows;C:\\Windows\\System32\\WindowsPowerShell\\v1.0"

# Default entrypoint - Windows PowerShell (full path required for exec form)
ENTRYPOINT ["C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", "-NoExit"]
