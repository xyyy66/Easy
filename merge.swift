import Quartz
import Foundation

let args = CommandLine.arguments
if args.count < 3 {
    print("Usage: merge <output.pdf> <input1.pdf> <input2.pdf> ...")
    exit(1)
}

let outPath = args[1]
let outURL = URL(fileURLWithPath: outPath)

guard let ctx = CGContext(outURL as CFURL, mediaBox: nil, nil) else {
    print("Cannot create output context")
    exit(1)
}

for i in 2..<args.count {
    let inURL = URL(fileURLWithPath: args[i])
    guard let doc = CGPDFDocument(inURL as CFURL) else {
        print("Cannot open \(args[i])")
        continue
    }
    
    let pages = doc.numberOfPages
    for p in 1...pages {
        guard let page = doc.page(at: p) else { continue }
        var mediaBox = page.getBoxRect(.mediaBox)
        ctx.beginPage(mediaBox: &mediaBox)
        ctx.drawPDFPage(page)
        ctx.endPage()
    }
}
ctx.closePDF()
