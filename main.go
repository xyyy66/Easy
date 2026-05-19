package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/skip2/go-qrcode"
)

// AppConfig holds the state of the application
type AppConfig struct {
	Module          string
	Action          string
	InputFiles      string
	OutputFile      string
	PageRange       string
	CompressionMode string
	ImageWidth      string
	ArchiveFormat   string
	ArchivePassword string
	LANPort         string
	LANHTTPS        bool
	LANTunnel       bool
	ScreenAction    string
	FilterType      string
	OCRMode         string
}

func main() {
	config := &AppConfig{}

	// Define styles
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Padding(1)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true).Padding(1)

	// Infinite loop to keep returning to main menu
	for {
		err := showMainMenu(config)
		if err != nil {
			if err == huh.ErrUserAborted {
				fmt.Println("\nExiting application. Goodbye!")
				break
			}
			fmt.Println(errorStyle.Render(fmt.Sprintf("Error: %v", err)))
			continue
		}

		if config.Module == "Quit" {
			fmt.Println("\nExiting application. Goodbye!")
			break
		}

		// Process user selections
		if config.Module == "LAN" {
			err = handleLANSharing(config)
		} else {
			err = runAction(config)
		}

		if err != nil {
			if err == huh.ErrUserAborted {
				continue
			}
			fmt.Println(errorStyle.Render(fmt.Sprintf("Execution Failed: %v", err)))
		} else {
			if config.Module != "LAN" {
				fmt.Println(successStyle.Render("Success! Task completed successfully."))
			} else {
				fmt.Println(successStyle.Render("LAN Sharing stopped successfully."))
			}
		}

		// Wait before looping back
		fmt.Print("Press Enter to return to main menu...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		fmt.Println()
	}
}

// showMainMenu handles the top-level form wizard
func showMainMenu(config *AppConfig) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Module").
				Options(
					huh.NewOption("📄 PDF Production Master", "PDF"),
					huh.NewOption("🖼️  Image Lab", "Image"),
					huh.NewOption("📦 Smart Archive Center", "Archive"),
					huh.NewOption("🔍 OCR Master (Text Extract)", "OCR"),
					huh.NewOption("📸 Screen Master (Pro Capture)", "Screen"),
					huh.NewOption("📡 LAN File Sharing", "LAN"),
					huh.NewOption("🚪 Quit", "Quit"),
				).
				Value(&config.Module),
		),
	)

	err := form.Run()
	if err != nil {
		return err
	}

	if config.Module == "Quit" {
		return nil
	}

	// Dynamic sub-menus based on Module
	switch config.Module {
	case "PDF":
		return showPDFMenu(config)
	case "Image":
		return showImageMenu(config)
	case "Archive":
		return showArchiveMenu(config)
	case "LAN":
		return showLANMenu(config)
	case "OCR":
		return showOCRMenu(config)
	case "Screen":
		return showScreenMenu(config)
	}

	return nil
}

func showPDFMenu(config *AppConfig) error {
	// 1. Select Action
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("PDF Actions").
				Options(
					huh.NewOption("Merge PDFs", "Merge"),
					huh.NewOption("Split PDF", "Split"),
					huh.NewOption("Compress PDF", "Compress"),
					huh.NewOption("Convert PDF to Image", "Convert"),
					huh.NewOption("Images to PDF", "ImgToPDF"),
				).
				Value(&config.Action),
		),
	).Run()
	if err != nil {
		return err
	}

	// 2. Capture Input Files (with Finder option)
	inputTitle := "Input PDF File (Leave blank to select via Finder)"
	dialogType := "file"
	if config.Action == "Merge" || config.Action == "ImgToPDF" {
		inputTitle = "Input Files (Comma-separated, or leave blank for Finder)"
		dialogType = "files"
	}

	config.InputFiles = ""
	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(inputTitle).Value(&config.InputFiles),
	)).Run()
	if err != nil {
		return err
	}

	if strings.TrimSpace(config.InputFiles) == "" {
		paths, err := openFinder(dialogType)
		if err != nil {
			return err
		}
		config.InputFiles = paths
	}

	// 3. Output Configuration
	config.OutputFile = generateDefaultOutput(config.InputFiles, config.Action, "")
	var fields []huh.Field

	if config.Action == "Split" {
		config.PageRange = "1"
		fields = append(fields, huh.NewInput().Title("Page Range (e.g., '1-5, 8')").Value(&config.PageRange))
	} else if config.Action == "Compress" {
		config.CompressionMode = "/default"
		fields = append(fields, huh.NewSelect[string]().Title("Compression Level").
			Options(
				huh.NewOption("Balanced", "/default"),
				huh.NewOption("High Quality", "/prepress"),
				huh.NewOption("Extreme Compression", "/screen"),
			).Value(&config.CompressionMode))
	}

	fields = append(fields, huh.NewInput().Title("Output File/Directory").Value(&config.OutputFile))

	return huh.NewForm(huh.NewGroup(fields...)).Run()
}

