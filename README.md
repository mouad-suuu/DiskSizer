# DiskSizer

**DiskSizer** is a fast, terminal-based disk usage analyzer written in Go. It helps you scan and visualize disk space usage in an interactive TUI, similar to `gdu`, with support for caching, live feedback, and parallel processing.

## Features

- ⚡ **Fast Scanning** with parallel directory traversal
- 📁 **Directory Tree View** to explore disk usage interactively
- 📊 **Real-time Statistics** including total processed size and time
- 💾 **Caching** for previously scanned directories
- ⏱️ **Progress Spinner** with processed size and elapsed time
- 🔍 **Skips symlinks and inaccessible paths safely**

## Installation

```bash
git clone https://github.com/mouad-suuu/disksizer.git
cd disksizer
go build -o disksizer
```

## Usage

```bash
./disksizer <path>
```

or 

```bash
go run main.go <path>
```


Use arrow keys to navigate the directory tree.

Press Enter to expand and scan a directory.

Press q to quit the application.

Performance Notes
The scanning is multi-threaded for top-level directories and becomes sequential for deeper levels to prevent excessive resource use.

Some directories (e.g., C:\Users) may contain a large number of nested files, which can increase scan time and inflate the processed size due to traversal overhead (e.g., duplicated temp files, junctions, large caches).

Known Issues
Processed size may exceed actual used size: This happens when many intermediate files or duplicate data (e.g., user cache, temp folders) are scanned. The scanner counts every file encountered.

Slower on C:\Users: This is expected due to high file count, roaming profiles, and AppData folders.

Planned Improvements
⏳ Scan depth control

📂 Exclude/Include filters

📉 Better estimation of actual disk usage

🧪 Unit tests and benchmarks

Contributing
Pull requests are welcome! Please open an issue to discuss your ideas or report bugs.


Author: [Mouad-Mennioui](https://github.com/mouad-suuu)
Project Status: In development 🚧