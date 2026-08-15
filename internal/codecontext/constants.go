package codecontext

// maxScannerLineBytes bounds bufio.Scanner's per-line buffer for both
// snippet and blame parsing. The default (~64KiB, bufio.MaxScanTokenSize)
// is too small for a single minified/generated source line, which would
// otherwise surface as a spurious "token too long" read error on an
// otherwise perfectly normal file. 1MiB is a generous ceiling for any
// real source line without holding the whole file in memory at once.
const maxScannerLineBytes = 1024 * 1024