func showImageMenu(config *AppConfig) error {
	// 1. Select Action
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Image Actions").
				Options(
					huh.NewOption("Convert to WebP", "WebP"),
					huh.NewOption("Smart Resize", "Resize"),
					huh.NewOption("Strip Metadata (Privacy)", "Strip"),
				).
				Value(&config.Action),
		),
	).Run()
	if err != nil {
		return err
	}

	// 2. Capture Input Image
	config.InputFiles = ""
	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Input Image (Leave blank to select via Finder)").Value(&config.InputFiles),
	)).Run()
	if err != nil {
		return err
	}

	if strings.TrimSpace(config.InputFiles) == "" {
		paths, err := openFinder("file")
		if err != nil {
			return err
		}
		config.InputFiles = paths
	}

	// 3. Output Configuration
	config.OutputFile = generateDefaultOutput(config.InputFiles, config.Action, "")
	var fields []huh.Field

	if config.Action == "Resize" {
		config.ImageWidth = "800"
		fields = append(fields, huh.NewInput().Title("Target Width (px)").Value(&config.ImageWidth))
	}
	fields = append(fields, huh.NewInput().Title("Output Image").Value(&config.OutputFile))

	return huh.NewForm(huh.NewGroup(fields...)).Run()
}

func showArchiveMenu(config *AppConfig) error {
	config.Action = "Archive"
	// 1. Archive Format
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Archive Format").
				Options(
					huh.NewOption("Standard ZIP (.zip)", "zip"),
					huh.NewOption("Linux-Friendly TarGz (.tar.gz)", "tar.gz"),
					huh.NewOption("High Compression (.7z)", "7z"),
				).
				Value(&config.ArchiveFormat),
		),
	).Run()
	if err != nil {
		return err
	}

	// 2. Target File/Folder
	config.InputFiles = ""
	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Target Folder/File (Leave blank to select via Finder)").Value(&config.InputFiles),
	)).Run()
	if err != nil {
		return err
	}

	if strings.TrimSpace(config.InputFiles) == "" {
		paths, err := openFinder("file_or_folder")
		if err != nil {
			return err
		}
		config.InputFiles = paths
	}

	// 3. Output Configuration
	config.OutputFile = generateDefaultOutput(config.InputFiles, config.Action, config.ArchiveFormat)
	config.ArchivePassword = ""
	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Output Archive Name").Value(&config.OutputFile),
		huh.NewInput().Title("Password (Leave empty for none)").EchoMode(huh.EchoModePassword).Value(&config.ArchivePassword),
	)).Run()

	return err
}

// openFinder triggers a macOS native AppleScript prompt
func openFinder(dialogType string) (string, error) {
	var script string
	switch dialogType {
	case "files":
		script = `
		set theFiles to choose file with prompt "Select files" with multiple selections allowed
		set pathList to ""
		repeat with aFile in theFiles
			set pathList to pathList & POSIX path of aFile & ","
		end repeat
		if (count of pathList) > 0 then
			set pathList to text 1 thru -2 of pathList
		end if
		return pathList
		`
	case "file_or_folder":
		script = `
		set dialogResult to display dialog "What do you want to select?" buttons {"File", "Folder", "Cancel"} default button "Folder"
		if button returned of dialogResult is "Folder" then
			return POSIX path of (choose folder with prompt "Select a folder")
		else
			return POSIX path of (choose file with prompt "Select a file")
		end if
		`
	default: // "file"
		script = `return POSIX path of (choose file with prompt "Select a file")`
	}

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return "", huh.ErrUserAborted
	}
	return strings.TrimSpace(string(out)), nil
}

