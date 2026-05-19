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
