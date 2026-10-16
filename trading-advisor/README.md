# Day-Trading Advisor

A personal trading journal and coach for US stocks.

- It **never** connects to a brokerage and **never** places orders. It only analyzes the trades you give it and gives advice.
- Your trades are private to you.

## Open it (no install)

**https://claude.ai/artifact/VUNk24GVvv5xewtmwJEMHq**

Open that link on your phone or computer while signed in to claude.ai. Bookmark it, or add it to your home screen.

- Your trades, rules and screenshots are saved with the page on your claude.ai account. Only you can open it, and only you can see the data: nothing is stored in this public repo.
- Claude can read your trades and improve the page in later chats. Just give it the link above.
- Use **Download backup (CSV)** on the Trades page now and then for your own copy.

### Your first 10 minutes

1. **My Rules:** set your account size, max risk per trade (1% is a common start), daily loss limit, max trades per day, minimum reward/risk, and the times of day you don't want to trade. Click **Save rules**.
2. **Import CSV:** click **32 fake trades** to load the sample, check the column matches, click **Preview import**, then **Import**.
3. Look around **Dashboard**, **Mistakes** and **Pre-Trade Check**.
4. When you're done exploring, go to **My Rules → Your data**, choose **Delete all paper trades**, and confirm. That clears the sample.

---

## Optional: the offline version on your computer

This folder also has an older version that runs on your own computer with Python and keeps data in a local `data` folder.
**The website above is the main version.** The offline version still works, but new features go to the website first.


### Step-by-step: first time setup

#### Step 1: Install Python (one time only)

**Windows**
1. Go to <https://www.python.org/downloads/> and click the big yellow **Download Python** button.
2. Open the file you downloaded.
3. **Important:** on the first screen, tick the box **"Add python.exe to PATH"**, then click **Install Now**.
4. When it says "Setup was successful", close it.

**Mac**
1. Go to <https://www.python.org/downloads/> and click **Download Python**.
2. Open the downloaded `.pkg` file and click through the installer (Continue → Agree → Install).

Any Python version from 3.9 up works.

#### Step 2: Get this folder onto your computer

You need the `trading-advisor` folder (the one this README is in) somewhere easy to find, like your Desktop or Documents.
Download it from GitHub as a ZIP: <https://github.com/TheEmeraldPig/mobileClaude/archive/refs/heads/tradingAdvisor.zip>. Unzip it and find the `trading-advisor` folder inside.

#### Step 3: Start the advisor

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

#### Step 4: Stop it

Click on the terminal window and press **Ctrl + C**, or just close the window. Your data is saved automatically after every change.

---

### Offline version: first 10 minutes

1. **My Rules:** set your account size, max risk per trade (1% is a common starting point), daily loss limit, max trades per day, minimum reward/risk, and the times of day you don't want to trade. Click **Save rules**.
2. **Try it with fake trades:** go to **Import CSV**, click the `sample_trades.csv` link to download it, then choose that file in the **CSV file** box. Check the column matches and click **Preview import**, then **Import**.
3. **Dashboard:** look at your stats, equity curve, the plain-English advice, and the breakdowns.
4. **Mistakes:** see which mistakes cost the most.
5. **Pre-Trade Check:** type in a trade you're thinking about and see the green/yellow/red warnings.
6. When you're done exploring, go to **My Rules → Your data** and choose **Delete all paper trades** to clear out the sample (it was imported as paper trades).

---

## Market context: what the stock was doing

For each trade, Claude can fetch what the market looked like and save it with the trade:

- the stock's **5-minute chart** for that day, with your entry, exit, stop and target drawn on it
- the **gap** at the open, **volume vs normal**, the day's range and ATR
- what the **overall market (SPY)** did that day
- **VWAP** when you entered, and whether you bought above or below it
- **news headlines** about the stock that day

The Dashboard then breaks your results down by these conditions (gap ups vs gap downs, heavy vs quiet volume, market up vs down days, above vs below VWAP, news days). On **Pre-Trade Check** you can enter today's conditions and see how your past trades did in the same situation.

