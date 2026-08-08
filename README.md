<p align="center">
  <img src="assets/icon.png" width="128" alt="aads">
  <h1 align="center">aads</h1>
  <p align="center">Manage Apple Ads campaigns from the terminal</p>
</p>

<p align="center">
  <a href="https://github.com/Aayush9029/aads/releases/latest"><img src="https://img.shields.io/github/v/release/Aayush9029/aads" alt="Release"></a>
  <a href="https://github.com/Aayush9029/aads/blob/main/LICENSE"><img src="https://img.shields.io/github/license/Aayush9029/aads" alt="License"></a>
</p>

<p align="center">
  <img src="assets/demo.gif" alt="aads demo" width="800">
</p>

## Install

```bash
brew install aayush9029/tap/aads
```

Or tap first:

```bash
brew tap aayush9029/tap
brew install aads
```

## Usage

```bash
aads configure
aads acls me
aads campaigns list --all
aads campaigns get --id 12345
aads reports campaigns --start-time 2026-07-01 --end-time 2026-07-31
aads help reports campaigns
```

See the [usage guide](docs/USAGE.md), [command reference](docs/commands/aads.md), and [Apple API map](docs/OFFICIAL_APPLE_ADS_DOCS.md).
