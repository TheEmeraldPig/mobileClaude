#!/usr/bin/env bash
# Sets up a Raspberry Pi (Raspberry Pi OS 64-bit, Bookworm or newer) as the Lens hub.
# Run it from this folder as your normal user, not root:  bash setup.sh
# Safe to run again; it skips what's already done.
set -euo pipefail
cd "$(dirname "$0")"
PI_DIR="$(pwd)"

say() { printf '\n\033[1;36m== %s\033[0m\n' "$*"; }

say "System packages"
sudo apt-get update -qq
sudo apt-get install -y -qq git python3 alsa-utils ffmpeg espeak-ng avahi-daemon cmake build-essential curl

say "Hostname lens-hub (the glasses find the hub as lens-hub.local)"
if [ "$(hostname)" != "lens-hub" ]; then
  sudo hostnamectl set-hostname lens-hub
  sudo sed -i "s/127.0.1.1.*/127.0.1.1\tlens-hub/" /etc/hosts
fi

say "Speech to text: whisper.cpp"
if [ ! -x ~/whisper.cpp/build/bin/whisper-cli ]; then
  [ -d ~/whisper.cpp ] || git clone --depth 1 https://github.com/ggml-org/whisper.cpp ~/whisper.cpp
  cmake -S ~/whisper.cpp -B ~/whisper.cpp/build -DCMAKE_BUILD_TYPE=Release
  cmake --build ~/whisper.cpp/build -j"$(nproc)" --config Release --target whisper-cli
fi
# base.en is a good balance on a Pi 4/5; on slower boards use tiny.en and change config.toml
[ -f ~/whisper.cpp/models/ggml-base.en.bin ] || bash ~/whisper.cpp/models/download-ggml-model.sh base.en

say "Text to speech: Piper"
if [ ! -x ~/piper/piper ]; then
  curl -fsSL https://github.com/rhasspy/piper/releases/download/2023.11.14-2/piper_linux_aarch64.tar.gz | tar xz -C ~
fi
mkdir -p ~/piper/voices
VOICE=https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/medium/en_US-lessac-medium.onnx
[ -f ~/piper/voices/en_US-lessac-medium.onnx ] || curl -fsSL -o ~/piper/voices/en_US-lessac-medium.onnx "$VOICE"
[ -f ~/piper/voices/en_US-lessac-medium.onnx.json ] || curl -fsSL -o ~/piper/voices/en_US-lessac-medium.onnx.json "$VOICE.json"

say "Claude Code"
if ! command -v claude >/dev/null; then
  curl -fsSL https://claude.ai/install.sh | bash
  export PATH="$HOME/.local/bin:$PATH"
fi

say "Settings"
[ -f config.toml ] || cp config.example.toml config.toml

say "Services (start on boot, restart on crash, doctor on repeated crashes)"
mkdir -p ~/.config/systemd/user
sed "s#%h/mobileClaude/glasses/pi#$PI_DIR#g" systemd/lens.service > ~/.config/systemd/user/lens.service
sed "s#%h/mobileClaude/glasses/pi#$PI_DIR#g" systemd/lens-doctor.service > ~/.config/systemd/user/lens-doctor.service
sudo loginctl enable-linger "$USER"   # run without anyone logged in
systemctl --user daemon-reload
systemctl --user enable lens.service

cat <<EOF

Almost done. Two things only you can do:

  1. Log Claude Code into your account:   claude      (follow the login link, then type /exit)
  2. Edit config.toml: set a token (and the same one in the glasses firmware).

Then check everything and start it:

  python3 lens.py --selftest --deep
  systemctl --user start lens.service
  journalctl --user -u lens -f        # watch it work

EOF
