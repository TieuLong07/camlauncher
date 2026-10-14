# CamLauncher

Minecraft launcher for Windows, built with Go, Wails v2, React 19, and TypeScript.

> **Alpha 0.1.0:** for early testers. Expect changes while the launcher and Chill Town onboarding are refined.

## What works

- Microsoft Device Code login and offline accounts.
- Isolated Vanilla, Fabric, Forge, and NeoForge instances.
- Dynamic loader-version lookup from upstream APIs.
- Java runtime scan plus portable Java 8, 17, and 21 setup without administrator privileges.
- Chill Town server status, modpack update checks, and instance sync.
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

## Security model for the P2P tunnel

The P2P broker source lives in `deploy/camlauncher-key-broker.js`. Its Tailscale OAuth credentials are stored only as Cloudflare Worker Secrets:

- `TAILSCALE_CLIENT_ID`
- `TAILSCALE_CLIENT_SECRET`

Do not place either value in source code, a GitHub secret-free config file, releases, screenshots, or chat messages.

## License

Source licensing will be added before the first stable release.
