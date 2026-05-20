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
	ImgPrefix       string
}

// Theme Colors
var (
	purple = lipgloss.Color("#8839ef")
	cyan   = lipgloss.Color("#179299")
	green  = lipgloss.Color("#40a02b")
	red    = lipgloss.Color("#d20f39")
	gray   = lipgloss.Color("#737994")
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EFF1F5")).
			Background(purple).
			Padding(0, 1).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(gray).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(1, 2).
			Margin(1, 0)

	successBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(green).
			Foreground(green).
			Padding(0, 1).
			Bold(true)

	errorBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(red).
			Foreground(red).
			Padding(0, 1).
			Bold(true)
)

func renderHeader() {
	fmt.Print("\033[H\033[2J") // Clear screen
	t := titleStyle.Render(" Easy: macOS Efficiency Toolkit ")
	d := descStyle.Render(" v1.2 • Pro Suite ")
	fmt.Println("\n  " + t + d + "\n")
}

func renderGradientSuccess(msg string) {
	// Mint to Cyan gradient effect
	mint := lipgloss.Color("#40a02b")
	cyan := lipgloss.Color("#179299")
	
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mint).
		Padding(0, 2).
		Bold(true)
	
	text := lipgloss.NewStyle().Foreground(mint).Render("✓ SUCCESS ") + 
	       lipgloss.NewStyle().Foreground(cyan).Render("• "+msg)
	
	fmt.Println("\n" + style.Render(text))
}

func main() {
	config := &AppConfig{}

	for {
		renderHeader()
		err := showMainMenu(config)
		if err != nil {
			if err == huh.ErrUserAborted {
				fmt.Println("\n  " + descStyle.Render("Session ended. Goodbye!"))
				break
			}
			fmt.Println("\n" + errorBox.Render(fmt.Sprintf("ERROR: %v", err)))
			continue
		}

		if config.Module == "Quit" {
			fmt.Println("\n  " + descStyle.Render("Session ended. Goodbye!"))
			break
		}

		// Routing to sub-menus
		var menuErr error
		switch config.Module {
		case "PDF":
			menuErr = showPDFMenu(config)
		case "Image":
			menuErr = showImageMenu(config)
		case "Archive":
			menuErr = showArchiveMenu(config)
		case "LAN":
			menuErr = showLANMenu(config)
		case "OCR":
			menuErr = showOCRMenu(config)
		case "Screen":
			menuErr = showScreenMenu(config)
		}

		// ESC support: If user aborted sub-menu, go back to main menu
		if menuErr != nil {
			if menuErr == huh.ErrUserAborted {
				continue 
			}
			fmt.Println("\n" + errorBox.Render(fmt.Sprintf("ERROR: %v", menuErr)))
			continue
		}
		
		// If Action is "Back", continue to main menu
		if config.Action == "Back" {
			continue
		}

		// Process user selections
		var runErr error
		if config.Module == "LAN" {
			runErr = handleLANSharing(config)
		} else {
			runErr = runAction(config)
		}

		if runErr != nil {
			if runErr != huh.ErrUserAborted {
				fmt.Println("\n" + errorBox.Render(fmt.Sprintf("FAILED: %v", runErr)))
			}
		} else {
			if config.Module != "LAN" && config.Module != "Quit" {
				renderGradientSuccess("COMPLETED SUCCESSFULLY")
			}
		}

		fmt.Print("\n  " + descStyle.Render("Press Enter to continue..."))
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
}

// showMainMenu handles the top-level form wizard
func showMainMenu(config *AppConfig) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Tool").
				Options(
					huh.NewOption("📄 PDF Operation", "PDF"),
					huh.NewOption("🖼️ Image Lab", "Image"),
					huh.NewOption("📦 Smart Archive", "Archive"),
					huh.NewOption("🔍 OCR Master", "OCR"),
					huh.NewOption("📸 ScreenShot", "Screen"),
					huh.NewOption("📡 LAN Sharing", "LAN"),
					huh.NewOption("🚪 Quit", "Quit"),
				).
				Value(&config.Module),
		),
	)

	return form.Run()
}