// generateDefaultOutput guesses a smart default output path based on the input
func generateDefaultOutput(input string, action string, format string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	// Handle multiple paths safely
	paths := strings.Split(input, ",")
	firstPath := cleanPath(paths[0])
	ext := filepath.Ext(firstPath)
	base := strings.TrimSuffix(firstPath, ext)

	switch action {
	case "Merge":
		return base + "_merged.pdf"
	case "ImgToPDF":
		return base + ".pdf"
	case "Split":
		return base + "_split.pdf"
	case "Compress":
		return base + "_compressed.pdf"
	case "Convert":
		return base + "_converted.png"
	case "WebP":
		return base + ".webp"
	case "Resize":
		return base + "_resized" + ext
	case "Strip":
		return base + "_stripped" + ext
	case "Archive":
		if format == "" {
			format = "zip"
		}
		return base + "." + format
	}
	return base + "_out" + ext
}

// cleanPath sanitizes file paths dragged and dropped into the terminal
func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "'") && strings.HasSuffix(p, "'") {
		p = p[1 : len(p)-1]
	} else if strings.HasPrefix(p, "\"") && strings.HasSuffix(p, "\"") {
		p = p[1 : len(p)-1]
	}
	// macOS drag-and-drop escapes spaces with backslashes
	p = strings.ReplaceAll(p, "\\ ", " ")
	return strings.TrimSpace(p)
}

func runAction(config *AppConfig) error {
	var actionErr error
	actionFunc := func() {
		switch config.Module {
		case "PDF":
			actionErr = handlePDF(config)
		case "Image":
			actionErr = handleImage(config)
		case "Archive":
			actionErr = handleArchive(config)
		case "OCR":
			actionErr = handleOCR(config)
		case "Screen":
			actionErr = handleScreen(config)
		default:
			actionErr = fmt.Errorf("unknown module %s", config.Module)
		}
	}

	err := spinner.New().Title("Processing...").Action(actionFunc).Run()
	if err != nil {
		return err
	}
	return actionErr
}

