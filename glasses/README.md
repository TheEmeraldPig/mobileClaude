# Lens: DIY smart glasses

A home-made take on camera + AI glasses: you look at something, ask out loud, and hear an answer. Personal project, not for sale.

It's built in two halves:

1. **The phone app** (`index.html`, this folder): does the thinking. It shows the camera, listens to you, sends what you see to Claude and speaks the answer. It works right now on its own, using the phone's camera.
2. **The glasses** (to build): a tiny camera board on a frame with a button and a light. Later it sends its camera picture to the phone app over Bluetooth, so the phone can stay in your pocket.

## Using the phone app

Open the page on your phone (see "Hosting" below), tap **MENU**, paste a Claude API key from [console.anthropic.com](https://console.anthropic.com), and allow camera and microphone.

| Control | What it does |
|---|---|
| Mic (big button) | Tap and talk; it stops when you pause. Or hold it while you talk. |
| Eye | "What am I looking at?" (or the current mode's question) |
| Camera / video | Take a photo or start/stop a video. Captures are in MENU. |
| Keyboard | Type a question instead of talking |
| Mode chips | **Ask**, **Read** (reads text out), **Translate** (language in MENU), **Identify**, **How do I** (step-by-step help) |
| Tap the answer | Stop talking |

Say "remember that I parked on level 3" and it saves a note (in MENU). Later, "where did I park?" works. It can also search the web for things like opening hours.

On a computer: Space talk, L look, P photo, V video, T type, Esc stop.

Notes:
- The API key, notes and settings are stored only in that browser. Each question costs a little API money; the camera picture is the biggest part. Haiku 5.5 in MENU is the cheapest model.
- Voice input needs Chrome (Android) or Safari (iPhone). Firefox can only type.
- Captures last until you close the page; tap one to save or share it.

### Hosting

The camera and mic only work on an `https://` page. `.github/workflows/glasses.yml` publishes this folder to GitHub Pages on every push that touches `glasses/`. One-time setup: repo **Settings → Pages → Source: GitHub Actions**. Pages only accepts the repo's default branch at first; to publish from other branches too, set **Settings → Environments → github-pages → Deployment branches** to allow them. The page is then at `https://<user>.github.io/<repo>/`.

To try it on a computer without Pages: `npx serve glasses` (or `python3 -m http.server -d glasses`) and open `http://localhost:3000`; localhost counts as secure.

## Building the glasses

### Parts (about $35–60)

| Part | Why | Rough price |
|---|---|---|
| **Seeed Studio XIAO ESP32S3 Sense** | Thumb-sized board with a camera, microphone, Wi-Fi, Bluetooth and battery charging built in. This is the brain of the glasses. | $14–18 |
| 3.7 V LiPo battery, 250–500 mAh, with JST plug | Power. Smaller = lighter, larger = longer. | $5–8 |
| Small push button (6 mm tactile) | Press to talk / take a photo, like the button on Meta's glasses | $1 |
| Small LED (white or red) + 220 Ω resistor | Recording light so people know when the camera is on | $1 |
| Slide switch | On/off | $1 |
| Cheap sunglasses or safety glasses | The frame to mount everything on | $5–10 |
| Thin wire, heat-shrink, hot glue or zip ties | Holding it together | $5 |
| *Later:* MAX98357A amp + small 8 Ω speaker | Hear answers from the glasses instead of earbuds | $5–8 |
| *Optional:* 3D-printed case for the temple arm | Neater than hot glue; any print service can make it | $5–15 |

Tools: a soldering iron, a USB-C cable, and a computer with the Arduino IDE (free).

### Plan

1. **Phone app** (done): `index.html`. Try it, decide which features matter.
2. **Board on the desk.** Flash the XIAO with firmware that, over Bluetooth, takes a JPEG when the phone asks for one and sends it in chunks, and tells the phone when the button is pressed. The LED is wired so it's on whenever the camera runs.
3. **App talks to the glasses.** Add "Camera: glasses" to MENU using Web Bluetooth: button press on the glasses starts listening on the phone, and the question uses the glasses' picture instead of the phone's. Web Bluetooth works in Chrome on Android; on an iPhone it needs the Bluefy browser (or a small native app later).
4. **Put it on a frame.** Camera at the hinge pointing forward, board and battery along one temple arm, button on top of the arm, LED next to the camera.
5. **Audio on the glasses** (optional): speaker by the ear, then the glasses' own mic so the phone can stay in your pocket.

Bluetooth is used instead of Wi-Fi because a page served from `https://` isn't allowed to talk to a plain `http://` device on your Wi-Fi.

### Be a good camera wearer

Keep the recording light working and visible, don't film people who haven't agreed to it, and check local rules on recording audio: some places need everyone's consent.
