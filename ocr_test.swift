import Vision
import AppKit

let args = CommandLine.arguments
if args.count < 2 { exit(1) }
let imagePath = args[1]

guard let image = NSImage(contentsOfFile: imagePath),
      let cgImage = image.cgImage(forProposedRect: nil, context: nil, hints: nil) else {
    print("Error: Could not load image")
    exit(1)
}

let requestHandler = VNImageRequestHandler(cgImage: cgImage, options: [:])
let request = VNRecognizeTextRequest { (request, error) in
    guard let observations = request.results as? [VNRecognizedTextObservation] else { return }
    let recognizedStrings = observations.compactMap { $0.topCandidates(1).first?.string }
    print(recognizedStrings.joined(separator: "\n"))
}

// Set recognition level and languages
request.recognitionLevel = .accurate
request.recognitionLanguages = ["zh-Hans", "en-US"]

do {
    try requestHandler.perform([request])
} catch {
    print("Error: \(error)")
    exit(1)
}
