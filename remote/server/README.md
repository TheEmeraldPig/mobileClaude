# Remote server

The online server for Remote. Every player connects to it over a WebSocket at `/ws`; `/` answers with how many are online (used as the health check).

## Running it

```sh
cd remote
go run ./server                    # listens on $PORT (default 8080)
```

| Setting | What it does |
|---|---|
| `PORT` | Port to listen on (default 8080) |
| `REMOTE_DATA` | Where characters are saved (default `./data`) |
| `REMOTE_SECRET` | Signs the character backups the games keep. Use a long random value that never changes: if it changes, old backups stop working |

## Putting it online on Render (free)

1. On [render.com](https://render.com): **New > Blueprint**, pick this repository, and set the **Blueprint path** to `remote/render.yaml` (Mote's own blueprint is the `render.yaml` at the root). Apply it. Render builds `server/Dockerfile` and makes up a `REMOTE_SECRET` for you.
   - If your Render account doesn't offer a Blueprint path: **New > Web Service**, pick this repository, Runtime **Docker**, Dockerfile path `./remote/server/Dockerfile`, Docker context `./remote`, plan Free. Under Environment, add `REMOTE_SECRET` with a long random value.
2. Copy the service's address (like `https://remote-server-xxxx.onrender.com`).
3. In GitHub: **Settings > Secrets and variables > Actions > Variables**, add a repository variable `REMOTE_SERVER_URL` with that address. The next build of the game has it built in, and **Play online** connects to it. Players can also type an address under Settings > Server on a PC.

Each push that changes `remote/mmo`, `remote/srv` or `remote/server` redeploys the server.

### About the free plan

- It sleeps after 15 minutes with nobody connected. The first player to connect wakes it, which takes up to a minute, and the game says "Waking the server up..." while it waits.
- Its disk is wiped whenever it restarts or redeploys. Characters survive that: every game keeps a signed copy of its character and hands it back when it connects. Keep `REMOTE_SECRET` the same, or those copies stop being accepted.
- A paid plan (about $7 a month) stays awake. A persistent disk (on paid plans) can be mounted with `REMOTE_DATA` pointing at it.
