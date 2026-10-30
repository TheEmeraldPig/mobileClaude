package com.theemeraldpig.lens;

import android.Manifest;
import android.app.Activity;
import android.app.AlertDialog;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.graphics.Color;
import android.net.Uri;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.text.InputType;
import android.view.ViewGroup;
import android.webkit.JavascriptInterface;
import android.webkit.WebResourceError;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.TextView;
import android.widget.Toast;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.net.URLEncoder;

/**
 * Lens: shows the hub's control panel (served by lens.py in Termux) as an app of its own.
 * The hub does all the work; this app only displays it, and can start the hub in Termux.
 */
public class MainActivity extends Activity {
    static final String DEFAULT_HUB = "http://127.0.0.1:8765";
    static final String TERMUX = "com.termux";
    static final String RUN_SH = "/data/data/com.termux/files/home/mobileClaude/glasses/pi/android/run.sh";
    static final String PERM_RUN = "com.termux.permission.RUN_COMMAND";

    static volatile boolean visible;   // the updater only opens the install prompt while you're looking
    WebView web;
    SharedPreferences prefs;
    boolean offline;
    final Handler main = new Handler(Looper.getMainLooper());

    @Override
    protected void onCreate(Bundle saved) {
        super.onCreate(saved);
        prefs = getSharedPreferences("lens", MODE_PRIVATE);
        getWindow().setStatusBarColor(Color.rgb(5, 8, 13));
        getWindow().setNavigationBarColor(Color.rgb(5, 8, 13));

        // Settings can also come from the launching intent (used by the automated emulator test).
        Intent in = getIntent();
        if (in != null && in.hasExtra("hub")) {
            prefs.edit().putString("hub", in.getStringExtra("hub")).putString("token", in.getStringExtra("token")).commit();
        }

        web = new WebView(this);
        web.setBackgroundColor(Color.rgb(5, 8, 13));
        WebSettings ws = web.getSettings();
        ws.setJavaScriptEnabled(true);
        ws.setDomStorageEnabled(true);
        web.addJavascriptInterface(new Bridge(), "LensApp");
        web.setWebViewClient(new WebViewClient() {
            @Override
            public void onReceivedError(WebView view, WebResourceRequest req, WebResourceError err) {
                if (req.isForMainFrame()) showOffline(String.valueOf(err.getDescription()));
            }

            @Override
            public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest req) {
                Uri u = req.getUrl();
                if (u.toString().startsWith(hub())) return false;
                startActivity(new Intent(Intent.ACTION_VIEW, u));   // other sites open in the browser
                return true;
            }
        });
        setContentView(web, new ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));

        if (prefs.getString("token", "").isEmpty()) openSettings();
        else loadPanel();
        startListening();
    }

    /** Runs ListenService, which does the hub's speech recognition, once the microphone is allowed. */
    void startListening() {
        java.util.List<String> need = new java.util.ArrayList<>();
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO) != PackageManager.PERMISSION_GRANTED) need.add(Manifest.permission.RECORD_AUDIO);
        if (android.os.Build.VERSION.SDK_INT >= 33 && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED)
            need.add(Manifest.permission.POST_NOTIFICATIONS);
        if (!need.isEmpty()) {
            requestPermissions(need.toArray(new String[0]), 2);
            return;
        }
        try {
            startForegroundService(new Intent(this, ListenService.class));
        } catch (Exception e) {
            Toast.makeText(this, "Couldn't start listening: " + e.getMessage(), Toast.LENGTH_LONG).show();
        }
    }

    String hub() {
        String h = prefs.getString("hub", DEFAULT_HUB).trim();
        return h.endsWith("/") ? h.substring(0, h.length() - 1) : h;
    }

    void loadPanel() {
        offline = false;
        String token = prefs.getString("token", "");
        // The hub remembers the token in a cookie and drops it from the address.
        String t;
        try {
            t = URLEncoder.encode(token, "UTF-8");
        } catch (java.io.UnsupportedEncodingException e) {
            t = token;
        }
        web.loadUrl(hub() + "/panel?t=" + t);
    }

    void showOffline(String reason) {
        offline = true;
        String page;
        try (InputStream is = getAssets().open("offline.html")) {
            ByteArrayOutputStream out = new ByteArrayOutputStream();
            byte[] buf = new byte[8192];
            for (int n; (n = is.read(buf)) > 0; ) out.write(buf, 0, n);
            page = out.toString("UTF-8");
        } catch (Exception e) {
            page = "<p>The hub isn't answering.</p>";
        }
        page = page.replace("{{HUB}}", escape(hub())).replace("{{REASON}}", escape(reason));
        web.loadDataWithBaseURL("file:///android_asset/", page, "text/html", "utf-8", null);
    }

    static String escape(String s) {
        return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;").replace("\"", "&quot;");
    }

    void openSettings() {
        LinearLayout box = new LinearLayout(this);
        box.setOrientation(LinearLayout.VERTICAL);
        int pad = (int) (20 * getResources().getDisplayMetrics().density);
        box.setPadding(pad, pad / 2, pad, 0);

        TextView help = new TextView(this);
        help.setText("Hub address: leave it as is when the hub runs on this phone. "
                + "Token: the token from config.toml in Termux.");
        EditText address = new EditText(this);
        address.setHint(DEFAULT_HUB);
        address.setText(prefs.getString("hub", DEFAULT_HUB));
        address.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        EditText token = new EditText(this);
        token.setHint("token");
        token.setText(prefs.getString("token", ""));
        token.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_VISIBLE_PASSWORD);
        box.addView(help);
        box.addView(address);
        box.addView(token);

        new AlertDialog.Builder(this)
                .setTitle("Connect to your hub")
                .setView(box)
                .setPositiveButton("Save", (d, w) -> {
                    String h = address.getText().toString().trim();
                    prefs.edit().putString("hub", h.isEmpty() ? DEFAULT_HUB : h)
                            .putString("token", token.getText().toString().trim()).apply();
                    loadPanel();
                })
                .setNegativeButton("Cancel", (d, w) -> { if (web.getUrl() == null) loadPanel(); })
                .show();
    }

    /** Starts run.sh in Termux. Needs Termux's "allow-external-apps" setting and this app's Termux permission. */
    void startHub() {
        if (checkSelfPermission(PERM_RUN) != PackageManager.PERMISSION_GRANTED) {
            requestPermissions(new String[]{PERM_RUN}, 1);
            return;
        }
        Intent i = new Intent();
        i.setClassName(TERMUX, "com.termux.app.RunCommandService");
        i.setAction("com.termux.RUN_COMMAND");
        i.putExtra("com.termux.RUN_COMMAND_PATH", "/data/data/com.termux/files/usr/bin/bash");
        i.putExtra("com.termux.RUN_COMMAND_ARGUMENTS", new String[]{RUN_SH});
        i.putExtra("com.termux.RUN_COMMAND_WORKDIR", "/data/data/com.termux/files/home");
        i.putExtra("com.termux.RUN_COMMAND_BACKGROUND", true);
        try {
            startForegroundService(i);
            Toast.makeText(this, "Starting the hub in Termux…", Toast.LENGTH_SHORT).show();
            main.postDelayed(this::loadPanel, 8000);
        } catch (Exception e) {
            new AlertDialog.Builder(this)
                    .setTitle("Couldn't start the hub")
                    .setMessage("Termux needs to allow other apps to run commands. In Termux, run:\n\n"
                            + "echo \"allow-external-apps = true\" >> ~/.termux/termux.properties\n\n"
                            + "then close and reopen Termux, and try again.\n\n(" + e.getMessage() + ")")
                    .setPositiveButton("OK", null)
                    .show();
        }
    }

    @Override
    public void onRequestPermissionsResult(int code, String[] perms, int[] results) {
        if (code == 2) {
            if (checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED) startListening();
            else Toast.makeText(this, "Without the microphone, the hub uses Termux's speech recognition, which can cut off your last words.", Toast.LENGTH_LONG).show();
            return;
        }
        if (code == 1 && results.length > 0 && results[0] == PackageManager.PERMISSION_GRANTED) startHub();
        else Toast.makeText(this, "Allow Lens to run commands in Termux to start the hub from here.", Toast.LENGTH_LONG).show();
    }

    void openTermux() {
        Intent i = getPackageManager().getLaunchIntentForPackage(TERMUX);
        if (i != null) startActivity(i);
        else Toast.makeText(this, "Termux isn't installed.", Toast.LENGTH_SHORT).show();
    }

    @Override
    protected void onResume() {
        super.onResume();
        visible = true;
        if (offline) loadPanel();   // coming back from Termux: try again
        Updater.check(this, false, null);
    }

    @Override
    protected void onPause() {
        visible = false;
        super.onPause();
    }

    @Override
    public void onBackPressed() {
        if (web.canGoBack() && !offline) web.goBack();
        else super.onBackPressed();
    }

    /** What the panel and the offline page can call. */
    class Bridge {
        @JavascriptInterface public void openSettings() { main.post(MainActivity.this::openSettings); }
        @JavascriptInterface public void retry() { main.post(MainActivity.this::loadPanel); }
        @JavascriptInterface public void startHub() { main.post(MainActivity.this::startHub); }
        @JavascriptInterface public void openTermux() { main.post(MainActivity.this::openTermux); }
        @JavascriptInterface public void checkForUpdate() {
            Updater.check(MainActivity.this, true, msg -> main.post(() -> Toast.makeText(MainActivity.this, msg, Toast.LENGTH_LONG).show()));
        }
        @JavascriptInterface public String version() {
            try {
                return getPackageManager().getPackageInfo(getPackageName(), 0).versionName;
            } catch (Exception e) {
                return "";
            }
        }
    }
}
