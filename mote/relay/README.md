# Mote relay (multiplayer server)

A tiny server that lets players join each other's worlds with a 5-letter code. It
only passes messages between the host's game and its guests (up to 3), so it needs
almost no CPU or memory. The protocol is described at the top of `main.go`.

## Put it online (free, on Render)

1. Sign in at [render.com](https://render.com) with your GitHub account.
2. Choose **New > Blueprint**, pick the `mobileClaude` repository and apply it. It reads
   `render.yaml` at the top of the repository and creates a free web service called
   `mote-relay`.
3. When it's live, copy its address, e.g. `https://mote-relay.onrender.com`.
4. In the GitHub repository go to **Settings > Secrets and variables > Actions >
   Variables** and add a variable named `MOTE_RELAY_URL` with that address but starting
   with `wss://` instead of `https://` (e.g. `wss://mote-relay.onrender.com`).
5. Push any change to `mote/` (or re-run the Build Mote workflow): every build after
   that has the server built in.

Free Render services go to sleep after about 15 minutes with nobody connected. The
first player to open a world after that waits up to a minute while it wakes up; after
that it's instant.

Players can also type a server address into the Multiplayer screen on a PC (click the
"Server:" line at the bottom), which is handy for testing.

## Run it yourself

```
cd mote
go run ./relay            # listens on :8080 (or PORT)
```

Then use `ws://localhost:8080` as the server. With Docker:
`docker build -f relay/Dockerfile -t mote-relay . && docker run -p 8080:8080 mote-relay`.