func handlePDF(config *AppConfig) error {
	out := cleanPath(config.OutputFile)

	switch config.Action {
	case "Merge":
		rawPaths := strings.Split(config.InputFiles, ",")
		var paths []string
		for _, p := range rawPaths {
			paths = append(paths, cleanPath(p))
		}

		swiftScript := `
	import Quartz
	import Foundation

	let args = CommandLine.arguments
	if args.count < 3 { exit(1) }

	let outDoc = PDFDocument()
	var pageIndex = 0

	for i in 2..<args.count {
	let inURL = URL(fileURLWithPath: args[i])
	guard let doc = PDFDocument(url: inURL) else { continue }

	for p in 0..<doc.pageCount {
	guard let page = doc.page(at: p) else { continue }
	outDoc.insert(page, at: pageIndex)
	pageIndex += 1
	}
	}
	outDoc.write(to: URL(fileURLWithPath: args[1]))
	`
		cmd := exec.Command("swift")
		cmd.Args = append([]string{"swift", "-", out}, paths...)
		cmd.Stdin = strings.NewReader(swiftScript)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("macOS native merge failed: %v", err)
		}
		return nil

	case "ImgToPDF":
		rawPaths := strings.Split(config.InputFiles, ",")
		var paths []string
		for _, p := range rawPaths {
			paths = append(paths, cleanPath(p))
		}

		swiftScript := `
	import Quartz
	import Foundation
	import AppKit

	let args = CommandLine.arguments
	if args.count < 3 { exit(1) }

	let outDoc = PDFDocument()
	var pageIndex = 0

	for i in 2..<args.count {
	let inURL = URL(fileURLWithPath: args[i])
	guard let image = NSImage(contentsOfFile: args[i]) else { continue }
	guard let page = PDFPage(image: image) else { continue }
	outDoc.insert(page, at: pageIndex)
	pageIndex += 1
	}
	outDoc.write(to: URL(fileURLWithPath: args[1]))
	`
		cmd := exec.Command("swift")
		cmd.Args = append([]string{"swift", "-", out}, paths...)
		cmd.Stdin = strings.NewReader(swiftScript)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("macOS native image to PDF failed: %v", err)
		}
		return nil
	case "Split":
		in := cleanPath(config.InputFiles)

		swiftScript := `
import Quartz
import Foundation

let args = CommandLine.arguments
if args.count < 4 { exit(1) }
let inURL = URL(fileURLWithPath: args[1])
let outURL = URL(fileURLWithPath: args[2])
let rangeStr = args[3]

guard let doc = PDFDocument(url: inURL) else { exit(1) }
let outDoc = PDFDocument()
var outIdx = 0

let parts = rangeStr.split(separator: ",")
for part in parts {
    let p = part.trimmingCharacters(in: .whitespaces)
    if p.contains("-") {
        let bounds = p.split(separator: "-")
        if bounds.count == 2, let start = Int(bounds[0]), let end = Int(bounds[1]) {
            for i in start...end {
                if i >= 1 && i <= doc.pageCount {
                    guard let page = doc.page(at: i - 1) else { continue }
                    outDoc.insert(page, at: outIdx)
                    outIdx += 1
                }
            }
        }
    } else {
        if let i = Int(p), i >= 1 && i <= doc.pageCount {
            guard let page = doc.page(at: i - 1) else { continue }
            outDoc.insert(page, at: outIdx)
            outIdx += 1
        }
    }
}
outDoc.write(to: outURL)
`
		cmd := exec.Command("swift", "-", in, out, config.PageRange)
		cmd.Stdin = strings.NewReader(swiftScript)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("macOS native split failed: %v", err)
		}
		return nil

	case "Compress":
		in := cleanPath(config.InputFiles)
		_, err := exec.LookPath("gs")
		if err == nil {
			cmd := exec.Command("gs", "-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.4",
				fmt.Sprintf("-dPDFSETTINGS=%s", config.CompressionMode),
				"-dNOPAUSE", "-dQUIET", "-dBATCH",
				fmt.Sprintf("-sOutputFile=%s", out), in)
			return cmd.Run()
		}
		return fmt.Errorf("ghostscript (gs) is required for compression. Please run: brew install ghostscript")

	case "Convert":
		in := cleanPath(config.InputFiles)
		cmd := exec.Command("sips", "-s", "format", "png", in, "--out", out)
		return cmd.Run()
	}
	return nil
}

func handleImage(config *AppConfig) error {
	in := cleanPath(config.InputFiles)
	out := cleanPath(config.OutputFile)

	switch config.Action {
	case "WebP":
		cmd := exec.Command("sips", "-s", "format", "webp", in, "--out", out)
		return cmd.Run()
	case "Resize":
		cmd := exec.Command("sips", "--resampleWidth", config.ImageWidth, in, "--out", out)
		return cmd.Run()
	case "Strip":
		// Strip EXIF profile
		cmd := exec.Command("sips", "-d", "profile", "--deleteColorManagementProperties", in, "--out", out)
		return cmd.Run()
	}
	return nil
}