func showPDFMenu(config *AppConfig) error {
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("PDF Actions").
				Options(
					huh.NewOption("Merge Files", "Merge"),
					huh.NewOption("Split Pages", "Split"),
					huh.NewOption("Optimize Size", "Compress"),
					huh.NewOption("Export to Image", "Convert"),
					huh.NewOption("Images to PDF", "ImgToPDF"),
					huh.NewOption("Extract Images", "Extract"),
					huh.NewOption("<- Back", "Back"),
				).
				Value(&config.Action),
		),
	).Run()
	if err != nil {
		return err
	}
	if config.Action == "Back" {
		return nil
	}

	inputTitle := "Select PDF"
	dialogType := "file"
	if config.Action == "Merge" || config.Action == "ImgToPDF" {
		inputTitle = "Select Files"
		dialogType = "files"
	}

	config.InputFiles = ""
	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(inputTitle).Placeholder("Drag here or press Enter for Finder").Value(&config.InputFiles),
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

	config.OutputFile = generateDefaultOutput(config.InputFiles, config.Action, "")
	config.ImgPrefix = "img"
	var fields []huh.Field

	if config.Action == "Split" {
		config.PageRange = "1"
		fields = append(fields, huh.NewInput().Title("Page Range").Placeholder("e.g. 1-5, 8").Value(&config.PageRange))
	} else if config.Action == "Extract" {
		config.PageRange = ""
		fields = append(fields, 
			huh.NewInput().Title("Page Range").Placeholder("Leave blank for all, or e.g. 1-5").Value(&config.PageRange),
			huh.NewInput().Title("Image Prefix").Value(&config.ImgPrefix),
		)
	} else if config.Action == "Compress" {
		config.CompressionMode = "/default"
		fields = append(fields, huh.NewSelect[string]().Title("Level").
			Options(
				huh.NewOption("Balanced", "/default"),
				huh.NewOption("High Quality", "/prepress"),
				huh.NewOption("Maximum", "/screen"),
			).Value(&config.CompressionMode))
	}

	fields = append(fields, huh.NewInput().Title("Save To").Value(&config.OutputFile))

	return huh.NewForm(huh.NewGroup(fields...)).Run()
}

func showImageMenu(config *AppConfig) error {
	config.FilterType = "None"
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Image Lab").
				Options(
					huh.NewOption("Modern WebP", "WebP"),
					huh.NewOption("Fast Resize", "Resize"),
					huh.NewOption("Privacy Strip", "Strip"),
					huh.NewOption("Artistic Filter", "Filter"),
					huh.NewOption("<- Back", "Back"),
				).
				Value(&config.Action),
		),
	).Run()
	if err != nil {
		return err
	}
	if config.Action == "Back" {
		return nil
	}

	if config.Action == "Filter" {
		err = huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().Title("Effect").
				Options(
					huh.NewOption("Gaussian Blur", "Blur"),
					huh.NewOption("Classic Mono", "Mono"),
					huh.NewOption("Vintage Sepia", "Sepia"),
					huh.NewOption("Vibrant Chrome", "Chrome"),
				).Value(&config.FilterType),
		)).Run()
		if err != nil {
			return err
		}
	}

	config.InputFiles = ""
	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Select Image").Placeholder("Drag here or Enter for Finder").Value(&config.InputFiles),
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

	config.OutputFile = generateDefaultOutput(config.InputFiles, config.Action, "")
	var fields []huh.Field

	if config.Action == "Resize" {
		config.ImageWidth = "800"
		fields = append(fields, huh.NewInput().Title("Width (px)").Value(&config.ImageWidth))
	}
	fields = append(fields, huh.NewInput().Title("Save To").Value(&config.OutputFile))

	return huh.NewForm(huh.NewGroup(fields...)).Run()
}

