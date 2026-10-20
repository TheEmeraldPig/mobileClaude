#!/data/data/com.termux/files/usr/bin/bash
# Sets up an Android phone (in Termux) as the Lens hub.
#   pkg install git && git clone -b ccr-ce1e995c-dmjvgi https://github.com/TheEmeraldPig/mobileClaude ~/mobileClaude
#   bash ~/mobileClaude/glasses/pi/android/setup.sh
# Safe to run again; it skips what's already done.
set -euo pipefail
ANDROID_DIR="$(cd "$(dirname "$0")" && pwd)"
PI_DIR="$(dirname "$ANDROID_DIR")"

say() { printf '\n\033[1;36m== %s\033[0m\n' "$*"; }

say "Termux packages"
pkg update -y
pkg install -y python git curl proot-distro termux-api

say "Ubuntu for Claude Code (Claude Code has no Android build)"
[ -d "$PREFIX/var/lib/proot-distro/installed-rootfs/ubuntu" ] || proot-distro install ubuntu
proot-distro login ubuntu -- bash -c '
  set -e
  if [ ! -x /root/.local/bin/claude ] || ! command -v python3 >/dev/null; then
    apt-get update -qq && apt-get install -y -qq curl ca-certificates python3 git
    curl -fsSL https://claude.ai/install.sh | bash
  fi'
install -m 755 "$ANDROID_DIR/claude-proot" "$PREFIX/bin/claude-proot"

say "Settings"
[ -f "$PI_DIR/config.toml" ] || cp "$ANDROID_DIR/config.android.toml" "$PI_DIR/config.toml"

say "Start on boot (needs the Termux:Boot app)"
mkdir -p ~/.termux/boot
cat > ~/.termux/boot/lens <<EOF
#!/data/data/com.termux/files/usr/bin/sh
termux-wake-lock
nohup bash "$ANDROID_DIR/run.sh" >> ~/lens-data/run.log 2>&1 &
EOF
chmod +x ~/.termux/boot/lens "$ANDROID_DIR/run.sh"
mkdir -p ~/lens-data

cat <<EOF

Almost done. Things only you can do (all explained in glasses/README.md):

  1. Install the Termux:API and Termux:Boot apps from F-Droid (same place as Termux), open each once.
  2. Log Claude Code into your account:   claude-proot     (open the link, sign in, paste the code, then /exit)
  3. Edit $PI_DIR/config.toml: set a token (and the same one in the glasses firmware).
  4. Android settings: Battery -> Termux and Termux:API -> Unrestricted; let Termux:API use the microphone.

Then check everything and start it:

  cd $PI_DIR && python3 lens.py --selftest --deep
  bash $ANDROID_DIR/run.sh          (or restart the phone; Termux:Boot starts it)

EOF