func handleArchive(config *AppConfig) error {
	in := cleanPath(config.InputFiles)
	out := cleanPath(config.OutputFile)

	var cmd *exec.Cmd

	switch config.ArchiveFormat {
	case "zip":
		args := []string{"-r", "-X", "-x", "*.DS_Store", "-x", "__MACOSX/*"}
		if config.ArchivePassword != "" {
			args = append(args, "-e", "-P", config.ArchivePassword)
		}
		args = append(args, out, filepath.Base(in))
		cmd = exec.Command("zip", args...)
		cmd.Dir = filepath.Dir(in) // run in parent to avoid full absolute paths

	case "tar.gz":
		// Tar doesn't support built-in password easily, so we just compress
		args := []string{"-czvf", out, "--exclude=.DS_Store", "--exclude=__MACOSX"}
		args = append(args, filepath.Base(in))
		cmd = exec.Command("tar", args...)
		cmd.Dir = filepath.Dir(in)

	case "7z":
		// Requires p7zip to be installed
		_, err := exec.LookPath("7z")
		if err != nil {
			return fmt.Errorf("7z command not found. Please install via Homebrew: brew install p7zip")
		}
		args := []string{"a", out, in, "-xr!*.DS_Store", "-xr!__MACOSX"}
		if config.ArchivePassword != "" {
			args = append(args, fmt.Sprintf("-p%s", config.ArchivePassword))
		}
		cmd = exec.Command("7z", args...)
	}

	if cmd != nil {
		outStr, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("archive error: %v, output: %s", err, string(outStr))
		}
		return nil
	}

	return fmt.Errorf("unsupported format")
}

func showLANMenu(config *AppConfig) error {
	config.Action = "LAN"
	config.LANPort = "8080"
	config.InputFiles = ""
	config.LANHTTPS = false
	config.LANTunnel = false

	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Target Folder/File (Leave blank to select via Finder)").Value(&config.InputFiles),
		huh.NewInput().Title("Port").Value(&config.LANPort),
		huh.NewConfirm().Title("Enable HTTPS (May trigger Safari warning)").Value(&config.LANHTTPS),
		huh.NewConfirm().Title("Enable Public Tunnel (Bypass Campus/AP Isolation)").Value(&config.LANTunnel),
	)).Run()
	if err != nil {
		return err
	}

	if strings.TrimSpace(config.InputFiles) == "" {
		paths, err := openFinder("file_or_folder")
		if err != nil {
			return err
		}
		config.InputFiles = paths
	}
	return nil
}

func getLocalIP() (string, string) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1", ""
	}

	var fallbackIPv4 string
	var bestIPv4 string
	var bestIPv6 string

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		
		name := iface.Name
		if strings.HasPrefix(name, "utun") || strings.HasPrefix(name, "awdl") || strings.HasPrefix(name, "bridge") || strings.HasPrefix(name, "llw") || strings.HasPrefix(name, "ipsec") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				ip := ipnet.IP
				if ip.To4() != nil {
					if strings.HasPrefix(name, "en") && bestIPv4 == "" {
						bestIPv4 = ip.String()
					}
					if fallbackIPv4 == "" {
						fallbackIPv4 = ip.String()
					}
				} else if ip.To16() != nil && !ip.IsLinkLocalUnicast() {
					// We found a Global Unicast IPv6 address!
					if strings.HasPrefix(name, "en") && bestIPv6 == "" {
						bestIPv6 = ip.String()
					}
				}
			}
		}
	}

	finalIPv4 := bestIPv4
	if finalIPv4 == "" {
		finalIPv4 = fallbackIPv4
	}
	if finalIPv4 == "" {
		finalIPv4 = "127.0.0.1"
	}

	return finalIPv4, bestIPv6
}