func showArchiveMenu(config *AppConfig) error {
	config.Action = "Archive"
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Format").
				Options(
					huh.NewOption("ZIP (Standard)", "zip"),
					huh.NewOption("Tar.gz (Unix)", "tar.gz"),
					huh.NewOption("7Z (Maximum)", "7z"),
					huh.NewOption("<- Back", "Back"),
				).
				Value(&config.ArchiveFormat),
		),
	).Run()
	if err != nil {
		return err
	}

	if config.ArchiveFormat == "Back" {
		config.Action = "Back"
		return nil
	}

	config.InputFiles = ""
	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Select Items").Placeholder("Drag files/folders or Enter for Finder").Value(&config.InputFiles),
	)).Run()
	if err != nil {
		return err
	}

	if strings.TrimSpace(config.InputFiles) == "" {
		var selectionType string
		err = huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Type").
				Options(
					huh.NewOption("Files", "files"),
					huh.NewOption("Folders", "folders"),
					huh.NewOption("<- Back", "Back"),
				).
				Value(&selectionType),
		)).Run()
		if err != nil {
			return err
		}
		if selectionType == "Back" {
			config.Action = "Back"
			return nil
		}

		paths, err := openFinder(selectionType)
		if err != nil {
			return err
		}
		config.InputFiles = paths
	}

	config.OutputFile = generateDefaultOutput(config.InputFiles, config.Action, config.ArchiveFormat)
	config.ArchivePassword = ""
	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Archive Name").Value(&config.OutputFile),
		huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Placeholder("Optional").Value(&config.ArchivePassword),
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
	case "folders":
		script = `
		set theFolders to choose folder with prompt "Select folders" with multiple selections allowed
		set pathList to ""
		repeat with aFolder in theFolders
			set pathList to pathList & POSIX path of aFolder & ","
		end repeat
		if (count of pathList) > 0 then
			set pathList to text 1 thru -2 of pathList
		end if
		return pathList
		`
	default: // "file" (single)
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
	case "Extract":
		return base + "_extracted_images"
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

	err := spinner.New().
		Title("Processing Task...").
		Style(lipgloss.NewStyle().Foreground(purple)).
		Action(actionFunc).
		Run()
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

		case "Extract":
		in := cleanPath(config.InputFiles)
		outDir := cleanPath(config.OutputFile)
		os.MkdirAll(outDir, 0755)

		args := []string{"-j", "-png"} // -j: attempt to write JPEGs, -png: fallback/output PNG

		if config.PageRange != "" {
			parts := strings.Split(config.PageRange, "-")
			if len(parts) == 2 {
				args = append(args, "-f", parts[0], "-l", parts[1])
			} else if len(parts) == 1 {
				args = append(args, "-f", parts[0], "-l", parts[0])
			}
		}

		outputPath := filepath.Join(outDir, config.ImgPrefix)
		args = append(args, in, outputPath)

		cmd := exec.Command("pdfimages", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract images: %v. Ensure Poppler is installed: brew install poppler", err)
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
	case "Filter":
		return applyFilter(in, out, config.FilterType)
	}
	return nil
}

func applyFilter(in, out, effect string) error {
	swiftScript := `
import Quartz
import CoreImage
import AppKit

let args = CommandLine.arguments
let inPath = args[1]
let outPath = args[2]
let effect = args[3]

guard let image = NSImage(contentsOfFile: inPath),
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
            try? data.write(to: URL(fileURLWithPath: outPath))
        }
    }
}
`
	cmd := exec.Command("swift", "-", in, out, effect)
	cmd.Stdin = strings.NewReader(swiftScript)
	return cmd.Run()
}

