import AppKit

var args = Array(CommandLine.arguments.dropFirst())
if let idx = args.firstIndex(of: "--") {
    args = Array(args.dropFirst(idx + 1))
}

guard !args.isEmpty else {
    fputs("usage: \(CommandLine.arguments[0]) -- /abs/path [/abs/path ...]\n", stderr)
    exit(1)
}

let pb = NSPasteboard.general
pb.clearContents()
var urls: [NSURL] = []
for p in args {
    urls.append(NSURL(fileURLWithPath: p, isDirectory: false))
}
if !pb.writeObjects(urls) {
    fputs("NSPasteboard.writeObjects failed\n", stderr)
    exit(1)
}