func generateTLSCert(ipStr string) (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"macOS Toolkit Local Share"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	
	if parsedIP := net.ParseIP(ipStr); parsedIP != nil {
		template.IPAddresses = []net.IP{parsedIP}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}

func handleLANSharing(config *AppConfig) error {
	in := cleanPath(config.InputFiles)
	port := config.LANPort
	if port == "" {
		port = "8080"
	}

	info, err := os.Stat(in)
	if err != nil {
		return fmt.Errorf("invalid path: %v", err)
	}

	ipv4, ipv6 := getLocalIP()
	schema := "http"
	if config.LANHTTPS {
		schema = "https"
	}

	mux := http.NewServeMux()
	if info.IsDir() {
		mux.Handle("/", http.FileServer(http.Dir(in)))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(in)))
			http.ServeFile(w, r, in)
		})
	}

	server := &http.Server{Addr: ":" + port, Handler: mux}
	if config.LANHTTPS {
		cert, _ := generateTLSCert(ipv4)
		server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("port %s blocked", port)
	}

	go func() {
		if config.LANHTTPS {
			_ = server.ServeTLS(listener, "", "")
		} else {
			_ = server.Serve(listener)
		}
	}()

	// --- Smart Selection & Tunnel Logic ---
	var tunnelURL string
	var tunnelCmd *exec.Cmd
	if config.LANTunnel {
		tunnelCmd = exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-R", "80:localhost:"+port, "nokey@localhost.run")
		stdout, _ := tunnelCmd.StdoutPipe()
		tunnelCmd.Start()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "https://") {
				words := strings.Fields(line)
				for _, w := range words {
					if strings.HasPrefix(w, "https://") {
						tunnelURL = strings.Trim(w, ".")
						break
					}
				}
				if tunnelURL != "" {
					break
				}
			}
		}
	}

	// Determine the BEST URL to show
	var bestURL string
	var modeLabel string
	
	if tunnelURL != "" {
		bestURL = tunnelURL
		modeLabel = "🚀 Public Tunnel"
	} else if ipv6 != "" && strings.HasPrefix(ipv4, "172.") { // Smart detection for campus subnets
		bestURL = fmt.Sprintf("%s://[%s]:%s/", schema, ipv6, port)
		modeLabel = "📡 IPv6 Direct"
	} else {
		bestURL = fmt.Sprintf("%s://%s:%s/", schema, ipv4, port)
		modeLabel = "🏠 Local Network"
	}

	qr, _ := qrcode.New(bestURL, qrcode.Medium)
	qrStr := qr.ToSmallString(false)

	// --- Minimalist UI ---
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 4).
		Align(lipgloss.Center)

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Underline(true)
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("62")).
		Padding(0, 1)

	dashboard := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("SHARE READY"),
		badgeStyle.Render(modeLabel),
		qrStr,
		"",
		urlStyle.Render(bestURL),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(filepath.Base(in)),
	)

	fmt.Println("\n" + cardStyle.Render(dashboard) + "\n")

	var confirm bool
	huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Stop Sharing?").Value(&confirm))).Run()

	server.Shutdown(context.Background())
	if tunnelCmd != nil && tunnelCmd.Process != nil {
		tunnelCmd.Process.Kill()
	}
	return nil
}

func showOCRMenu(config *AppConfig) error {
	config.Action = "OCR"
	config.OCRMode = "File"
	config.InputFiles = ""

	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("OCR Source").
			Options(
				huh.NewOption("Select Image File", "File"),
				huh.NewOption("Capture Screen Area", "Capture"),
			).Value(&config.OCRMode),
	)).Run()
	if err != nil {
		return err
	}

	if config.OCRMode == "File" {
		err = huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Input Image (Leave blank for Finder)").Value(&config.InputFiles),
		)).Run()
		if err != nil {
			return err
		}

		if strings.TrimSpace(config.InputFiles) == "" {
			paths, err := openFinder("file")
			if err != nil {
				return err
			}
			config.InputFiles = paths
		}
	}
	return nil
}

