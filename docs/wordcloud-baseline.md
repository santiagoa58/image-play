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

## Replacement results

The replacement branch was measured with the same CI workflow at commit
`14edae5` ([run 36298769804](https://github.com/santiagoa58/image-play/actions/runs/36298769804)).
The floor is `max(6 px, round(1% of the shorter image dimension))`; the maximum
still comes from the image and the two most frequent words.

| Input | Automatic min | Automatic max | Placed | Skipped | Horizontal | Vertical | Total time |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `couple_tour.jpg` | 8 px | 106 px | 288 | 212 | 128 | 160 | 1.87 s |
| `deepseek-logo-icon.png` | 6 px | 44 px | 98 | 402 | 82 | 16 | 0.83 s |
| `gen-img-couple.png` | 10 px | 116 px | 267 | 233 | 149 | 118 | 2.32 s |
| `keeks_no_bckgrnd.png` | 6 px | 75 px | 77 | 423 | 41 | 36 | 0.98 s |

The timings are single CI runs on the same workflow, rather than repeated
benchmarks. The new search is much faster on these examples. It places fewer
words in three of the four images because it finds the largest fitting size
for each word, leaving less space for later words; the larger minimum on two
images contributes as well. Every skipped word has no legal placement at its
minimum size in either orientation with the current rectangular footprint.

Full-size [before](assets/placement-comparison/before) and
[after](assets/placement-comparison/after) PNGs were regenerated locally from
the baseline and replacement commits with the same inputs, font, and OpenCV
4.10. Both versions retain recognizable silhouettes. The replacement has
larger prominent words and more open areas, particularly in `gen-img-couple`.
Its orientation mix shifts toward vertical words. These are visible artistic
tradeoffs to review alongside the exact-fit and timing gains.
