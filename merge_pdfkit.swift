import Quartz
import Foundation

let args = CommandLine.arguments
if args.count < 3 {
    exit(1)
}

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
