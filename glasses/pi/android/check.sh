#!/data/data/com.termux/files/usr/bin/bash
# Prints a short report on the Android hub setup. Copy everything it prints when asking for help.
#   bash ~/mobileClaude/glasses/pi/android/check.sh
ANDROID_DIR="$(cd "$(dirname "$0")" && pwd)"
PI_DIR="$(dirname "$ANDROID_DIR")"
ROOTFS="$PREFIX/var/lib/proot-distro/installed-rootfs"

line() { printf '%-26s %s\n' "$1" "$2"; }
have() { command -v "$1" >/dev/null 2>&1 && echo yes || echo NO; }

echo "== Lens hub check $(date '+%Y-%m-%d %H:%M')"
line "Android" "$(getprop ro.build.version.release 2>/dev/null) on $(getprop ro.product.model 2>/dev/null)"
line "code" "$(git -C "$PI_DIR" log --oneline -1 2>/dev/null || echo 'NOT FOUND')"
line "python3" "$(have python3)"
line "termux-tts-speak" "$(have termux-tts-speak)"
line "proot-distro" "$(have proot-distro) $(proot-distro --version 2>/dev/null | head -1)"
line "claude-proot wrapper" "$( [ -x "$PREFIX/bin/claude-proot" ] && echo yes || echo NO)"
line "distros folder" "$(ls "$ROOTFS" 2>/dev/null | tr '\n' ' ' || echo 'none')"
line "claude inside ubuntu" "$( [ -e "$ROOTFS/ubuntu/root/.local/bin/claude" ] && echo yes || echo NO)"
line "config.toml" "$( [ -f "$PI_DIR/config.toml" ] && echo yes || echo NO)"

echo
echo "== proot-distro list"
proot-distro list 2>&1 | tail -15

echo
echo "== inside ubuntu"
proot-distro login ubuntu -- bash -c 'echo "logged in as $(whoami), home $HOME"; ls -la /root/.local/bin/ 2>&1; command -v python3 || echo "no python3"' 2>&1 | tail -10

echo
echo "== claude-proot --version"
cd "$HOME" && claude-proot --version 2>&1 | tail -5
echo
echo "== end of report"