To use it, just ask Claude: *"fetch market data for my trades"*. It needs a free Alpaca account's market-data keys, set up once:
1. Sign up at alpaca.markets and create API keys (paper account keys are fine; they're only used to read data).
2. In this Claude Code environment's settings (cloud environment menu in the session title bar → **Edit**), add them as secrets named `ALPACA_DATA_KEY_ID` and `ALPACA_DATA_SECRET_KEY`. Never paste keys into the chat.
3. In the same settings, under **Network access**, allow `data.alpaca.markets`.

## Ready for real money?

The top of the Dashboard shows a scorecard built only from your **paper** trades. All 10 checks must pass before moving to real money: enough trades and days, a profitable average day, mostly green days, green days clearly bigger than red days, a solid profit factor, small drawdowns, few rule breaks, never blowing through your daily loss limit, and still profitable over your last 20 trades. Change the targets under **My Rules**, but don't lower them to get there faster.

## Let Claude read your TradingView

Claude can read your TradingView watchlist, charts and Paper Trading history, and copy your paper trades into the advisor. It only reads: it never places, changes or closes orders.

1. Open the **Claude desktop app** on your computer (this needs your own computer, because TradingView is behind your login).
2. Start a Claude Code session on this repo there, and turn on the built-in browser or Claude in Chrome when asked.
3. Sign in to TradingView yourself in that browser. Never give Claude your password.
4. Ask things like *"check my TradingView watchlist"*, *"look at my AAPL chart"* or *"sync my TradingView paper trades"*. Claude shows you the trades before adding them.

No desktop app handy? In TradingView, open the Trading Panel → **Paper Trading → History**, export it as CSV, and import it on **Import CSV** with **One row per buy or sell**.

## Other traders' trades

You can import other people's trade histories to study and compare them, for example a journal a friend or mentor shares with you as a CSV.

1. On **Import CSV**, choose their file and type their name in **Whose trades are these?** (try **another trader's journal** to see a sample).
2. Their trades are kept separate. They never change your own stats, daily limits or pre-trade warnings.
3. Use **Whose trades** at the top of Dashboard, Trades and Mistakes to switch between *Mine*, one trader, *All other traders* or *Everyone*.
4. With other traders imported, the Dashboard adds a **You vs other traders** table, and **Pre-Trade Check** shows how they did with the same setup and time of day.
5. To remove someone, go to **My Rules → Other traders → Delete their trades**.

Only import data you're allowed to use. It stays private in your journal like everything else.

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

- **Website:** your trades are saved on your claude.ai account automatically. For your own copy, click **Download backup (CSV)** on the Trades page. To restore, import that CSV with **One row per trade** selected.
- **Offline version:** copy the whole `data` folder somewhere safe. To restore, put it back next to `app.py`.

## Troubleshooting

| Problem | Fix |
|---|---|
| The website says "Can't open the journal" | Sign in to claude.ai with the account that owns the page, then reload. |
| The website says "Preview: nothing is saved" | You opened a copy of the page outside claude.ai. Use the link at the top of this README. |
| `'python' is not recognized` (Windows) | Python wasn't added to PATH. Run the Python installer again, choose **Modify**/**Repair**, and make sure "Add Python to PATH" is ticked. Or restart your computer after installing. |
| `command not found: python3` (Mac) | Install Python from python.org (Step 1), then close and reopen the window. |
| "Port 8765 is busy" | The advisor is probably already running in another window. Just open http://127.0.0.1:8765. |
| The page says "Can't reach the advisor" | The terminal window was closed. Start it again (Step 3). |
| Times look shifted by an hour or more | Re-import and choose the right time zone, or edit the times. Use market time (Eastern). |

---

## For the curious: how it's built

- `web/`: the website. `index.html` (page), `app.js` (screens and saving), `logic.js` (all the trading math), `style.css`. Published to claude.ai; its data lives in the page's private database.
- `tests/web_logic_test.js`: checks for the website's math. Run with `node tests/web_logic_test.js`.
- `tools/market_context.py`: fetches market context (read-only; it never places orders). Tested by `tests/test_market_context.py`.

Offline version:
- `app.py`: the small local web server (Python standard library only).
- `db.py`: saves trades and settings in SQLite (`data/trades.db`).
- `analysis.py`: all the stats, insights, mistake flags and the pre-trade check.
- `importer.py`: CSV reading, date parsing and pairing fills into trades.
- `static/`: the web pages (HTML, CSS, JavaScript; charts are drawn locally, no internet needed).
- `samples/`: fake data to try things out.
- `tests/test_advisor.py`: checks for the offline version. Run with `python -m unittest discover tests`.

*This tool gives educational analysis of your own trading history. It is not financial advice, and it can't predict the market.*
