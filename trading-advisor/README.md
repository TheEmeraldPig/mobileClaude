# Day-Trading Advisor

A personal trading journal and coach that runs **only on your own computer**.

- It **never** connects to a brokerage and **never** places orders. It only analyzes the trades you give it and gives advice.
- Everything is saved in the `data` folder inside this folder: a database file (`trades.db`) and your screenshots. Nothing is uploaded anywhere.
- The app only accepts connections from your own computer (`127.0.0.1`). Other devices on your Wi-Fi can't open it.
- It needs nothing but Python. There are no extra packages to install, and it works without internet.

---

## Step-by-step: first time setup

### Step 1: Install Python (one time only)

**Windows**
1. Go to <https://www.python.org/downloads/> and click the big yellow **Download Python** button.
2. Open the file you downloaded.
3. **Important:** on the first screen, tick the box **"Add python.exe to PATH"**, then click **Install Now**.
4. When it says "Setup was successful", close it.

**Mac**
1. Go to <https://www.python.org/downloads/> and click **Download Python**.
2. Open the downloaded `.pkg` file and click through the installer (Continue → Agree → Install).

Any Python version from 3.9 up works.

### Step 2: Get this folder onto your computer

You need the `trading-advisor` folder (the one this README is in) somewhere easy to find, like your Desktop or Documents.
If you got it from GitHub: click the green **Code** button → **Download ZIP**, unzip it, and find the `trading-advisor` folder inside.

### Step 3: Start the advisor

