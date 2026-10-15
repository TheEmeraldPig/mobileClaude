# mobileClaude

## sandbox/ (the game "Mote")

- A single-file browser game in `sandbox/index.html`, played mostly on a phone in landscape. Keep phone (touch) and computer (keyboard and mouse) layouts working; `IS_TOUCH` picks between them.
- **World generation is seeded.** Any terrain or world feature must follow the rules in `sandbox/README.md` under "Adding terrain features". The short version: no `Math.random()` in generation, add features as `GEN_PASSES`, and handle features that cross chunk borders.
- **UI style comes from the mote sprite:** dark outline `--o`, violet body `--b`, light top-left edge `--l`, shaded bottom-right edge `--s`, gold accent `--t`, 3px pixel bevels via `.btn`, `.px` and `.sheet`, and the Pixelify Sans and Silkscreen fonts. New menus and buttons should reuse these classes.
- Test with Playwright using the preinstalled Chromium (`executablePath: '/opt/pw-browsers/chromium'`), in both a desktop viewport and a phone context (`hasTouch: true, isMobile: true`).