func handleArchive(config *AppConfig) error {
	rawPaths := strings.Split(config.InputFiles, ",")
	var paths []string
	var basePaths []string
	
	dir := ""
	sameDir := true

	for i, p := range rawPaths {
		cleanP := cleanPath(p)
		paths = append(paths, cleanP)
		if i == 0 {
			dir = filepath.Dir(cleanP)
		} else if filepath.Dir(cleanP) != dir {
			sameDir = false
		}
		basePaths = append(basePaths, filepath.Base(cleanP))
	}
	out := cleanPath(config.OutputFile)

	var cmd *exec.Cmd
	var targetPaths []string
	if sameDir {
		targetPaths = basePaths
	} else {
		targetPaths = paths
	}

	switch config.ArchiveFormat {
	case "zip":
		args := []string{"-r", "-X"}
		if config.ArchivePassword != "" {
			args = append(args, "-e", "-P", config.ArchivePassword)
		}
		args = append(args, out)
		args = append(args, targetPaths...)
		// -x must be at the very end of the zip command
		args = append(args, "-x", "*.DS_Store", "-x", "__MACOSX/*")
		cmd = exec.Command("zip", args...)

	case "tar.gz":
		args := []string{"-czvf", out, "--exclude=.DS_Store", "--exclude=__MACOSX"}
		args = append(args, targetPaths...)
		cmd = exec.Command("tar", args...)

	case "7z":
		_, err := exec.LookPath("7z")
		if err != nil {
			return fmt.Errorf("7z command not found. Please install via Homebrew: brew install p7zip")
		}
		args := []string{"a", out, "-xr!*.DS_Store", "-xr!__MACOSX"}
		if config.ArchivePassword != "" {
			args = append(args, fmt.Sprintf("-p%s", config.ArchivePassword))
		}
		args = append(args, targetPaths...)
		cmd = exec.Command("7z", args...)
	}

	if cmd != nil {
		if sameDir {
			cmd.Dir = dir
		}
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
		huh.NewInput().Title("Target").Placeholder("Drag here or Enter for Finder").Value(&config.InputFiles),
		huh.NewInput().Title("Port").Value(&config.LANPort),
		huh.NewConfirm().Title("HTTPS Mode").Value(&config.LANHTTPS),
		huh.NewConfirm().Title("Public Tunnel").Value(&config.LANTunnel),
	)).Run()
	if err != nil {
		return err
	}

	if strings.TrimSpace(config.InputFiles) == "" {
		var selectionType string
		err = huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Type").
				Options(
					huh.NewOption("File", "file"),
					huh.NewOption("Folder", "folders"),
					huh.NewOption("<- Back", "Back"),
				).
				Value(&selectionType),
		)).Run()
		if err != nil {
			return err
		}
		if selectionType == "Back" {
			config.Action = "Back"
			return nil
		}

		paths, err := openFinder(selectionType)
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
		BorderForeground(purple).
		Padding(1, 4).
		Align(lipgloss.Center)

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EFF1F5")).Background(purple).Padding(0, 1).Bold(true)
	urlStyle := lipgloss.NewStyle().Foreground(green).Underline(true)
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EFF1F5")).
		Background(cyan).
		Padding(0, 1)

	dashboard := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("SHARE READY"),
		"",
		badgeStyle.Render(modeLabel),
		qrStr,
		"",
		urlStyle.Render(bestURL),
		"",
		lipgloss.NewStyle().Foreground(gray).Render(filepath.Base(in)),
	)

	fmt.Println("\n" + cardStyle.Render(dashboard) + "\n")

	var confirm bool
	huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Stop Sharing?").Affirmative("End Session").Negative("Stay Online").Value(&confirm))).Run()


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
				huh.NewOption("Existing Image", "File"),
				huh.NewOption("Capture Area", "Capture"),
				huh.NewOption("<- Back", "Back"),
			).Value(&config.OCRMode),
			),
			).Run()
			if err != nil {
			return err
			}

			if config.OCRMode == "Back" {
			config.Action = "Back"
			return nil
			}
	if config.OCRMode == "File" {
		err = huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Select Image").Placeholder("Drag here or Enter for Finder").Value(&config.InputFiles),
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
	renderGradientSuccess("TEXT COPIED TO CLIPBOARD")

	// Show a preview of the text
	preview := result
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	fmt.Println(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(gray).
		Padding(0, 1).
		Foreground(gray).
		Render(preview))

	return nil
	}
func showScreenMenu(config *AppConfig) error {
	config.Action = "Capture"
	config.FilterType = "None"
	config.OutputFile = filepath.Join(os.Getenv("HOME"), "Desktop", "Screenshot_"+time.Now().Format("20060102_150405")+".png")

	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Effect").
			Options(
				huh.NewOption("Natural", "None"),
				huh.NewOption("Privacy Blur", "Blur"),
				huh.NewOption("Mono", "Mono"),
				huh.NewOption("Sepia", "Sepia"),
				huh.NewOption("Chrome", "Chrome"),
				huh.NewOption("<- Back", "Back"),
			).Value(&config.FilterType),
			),
			).Run()

			if err == nil && config.FilterType == "Back" {
				config.Action = "Back"
				return nil
			}
			if err != nil {
				return err
			}

	return huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Save To").Value(&config.OutputFile),
	)).Run()
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
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Render("🧪 Applying " + config.FilterType + " effect..."))
		if err := applyFilter(out, out, config.FilterType); err != nil {
			return fmt.Errorf("failed to apply filter: %v", err)
		}
	}

	renderGradientSuccess("SCREENSHOT SAVED")
	fmt.Println("  " + descStyle.Render(out))
	return nil
}

