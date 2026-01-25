# Verman

A lightweight version manager for Windows, born out of personal necessity.

A tool to manage JVM languages and the build tools that come with them.

## Supported Tools

- **Java** — Temurin, Corretto, Zulu
- **Scala** — 2.x and 3.x
- **Kotlin**
- **Gradle, Maven, SBT, Mill**
- **Node.js, Go**

## Getting Started

```powershell
# Build (requires Go 1.21+)
go build -o verman.exe

# Set up PATH
.\verman.exe setup

# Restart your terminal, then:
verman install java 21
verman install scala 3.3.1
verman install gradle 8.5

# Switch between versions
verman use java 17
verman use java 21
```

## Commands

```powershell
verman install <tool> <version>   # Install a version
verman use <tool> <version>       # Switch to a version
verman list [tool]                # List installed versions
verman list <tool> --all          # List available versions
verman current                    # Show active versions
verman detect                     # Detect versions from project files
verman detect --apply             # Detect and switch automatically
```

## Project Detection

Verman understands `.java-version`, `.nvmrc`, `.scala-version`, `go.mod`, and similar files. Walk into a project directory and run:

```powershell
verman detect --apply
```

## Java Distributions

Works with SDKMAN-style version identifiers:

```powershell
verman install java 21          # Eclipse Temurin (default)
verman install java 21-amzn     # Amazon Corretto
verman install java 21-zulu     # Azul Zulu
```

## Under the Hood

Nothing fancy. Verman downloads official binaries, extracts them to `~/.verman/versions/`, and uses Windows junction points to switch between versions. No admin privileges required.

Run `verman init --install` once to wire up your shell, and you're set.

## License

MIT
