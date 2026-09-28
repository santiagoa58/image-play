# Fonts for word clouds

`NotoSansMono-VariableFont_wdth,wght.ttf` gives every character roughly the
same width and is used for the text-mosaic example.

`NotoSans-Bold.ttf` is a proportional bold alternative: letters such as `I`
occupy less width than `W`, and thicker strokes remain visible at smaller
preview sizes. Select it with `-font fonts/NotoSans-Bold.ttf`; add `-uppercase`
for uppercase word-cloud text. Text is converted before measurement, so the
layout accounts for the new letter widths.

The bold font is an unmodified upstream file from the archived
[Noto font repository](https://github.com/notofonts/noto-fonts/blob/main/hinted/ttf/NotoSans/NotoSans-Bold.ttf),
distributed under the [SIL Open Font License 1.1](OFL.txt). Its source download is
[the upstream TTF](https://raw.githubusercontent.com/notofonts/noto-fonts/main/hinted/ttf/NotoSans/NotoSans-Bold.ttf).
