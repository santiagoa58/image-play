# Word-cloud baseline

This baseline is the successful `main` CI run at commit
`af9b3333525bd50d38242d6a23e003dd78c4cf0d`:
[run 36280357693](https://github.com/santiagoa58/image-play/actions/runs/36280357693).
The run generated each image with `testdata/text/sample_text_message.txt`,
`fonts/NotoSansMono-VariableFont_wdth,wght.ttf`, and the default word-cloud
configuration. Its [visual outputs](https://github.com/santiagoa58/image-play/actions/runs/36280357693/artifacts/10918393968)
are retained by GitHub Actions for seven days.

| Input | Dimensions | Automatic max | Prepared | Placed | Skipped | Horizontal | Vertical | Placement time | Total time |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `couple_tour.jpg` | 768 × 1152 | 105 px | 500 | 345 | 155 | 184 | 161 | 20.15 s | 22.87 s |
| `deepseek-logo-icon.png` | 512 × 512 | 44 px | 500 | 95 | 405 | 80 | 15 | 10.62 s | 11.43 s |
| `gen-img-couple.png` | 1024 × 1024 | 112 px | 500 | 432 | 68 | 345 | 87 | 9.43 s | 11.64 s |
| `keeks_no_bckgrnd.png` | 433 × 577 | 71 px | 500 | 80 | 420 | 62 | 18 | 50.91 s | 52.27 s |

The minimum font size was 6 px for every image. Times are from a single CI run,
so use them as a rough comparison, not a performance target with statistical
confidence. The current skip counts may include words that would have fit at
unsampled positions; the existing search stops after a configured number of
spiral steps.

## Comparison procedure

Run the same four inputs, text, font, and candidate limit on the replacement
branch. Compare placed and skipped counts, actual font-size hierarchy,
horizontal and vertical counts, region coverage, shape fidelity, and time.
Inspect the PNG outputs at their native dimensions as well as resized previews.
CI's seven-day artifact retention means the baseline visual outputs may need
to be regenerated from this commit for a later side-by-side review.

The current local workspace has OpenCV 4.6, which cannot compile the repository's
GoCV 0.43 bindings. CI builds against a newer OpenCV version through the
repository's Dockerfile. The visual outputs were not inspected locally for this
baseline; appearance judgments remain pending a reproducible compatible run.
