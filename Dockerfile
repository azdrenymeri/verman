# Verman Windows Docker Test Environment
# Requires: Pre-built verman.exe (run "go build -o verman.exe" first)
# Build: docker build -t verman-test .
# Run:   docker run --rm -it verman-test

# Small image with PowerShell (~300MB)
FROM mcr.microsoft.com/powershell:nanoserver-ltsc2022

WORKDIR C:/verman

# Copy pre-built binary and scripts
COPY verman.exe .
COPY scripts ./scripts

# Use PowerShell as shell for RUN commands
SHELL ["pwsh", "-Command"]

# Verify binary runs
RUN .\verman.exe --help

# Run setup (copies to ~/.verman/bin)
RUN .\verman.exe setup

# Add verman bin to PATH (shims will be created here too)
ENV PATH="C:\Users\ContainerUser\.verman\bin;${PATH}"

# Default entrypoint - use full path to pwsh
ENTRYPOINT ["C:\\Program Files\\PowerShell\\pwsh.exe"]
