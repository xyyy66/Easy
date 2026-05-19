#!/usr/bin/env python3
import sys
from Quartz.CoreGraphics import *
from CoreFoundation import *

def merge_pdfs(output_path, input_paths):
    ctx = CGPDFContextCreateWithURL(CFURLCreateFromFileSystemRepresentation(kCFAllocatorDefault, output_path, len(output_path), False), None, None)
    if not ctx:
        print("Failed to create context")
        sys.exit(1)
        
    for path in input_paths:
        pdf_url = CFURLCreateFromFileSystemRepresentation(kCFAllocatorDefault, path, len(path), False)
        doc = CGPDFDocumentCreateWithURL(pdf_url)
        if doc:
            pages = CGPDFDocumentGetNumberOfPages(doc)
            for i in range(1, pages + 1):
                page = CGPDFDocumentGetPage(doc, i)
                if page:
                    var_rect = CGPDFPageGetBoxRect(page, kCGPDFMediaBox)
                    CGContextBeginPage(ctx, var_rect)
                    CGContextDrawPDFPage(ctx, page)
                    CGContextEndPage(ctx)
    CGPDFContextClose(ctx)

if __name__ == "__main__":
    if len(sys.argv) < 3:
        sys.exit(1)
    merge_pdfs(sys.argv[1], sys.argv[2:])
