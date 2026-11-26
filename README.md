# CamLauncher

Minecraft launcher for Windows, built with Go, Wails v2, React 19, and TypeScript.

> **Alpha 0.1.0:** for early testers. Expect changes while the launcher and Chill Town onboarding are refined.

## What works

- Microsoft Device Code login and offline accounts.
- Isolated Vanilla, Fabric, Forge, and NeoForge instances.
- Dynamic loader-version lookup from upstream APIs.
- Java runtime scan plus portable Java 8, 17, and 21 setup without administrator privileges.
- Chill Town server status, modpack update checks, and instance sync.
- Optional Kamaron client mods on the server home, with opt-in installation and removal (Flashback).
- Cam P2P Tunnel: obtains a short-lived, one-time Tailscale key from a Cloudflare Worker. The launcher contains no reusable Tailscale credential.

## Requirements

- Windows 10 or Windows 11, 64-bit.
- Internet access for Minecraft assets, Java setup, Microsoft login, and the P2P session.
- Microsoft Edge WebView2 Runtime. It is included with most Windows 11 installations.

## Run

Download the current alpha from [GitHub Releases](https://github.com/TieuLong07/camlauncher/releases), then run `CamLauncher.exe`.

On first launch, open **Cài Đặt** and use the Java environment panel if Java 17 has not been detected. Chill Town Forge 1.20.1 uses Java 17.

## Development

```bash
wails dev
wails build
```

The production binary is produced at `build/bin/CamLauncher.exe`.

## Official modpack updates

Official servers are registered in `officialModpacks` in `app.go` and displayed
from `frontend/src/data/servers.ts`. Add an ID and manifest URL to the registry
and the matching server metadata to the frontend when introducing a server.
The shared `CheckOfficialModpack` and `SyncOfficialModpack` APIs handle its
modpack. A server with a special connection or migration also needs its own
connection or migration handling.

Publish the archive and any individual files before changing the server's
manifest. Set a new `version`, the archive URL, its SHA-256 and size, and list
the exact official JAR filenames in `expectedMods`. Use `files` for individual
overrides with a URL, SHA-1 and size. The archive is a ZIP whose paths are
relative to the instance root, such as `mods/example.jar`,
`config/example.toml` and `kubejs/server_scripts/example.js`.

The launcher detects missing and modified official mods, preserves additional
player mods and stages updates before applying them. Worlds, player options,
screenshots and other player data are preserved. Resource packs and shaders
added by the archive are updated when unchanged by the player. Legacy
installations without recorded mod hashes perform one baseline sync to establish
the official versions. The first baseline may require downloading the archive.

Suggested client mods are pinned in `pkg/sync/suggested.go`, separately from the
official manifest. Publish each JAR before adding its URL, exact filename, size
and SHA-1 to the catalog. Downloads are verified before installation. Personal
mods remain outside official sync; players upgrading from Kamaron 1.0.3 can
choose to keep their formerly official Flashback as an optional mod.

## Security model for the P2P tunnel

The P2P broker source lives in `deploy/camlauncher-key-broker.js`. Its Tailscale OAuth credentials are stored only as Cloudflare Worker Secrets:

- `TAILSCALE_CLIENT_ID`
- `TAILSCALE_CLIENT_SECRET`

Do not place either value in source code, a GitHub secret-free config file, releases, screenshots, or chat messages.

## License

Source licensing will be added before the first stable release.
