# Image-decoding security

Both effects accept image files. Decoder dependencies therefore matter even
when the usual inputs are trusted photographs or PNGs.

## Bounded TIFF decompression

`golang.org/x/image` is pinned to v0.43.0 or newer. Versions before v0.41.0 could
expand PackBits-compressed TIFF data without an adequate output limit, consuming
excessive resources for a small crafted image. The upstream patch bounds this
decompression. This addresses Dependabot alert #7, CVE-2026-46599 / GO-2026-5032.
See the [Go vulnerability report](https://pkg.go.dev/vuln/GO-2026-5032) and
[Go Security Team announcement](https://groups.google.com/g/golang-announce/c/uhYX90BlBvI).

The v0.43.0 upgrade also fixes two additional reachable TIFF decoder flaws found
by `govulncheck`: [unbounded tile sizes, GO-2026-5062](https://pkg.go.dev/vuln/GO-2026-5062)
and [invalid strip offsets, GO-2026-5066](https://pkg.go.dev/vuln/GO-2026-5066).
The original Dependabot proposal to use v0.41.0 did not include these fixes.

## Invalid palette indexes

An indexed-color image stores a palette of colors and a palette index for each
pixel. An index beyond the palette length is invalid. `imaging` v1.6.2 can panic
when its concurrent scanner encounters such an index; recovering in its caller
would not catch a panic in a worker goroutine. This is the condition reported
in Dependabot alert #4 / CVE-2023-36308.

The text-mosaic pipeline checks every visible pixel index in a paletted source
before calling any imaging resize, contrast, or grayscale operation. Invalid
indexes return an error. Validation respects image bounds and stride, including
subimages, and ignores pixels outside the source rectangle. Other image types
do not need this palette check. The word-cloud path uses OpenCV and does not
pass source images to the imaging scanner.

The modern Go TIFF decoder also rejects malformed palette indexes; the upstream
parser issue links the same published reproducer. Our validation provides a
second boundary for directly supplied paletted images. See the
[original imaging report](https://github.com/disintegration/imaging/issues/165),
[Go TIFF parser fix](https://github.com/golang/go/issues/67624), and
[proposed imaging scanner fix](https://github.com/disintegration/imaging/pull/180).

The [GitHub advisory](https://github.com/advisories/GHSA-q7pp-wcgr-pffx) lists
no patched imaging release. Keeping imaging v1.6.2 can therefore continue to
trigger a version-based alert even with these input boundaries. Any dismissal
should record this mitigation and its regression tests, rather than claim that
the imaging scanner itself has been patched.

## Verification and limits

Regression tests exercise invalid palette data before each image-preparation
operation and confirm that valid paletted subimages still render. CI also runs
the official Go vulnerability scanner against the source call graph:

```bash
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
```

Run it in the repository's OpenCV toolchain container, as the CI workflow does.
The scanner covers advisories in the Go vulnerability database; it does not
replace review of GitHub-only advisories or audit native OpenCV/system libraries.
These fixes address the documented decoder vulnerabilities; they are not a
general memory or execution-time limit on arbitrary input images.
