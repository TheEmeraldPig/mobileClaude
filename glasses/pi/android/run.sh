#!/data/data/com.termux/files/usr/bin/bash
# Keeps the hub running on Android (which has no systemd): restarts it when it stops,
# and runs the doctor when it crashes 5 times in 5 minutes. Termux:Boot starts this on boot.
cd "$(dirname "$0")/.." || exit 1
termux-wake-lock 2>/dev/null   # stop Android from pausing the hub while the screen is off

crashes=()
while true; do
  python3 lens.py
  code=$?
  now=$(date +%s)
  recent=()
  for t in "${crashes[@]}" "$now"; do (( now - t < 300 )) && recent+=("$t"); done
  crashes=("${recent[@]}")
  if (( ${#crashes[@]} >= 5 )); then
    echo "lens crashed ${#crashes[@]} times in 5 minutes (last exit $code): running the doctor"
    python3 doctor.py
    crashes=()
  fi
  sleep 3
done
