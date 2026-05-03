# wiki-vault

Developer setup for the `wiki` CLI.

## Prerequisites

- Go 1.25+
- `make` (optional; the Makefile is a thin wrapper around `go` commands)
- On Linux: a running Secret Service (GNOME Keyring, KWallet) if you plan
  to exercise `wiki login` locally.

## Clone and build

    git clone https://github.com/salingsing5/coolisap.git
    cd coolisap
    make build           # produces ./wiki (or wiki.exe on Windows)

Or without make:

Bash / Git Bash / WSL:

    go build -o wiki ./cmd/wiki

Windows PowerShell or cmd:

    go build -o wiki.exe .\cmd\wiki

## Configure

`wiki.yaml` must live in the directory you run `wiki sync` from.
Generate a template by running `./wiki sync` once (it writes the
template and exits), then edit it:

    organization: your-azure-devops-organization
    project: Your Project Name
    wiki: Your Project.wiki

`wiki sync` creates `articles/<wiki>/` next to `wiki.yaml` (where
`<wiki>` comes from the `wiki:` value) and writes the synced pages
there — no need to `mkdir` one yourself.

## Run

Bash / Git Bash / WSL:

    ./wiki login
    ./wiki sync
    ./wiki logout

Windows PowerShell or cmd (binary must be `wiki.exe`; PowerShell won't
launch an extension-less file):

    .\wiki.exe login
    .\wiki.exe sync
    .\wiki.exe logout

If you built with `go build -o wiki ./cmd/wiki` on Windows, rebuild with
`go build -o wiki.exe ./cmd/wiki` (or `make build`, which already adds
`.exe` when run from cmd/PowerShell).

## Test

    make test
    # or
    go test ./...

## Release snapshot

    goreleaser release --snapshot --clean   # requires goreleaser installed locally

Release config lives in `.goreleaser.yaml`.

## Layout

    cmd/wiki              # CLI entry point
    internal/azuredevops  # ADO API client
    internal/cli          # cobra commands
    internal/config       # wiki.yaml load/save
    internal/credentials  # keyring wrapper
    internal/sync         # tree diff + filesystem writer
