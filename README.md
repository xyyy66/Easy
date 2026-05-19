# ⚡️ macOS Efficiency Toolkit (Pro Suite)

A highly-engineered, Terminal-based Swiss Army knife designed exclusively for macOS. Built with **Go** and the **Charmbracelet** ecosystem (`huh`, `lipgloss`), this toolkit tightly integrates with native macOS frameworks (Swift, PDFKit, Vision, CoreImage) to deliver blazing-fast, uncompromising performance without bloated third-party GUI apps.

---

## ✨ Core Features

### 📄 PDF Production
* **Merge Files:** Losslessly combine multiple PDFs instantly using native `PDFKit`.
* **Split Pages:** Extract specific page ranges into a new PDF natively.
* **Optimize Size:** Multi-tier PDF compression (Balanced, High Quality, Maximum) via `Ghostscript`.
* **Export to Image:** Convert PDF pages to high-res images.
* **Images to PDF:** Stitch multiple images (JPG, PNG, HEIC) into a single PDF via `AppKit`/`PDFKit`.
* **Extract Images:** Strip original embedded images from PDFs losslessly using `Poppler`.

### 🖼️ Image Lab
* **Modern WebP:** Convert images to next-gen WebP format using macOS `sips`.
* **Fast Resize:** Aspect-ratio-locked image resizing.
* **Privacy Strip:** Safely strip EXIF metadata and color profiles before sharing online.

### 📦 Smart Archive
* **Mac-Purified Zip:** Compress multiple files/folders while strictly ignoring `.DS_Store` and `__MACOSX` junk.
* **Multi-Format:** Support for `zip`, `tar.gz`, and high-compression `7z`.
* **Secure:** AES-256 password protection for Zip and 7z archives.

### 🔍 OCR Master
* **Zero-Cloud Privacy:** Utilizes Apple's on-device **Vision Framework** via dynamic Swift scripts.
* **Snap-to-Text:** Trigger an interactive screen capture crosshair; text is recognized instantly and copied directly to your clipboard.
* **Image File Mode:** Drag and drop existing images for rapid text extraction.

### 📸 Screen Master
* **Pro Capture:** Interactive region/window selection with custom save destinations.
* **Effects Lab (CoreImage):** Apply in-memory, professional-grade filters before saving:
  * Privacy Blur (Gaussian)
  * Classic Grayscale (Mono)
  * Vintage Sepia
  * High Contrast (Chrome)

### 📡 LAN Sharing (The Ultimate Bypass)
* **Smart Dashboard:** Hosts files/folders instantly and generates a terminal ASCII QR Code.
* **Dual-Stack IP:** Automatically detects and displays IPv4 and globally routable IPv6 (perfect for bypassing AP Isolation on Campus/Enterprise networks).
* **HTTPS Mode:** On-the-fly self-signed ECDSA certificates to satisfy iOS Safari's strict security policies.
* **Public Tunnel:** One-click zero-config SSH reverse tunnel (`localhost.run`). Creates a public HTTPS URL to bypass all local firewalls, VPN conflicts, and router isolation.

---

## 🚀 Installation & Setup

### 1. Prerequisites
Since this tool relies heavily on native macOS binaries and specific CLI utilities, ensure you have the following installed via [Homebrew](https://brew.sh/):

```bash
# Install Go (if compiling from source)
brew install go

# Install backend dependencies for PDF and Archive modules
brew install ghostscript poppler p7zip
```

### 2. Build the Application
Clone the repository and compile the binary:

```bash
git clone <your-repo-url>
cd macos-efficiency-toolkit
go mod tidy
go build -o toolkit
```

### 3. Code Signing (Crucial for macOS Firewall)
To ensure the LAN Sharing module isn't silently blocked by the macOS Application Firewall every time you run it, apply an ad-hoc signature:

```bash
codesign --force --deep -s - ./toolkit
```

---

## 💻 Usage

Simply run the executable in your terminal:

```bash
./toolkit
```

### Navigation:
* Use **Arrow Keys** or **j/k** to navigate the beautiful TUI.
* Press **Enter** to confirm.
* Press **Esc** inside any module to instantly return to the main menu.
* **File Inputs:** You can either type paths, **drag-and-drop** files directly into the terminal, or leave fields blank and press Enter to trigger a native macOS Finder selection window.

---

## 🛠 Architecture & Under the Hood
* **TUI Framework:** `github.com/charmbracelet/huh` and `lipgloss` for a minimalist, Apple-inspired interface.
* **Dynamic Swift Orchestration:** For tasks where Go libraries are either too slow or overly strict (e.g., PDF merging, Vision OCR, CoreImage filters), the Go binary dynamically generates and pipes Swift scripts to the native macOS `swift` compiler. This bridges the gap between Go's cross-platform nature and macOS's proprietary optimizations.
* **Zero-Dependency Tunneling:** The Public Tunnel feature utilizes macOS's built-in `ssh` client alongside `localhost.run`, requiring no third-party daemons (like ngrok).

---

## 📜 License
MIT License. Feel free to fork, modify, and build upon this toolkit to supercharge your own workflows.