func handleOCR(config *AppConfig) error {
	var in string
	var isTemp bool

	if config.OCRMode == "Capture" {
		tempPath := filepath.Join(os.TempDir(), "ocr_capture_"+time.Now().Format("20060102_150405")+".png")
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render("\n📸 Drag to select text area..."))
		cmd := exec.Command("screencapture", "-i", tempPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("capture cancelled")
		}
		in = tempPath
		isTemp = true
	} else {
		in = cleanPath(config.InputFiles)
	}

	if isTemp {
		defer os.Remove(in)
	}

	// Swift script using Vision Framework
	swiftScript := `
import Vision
import AppKit

let args = CommandLine.arguments
if args.count < 2 { exit(1) }
let imagePath = args[1]

guard let image = NSImage(contentsOfFile: imagePath),
      let cgImage = image.cgImage(forProposedRect: nil, context: nil, hints: nil) else {
    exit(1)
}

let requestHandler = VNImageRequestHandler(cgImage: cgImage, options: [:])
let request = VNRecognizeTextRequest { (request, error) in
    guard let observations = request.results as? [VNRecognizedTextObservation] else { return }
    let recognizedStrings = observations.compactMap { $0.topCandidates(1).first?.string }
    print(recognizedStrings.joined(separator: "\n"))
}

request.recognitionLevel = .accurate
request.recognitionLanguages = ["zh-Hans", "en-US"]

do {
    try requestHandler.perform([request])
} catch {
    exit(1)
}
`
	cmd := exec.Command("swift", "-", in)
	cmd.Stdin = strings.NewReader(swiftScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("OCR failed: %v, output: %s", err, string(out))
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return fmt.Errorf("no text detected in image")
	}

	// Copy to clipboard using pbcopy
	pbCmd := exec.Command("pbcopy")
	pbCmd.Stdin = strings.NewReader(result)
	_ = pbCmd.Run()

	// Display snippet
	fmt.Println(lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true).
		Padding(1).
		Render("\n✅ Text extracted and copied to clipboard!"))

	// Show a preview of the text
	preview := result
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	fmt.Println(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(preview))

	return nil
}

func showScreenMenu(config *AppConfig) error {
	config.Action = "Capture"
	config.FilterType = "None"
	config.OutputFile = filepath.Join(os.Getenv("HOME"), "Desktop", "Screenshot_"+time.Now().Format("20060102_150405")+".png")

	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Post-Processing Effect").
			Options(
				huh.NewOption("Original (No Filter)", "None"),
				huh.NewOption("Privacy Blur (Gaussian)", "Blur"),
				huh.NewOption("Classic Grayscale", "Mono"),
				huh.NewOption("Vintage Sepia", "Sepia"),
				huh.NewOption("High Contrast (Chrome)", "Chrome"),
			).Value(&config.FilterType),
		huh.NewInput().Title("Save Destination").Value(&config.OutputFile),
	)).Run()

	return err
}

func handleScreen(config *AppConfig) error {
	out := cleanPath(config.OutputFile)
	outDir := filepath.Dir(out)
	os.MkdirAll(outDir, 0755)

	// Step 1: Trigger native interactive capture
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render("\n📸 Drag your mouse to select an area..."))
	cmd := exec.Command("screencapture", "-i", out)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("capture cancelled or failed")
	}

	// Step 2: Apply Filters via Swift CoreImage if requested
	if config.FilterType != "None" {
		swiftScript := `
import Quartz
import CoreImage
import AppKit

let args = CommandLine.arguments
let path = args[1]
let effect = args[2]

guard let image = NSImage(contentsOfFile: path),
      let tiff = image.tiffRepresentation,
      let ciImage = CIImage(data: tiff) else { exit(1) }

var filter: CIFilter?

switch effect {
case "Blur":
    filter = CIFilter(name: "CIGaussianBlur", parameters: ["inputRadius": 10.0])
case "Mono":
    filter = CIFilter(name: "CIPhotoEffectMono")
case "Sepia":
    filter = CIFilter(name: "CISepiaTone", parameters: ["inputIntensity": 1.0])
case "Chrome":
    filter = CIFilter(name: "CIPhotoEffectChrome")
default:
    break
}

if let filter = filter {
    filter.setValue(ciImage, forKey: kCIInputImageKey)
    if let output = filter.outputImage {
        let rep = NSBitmapImageRep(ciImage: output)
        if let data = rep.representation(using: .png, properties: [:]) {
            try? data.write(to: URL(fileURLWithPath: path))
        }
    }
}
`
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Render("🧪 Applying " + config.FilterType + " effect..."))
		filterCmd := exec.Command("swift", "-", out, config.FilterType)
		filterCmd.Stdin = strings.NewReader(swiftScript)
		if err := filterCmd.Run(); err != nil {
			return fmt.Errorf("failed to apply filter: %v", err)
		}
	}

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Render("\n✅ Screenshot saved: " + out))
	return nil
}

