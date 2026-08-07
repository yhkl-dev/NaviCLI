# NaviCLI 🎵
[![Go](https://img.shields.io/github/go-mod/go-version/yhkl-dev/NaviCLI)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/yhkl-dev/NaviCLI)](https://github.com/yhkl-dev/NaviCLI/releases)
[![License](https://img.shields.io/github/license/yhkl-dev/NaviCLI)](LICENSE)

A lightweight command line music player for Navidrome, written in Go.

![NaviCLI Screenshot](./screenshot/image1.png)
![NaviCLI Screenshot](./screenshot/image2.png)
![NaviCLI Screenshot](./screenshot/image3.png)
## Background
> I found that Feishin client is very slow on MacOS.
> To be honest, we don't need a GUI for listening to music.
> So I built this app to play Navidrome music from terminal.
> Hope you guys like it.

## Features

- 🚀 Fast and lightweight, fetches your entire library concurrently
- 🎨 Calm dark terminal UI with teal accent, braille cover art, and two-line song list
- ⏯ Play/pause/skip with real-time progress bar and spinner
- 🔍 Integrated search with in-place results (up to 100 matches)
- 🔊 Volume control with visual bar
- 🖼 Album cover art rendered as truecolor braille thumbnails
- 📊 Now Playing panel: cover art, track metadata, audio specs (format, bitrate, channels, size, plays)
- 🎛 Sort modes: Random (shuffled) / Title / Artist / Album (`s` key)
- 🔁 Repeat mode: off / all / one (`r` key)
- ⏩ Seek ±10s (`,` and `.` keys)
- 🟢 Live connection status indicator
- ⌨️ Vim-style keyboard shortcuts (`j/k`, `gg/G`, `h/l`)
- 📝 Pagination with dynamic column widths
- 🛠 Written in pure Go

## Installation

### Prerequisites
```bash
brew install mpv
```

### Install via Homebrew (Recommended)
```bash
# Add the tap
brew tap yhkl-dev/navicli

# Install navicli
brew install navicli

# Verify installation
navicli --help
```

To update NaviCLI in the future:
```bash
brew upgrade navicli
```

To uninstall:
```bash
brew uninstall navicli
brew untap yhkl-dev/navicli
```

### Install from Release
Download the latest pre-built binary from [Releases](https://github.com/yhkl-dev/NaviCLI/releases):

```bash
# For Apple Silicon (M1/M2/M3):
curl -L https://github.com/yhkl-dev/NaviCLI/releases/latest/download/navicli-darwin-arm64.tar.gz -o navicli.tar.gz
tar xzf navicli.tar.gz
chmod +x navicli
sudo mv navicli /usr/local/bin/
```

### Install for Linux
Download the latest pre-built binary from [Releases](https://github.com/yhkl-dev/NaviCLI/releases):

```bash
# For x86_64 (AMD64):
curl -L https://github.com/yhkl-dev/NaviCLI/releases/latest/download/navicli-linux-amd64.tar.gz -o navicli.tar.gz
tar xzf navicli.tar.gz
chmod +x navicli
sudo mv navicli /usr/local/bin/
```

**Prerequisite:** MPV must be installed:
```bash
# Debian/Ubuntu
sudo apt install mpv libmpv-dev

# Fedora
sudo dnf install mpv mpv-libs-devel

# Arch
sudo pacman -S mpv
```

**Troubleshooting:** If you see `error while loading shared libraries: libmpv.so.2` after installing the prerequisites, your system likely has `libmpv.so.1` while the pre-built binary was compiled against `libmpv.so.2`. This happens on Ubuntu 22.04 and older releases where the official `libmpv` package provides the older ABI version. As a workaround, create a symlink so the linker can resolve the library:

```bash
# Check which version you have
ldconfig -p | grep libmpv

# If only libmpv.so.1 is present, create a symlink
sudo ln -s /lib/x86_64-linux-gnu/libmpv.so.1 /lib/x86_64-linux-gnu/libmpv.so.2
sudo ldconfig
```

If the symlink causes issues, build from source instead — the local build will link against your system's `libmpv` version automatically.

### Install from Source (Go 1.24+ required)
```bash
git clone https://github.com/yhkl-dev/NaviCLI.git
cd NaviCLI
go build -o navicli .
sudo mv navicli /usr/local/bin/
```

### Install for Linux(Debian/Ubuntu)
```bash
sudo apt install libgl1-mesa-dev libglu1-mesa-dev freeglut3-dev libmpv-dev mpv

git clone https://github.com/yhkl-dev/NaviCLI.git
cd NaviCLI
go build -o navicli .
sudo mv navicli /usr/local/bin/
```

### Configuration
Create a config file at `~/.config/navicli/config.toml`:
```toml
[server]
url = "https://your-navidrome-server.com"
username = "your-username"
password = "your-password"
```

The legacy path `~/.config/config.toml` is also supported for backward compatibility.

## Usage
```bash
navicli
```

### Keyboard Shortcuts

**Playback Controls:**
- `Space`: Play/Pause
- `n` / `N` / `l`: Next track
- `p` / `P` / `h`: Previous track
- `→`: Next track (arrow)
- `←`: Previous track (arrow)
- `+` / `=`: Volume up (+5%)
- `-` / `_`: Volume down (-5%)
- `,` / `.`: Seek -10s / +10s
- `r`: Cycle repeat mode (off / all / one)

**Navigation (Vim-style):**
- `j` / `↓`: Move down in list
- `k` / `↑`: Move up in list
- `J` / `PgDn`: Next page
- `K` / `PgUp`: Previous page
- `>` / `]`: Next page (alternative)
- `<` / `[`: Previous page (alternative)
- `gg`: Go to first page
- `G`: Go to last page

**Sort:**
- `s`: Cycle sort mode (Random / Title / Artist / Album)

**Search & Info:**
- `/`: Open search
- `?`: Show help panel
- `ESC`: Close modal or exit (when not in search mode)
- `Ctrl+C`: Force quit

### Search

1. Press `/` to open the search box at the top
2. Type keywords to search
3. Press `Enter` to execute search
4. Results display in the main list
5. Use `↑↓` keys to select and `Enter` to play
6. Press `ESC` to clear search and restore original list
7. Press `Tab` or `↓` to switch focus from search box to list

### Display Information

When playing, the Now Playing panel shows:
- Album cover art (truecolor braille thumbnail)
- Song title (bold)
- Artist, album, format, sample rate, bitrate, channels, file size, play count (one field per line)
- Connection status dot (teal = connected, red = disconnected)
- Repeat mode indicator (shown in the title bar)

The song list shows two lines per song:
- Title (bold) + duration (right-aligned)
- Artist · album (dim, indented)

The bottom bar shows a progress bar, current/total time, volume bar, and playback status.

## Development
```bash
# Build
go build -o navicli .

# Run tests
go test ./...
```

## Roadmap
- [x] Publish to Homebrew
- [ ] Add lyrics support
- [ ] Add playlist support
- [ ] Add favorites/bookmarking
- [x] Add shuffle/repeat modes
- [x] Cross-platform builds (Linux)
- [ ] Cross-platform builds (Windows)

## Contributing
PRs are welcome! Please open an issue first to discuss what you'd like to change.

## License
[MIT](LICENSE)
