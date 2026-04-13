# keepachangelog-cli

A simple CLI tool to manage your `CHANGELOG.md` in [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.

Built with [gokeepachangelog](https://github.com/xmidt-org/gokeepachangelog) and [urfave/cli v3](https://github.com/urfave/cli).

## Installation

```bash
go install github.com/MaxServ/keepachangelog-cli@latest
```

## Usage

### Initialize a new changelog

```bash
keepachangelog-cli init --repo https://github.com/org/repo
```

### Add an entry

```bash
keepachangelog-cli add -t added -m "New feature"
keepachangelog-cli add -t fixed -m "Bug fix"
keepachangelog-cli add -t changed -m "Updated behaviour"
keepachangelog-cli add -t deprecated -m "Old API endpoint"
keepachangelog-cli add -t removed -m "Legacy support"
keepachangelog-cli add -t security -m "Patched vulnerability"
```

### Create a release

```bash
# Release with today's date
keepachangelog-cli release -v 1.0.0 --repo https://github.com/org/repo

# Release with a specific date
keepachangelog-cli release -v 1.0.0 --date 2026-04-13 --repo https://github.com/org/repo
```

### Show the changelog

```bash
# Show everything
keepachangelog-cli show

# Show a specific version
keepachangelog-cli show -v 1.0.0
```

### Mark a release as yanked

```bash
keepachangelog-cli yank -v 1.0.0
```

### Global flags

| Flag | Alias | Default | Description |
|---|---|---|---|
| `--file` | `-f` | `CHANGELOG.md` | Path to the changelog file |

## License

See [LICENSE](LICENSE) for details.