**Windows:** open the `trading-advisor` folder and double-click **`start-windows.bat`**.
(If Windows shows "Windows protected your PC", click **More info** → **Run anyway**. It's just the 6-line file in this folder.)

**Mac:** open the `trading-advisor` folder and double-click **`start-mac.command`**.
(The first time, macOS may block it. If so, right-click the file → **Open** → **Open**.)

A black (or white) terminal window opens and shows:

```
  Day-Trading Advisor is running (analysis only, no broker).
  Open this in your browser:  http://127.0.0.1:8765
```

Your web browser should open by itself. If it doesn't, open your browser and type **http://127.0.0.1:8765** in the address bar.

**Keep the terminal window open while you use the app.** Closing it stops the advisor.

<details>
<summary>Prefer typing commands? (optional)</summary>

Open Terminal (Mac) or Command Prompt (Windows), go into the folder and run the app:

```
cd path/to/trading-advisor
python app.py        # on Mac use: python3 app.py
```
</details>

### Step 4: Stop it

Click on the terminal window and press **Ctrl + C**, or just close the window. Your data is saved automatically after every change.

---

## Your first 10 minutes

1. **My Rules:** set your account size, max risk per trade (1% is a common starting point), daily loss limit, max trades per day, minimum reward/risk, and the times of day you don't want to trade. Click **Save rules**.
2. **Try it with fake trades:** go to **Import CSV**, click the `sample_trades.csv` link to download it, then choose that file in the **CSV file** box. Check the column matches and click **Preview import**, then **Import**.
3. **Dashboard:** look at your stats, equity curve, the plain-English advice, and the breakdowns.
4. **Mistakes:** see which mistakes cost the most.
5. **Pre-Trade Check:** type in a trade you're thinking about and see the green/yellow/red warnings.
6. When you're done exploring, go to **My Rules → Your data** and choose **Delete all paper trades** to clear out the sample (it was imported as paper trades).

---

## The pages

| Page | What it does |
|---|---|
| **Dashboard** | Total P&L, win rate, average win vs loss, expectancy, profit factor, equity curve, max drawdown, streaks, breakdowns (time of day, day of week, ticker, setup, hold time, plan followed, emotion) and plain-English insights. |
| **Trades** | Every trade with its P&L, R-multiple, hold time and any mistake flags. Click a trade to edit it or fill in its journal. |
| **Add Trade** | Manual entry: ticker, long/short, paper or real, entry/exit date and time, prices, shares, stop-loss, target, fees. It shows your risk and reward/risk while you type. Plus the **journal**: setup, reason, emotion before/after, "followed my plan?", mistakes, notes and a chart screenshot. |
| **Import CSV** | Import from any broker. You match their column names to ours. |
| **Mistakes** | Which mistakes cost you the most money, and which trades had them. |
| **Pre-Trade Check** | Enter a planned trade and see dollar risk, % of account, reward/risk, the max shares your rules allow, rule warnings, and how your past similar trades went. **Advice only. You decide.** |
| **My Rules** | Your rules and your lists (setups, emotions, common mistakes). Also: backup download and delete options. |

The **All / Real money / Paper** switch at the top of Dashboard, Trades and Mistakes filters everything. Paper and real trades are always judged separately. A paper loss never makes a real trade count as a "revenge trade".

### Mistakes the advisor flags automatically

- **Risked more than your max:** (entry − stop) × shares was more than your max risk.
- **No stop-loss**, or **stop on the wrong side** of your entry.
- **Reward/risk below your minimum:** based on your target and stop.
- **Revenge trade:** opened within 15 minutes (you can change this) of a losing trade closing.
- **Overtrading:** every trade past your max trades per day.
- **Traded after hitting daily loss limit**
- **Traded in a no-trade window**
- **Held a loser too long:** a losing trade held more than 1.5× as long as your typical winner.

You can also tag your own mistakes in each trade's journal (e.g. "Chased entry"). Those show up on the Mistakes page too.

### Words you'll see

- **Win rate:** % of trades that made money.
- **Expectancy:** your average profit or loss per trade. Positive means your trading makes money on average.
- **Profit factor:** total won ÷ total lost. Above 1 means profitable; 1.5+ is solid.
- **Max drawdown:** the biggest drop from a high point in your account to a later low.
- **R / R-multiple:** your result measured in units of what you risked. +2R means you made twice what you risked.
- **Reward/risk:** how far your target is compared with how far your stop is. 2 : 1 means the target is twice as far as the stop.

---

## Importing from your broker

1. In your broker's website, find something like **Account → History**, **Statements**, or **Trade Activity** and export/download it as **CSV**.
2. In the advisor, open **Import CSV** and pick the file.
3. Tell it how the file is laid out:
   - **One row per trade:** each row has both the entry and the exit.
   - **One row per buy or sell:** each row is a single order fill. This is what most brokers export (Robinhood, Webull, Schwab, Fidelity, E*TRADE, IBKR…). The advisor pairs your buys and sells into trades for you. A trade starts when you go from 0 shares to a position and ends when you're back to 0. Scaling in and out is handled.
4. Choose whether these are **paper** or **real money** trades, and which **time zone** the file uses (the advisor uses US Eastern / market time).
5. Check the column matches it guessed, click **Preview import**, read any notes, then click **Import**.

Importing the same file twice won't duplicate trades.

Broker files almost never include your stop-loss or target, so imported trades will be flagged "No stop" until you edit them and add the stop you actually used.

---

## Backups

- Easiest: copy the whole `data` folder somewhere safe (a USB stick or cloud drive).
- Or click **Download backup (CSV)** on the Trades page.
- To restore from the `data` folder: put it back next to `app.py`.

## Troubleshooting

| Problem | Fix |
|---|---|
| `'python' is not recognized` (Windows) | Python wasn't added to PATH. Run the Python installer again, choose **Modify**/**Repair**, and make sure "Add Python to PATH" is ticked. Or restart your computer after installing. |
| `command not found: python3` (Mac) | Install Python from python.org (Step 1), then close and reopen the window. |
| "Port 8765 is busy" | The advisor is probably already running in another window. Just open http://127.0.0.1:8765. |
| The page says "Can't reach the advisor" | The terminal window was closed. Start it again (Step 3). |
| Times look shifted by an hour or more | Re-import and choose the right time zone, or edit the times. Use market time (Eastern). |

---

## For the curious: how it's built

- `app.py`: the small local web server (Python standard library only).
- `db.py`: saves trades and settings in SQLite (`data/trades.db`).
- `analysis.py`: all the stats, insights, mistake flags and the pre-trade check.
- `importer.py`: CSV reading, date parsing and pairing fills into trades.
- `static/`: the web pages (HTML, CSS, JavaScript; charts are drawn locally, no internet needed).
- `samples/`: fake data to try things out.
- `tests/`: automated checks. Run them with `python -m unittest discover tests`.

*This tool gives educational analysis of your own trading history. It is not financial advice, and it can't predict the market.*
