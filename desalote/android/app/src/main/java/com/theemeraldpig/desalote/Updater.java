package com.theemeraldpig.desalote;

import android.app.Activity;
import android.app.AlertDialog;
import android.app.PendingIntent;
import android.content.Intent;
import android.content.pm.PackageInfo;
import android.content.pm.PackageInstaller;
import android.net.Uri;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.provider.Settings;
import android.util.Log;

import com.theemeraldpig.desalote.mobile.Mobile;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;

// Keeps the sideloaded app up to date. Android doesn't let an app replace itself
// silently (only app stores can), so this does the rest: it checks the GitHub releases
// when the game starts and every few minutes, downloads a newer Desalote.apk in the
// background, and hands it to Android's installer, which asks you to tap Update once.
public class Updater {
    private static final String RELEASES = "https://api.github.com/repos/TheEmeraldPig/mobileClaude/releases?per_page=30";
    private static final String PREFIX = "desalote-build-";
    private static final long EVERY_MS = 5 * 60 * 1000;

    private final Activity act;
    private final Handler main = new Handler(Looper.getMainLooper());
    private boolean busy, asked;
    private int skipped; // a build the player said "Later" to

    Updater(Activity act) { this.act = act; }

    // Check now, then keep checking while the app is open
    void start() {
        check();
        main.postDelayed(new Runnable() {
            @Override public void run() {
                check();
                main.postDelayed(this, EVERY_MS);
            }
        }, EVERY_MS);
    }

    private int installedBuild() {
        try {
            PackageInfo pi = act.getPackageManager().getPackageInfo(act.getPackageName(), 0);
            return pi.versionCode;
        } catch (Exception e) {
            return 0;
        }
    }

    void check() {
        if (busy) return;
        busy = true;
        new Thread(() -> {
            try {
                HttpURLConnection c = (HttpURLConnection) new URL(RELEASES).openConnection();
                c.setRequestProperty("Accept", "application/vnd.github+json");
                c.setConnectTimeout(10000);
                c.setReadTimeout(15000);
                String body = readAll(c.getInputStream());
                // This repository also releases Mote, so look through the list for the
                // newest release tagged desalote-build-N (never "latest", which is Mote's)
                JSONArray rels = new JSONArray(body);
                int latest = -1;
                String apk = null;
                for (int r = 0; r < rels.length(); r++) {
                    JSONObject rel = rels.getJSONObject(r);
                    String tag = rel.optString("tag_name", "");
                    if (!tag.startsWith(PREFIX)) continue;
                    int n;
                    try { n = Integer.parseInt(tag.substring(PREFIX.length())); } catch (NumberFormatException e) { continue; }
                    if (n <= latest) continue;
                    JSONArray assets = rel.optJSONArray("assets");
                    for (int i = 0; assets != null && i < assets.length(); i++) {
                        JSONObject a = assets.getJSONObject(i);
                        if ("Desalote.apk".equals(a.optString("name"))) { latest = n; apk = a.optString("browser_download_url"); }
                    }
                }
                if (latest > installedBuild() && latest != skipped && apk != null) {
                    final String url = apk;
                    final int build = latest;
                    main.post(() -> offer(build, url));
                } else {
                    busy = false;
                }
            } catch (Exception e) {
                Log.w("Desalote", "update check failed", e);
                busy = false;
            }
        }).start();
    }

    private void offer(int build, String url) {
        if (asked) { busy = false; return; }
        asked = true;
        new AlertDialog.Builder(act)
            .setTitle("Desalote build " + build + " is ready")
            .setMessage("A new version of Desalote is out. Update now? Your settings are kept.")
            .setPositiveButton("Update", (d, w) -> {
                asked = false;
                if (!canInstall()) return;
                download(url);
            })
            .setNegativeButton("Later", (d, w) -> {
                asked = false;
                skipped = build;
                busy = false;
            })
            .setCancelable(false)
            .show();
    }

    // Android 8+ asks once whether Desalote may install updates
    private boolean canInstall() {
        if (Build.VERSION.SDK_INT >= 26 && !act.getPackageManager().canRequestPackageInstalls()) {
            new AlertDialog.Builder(act)
                .setTitle("Allow Desalote to install updates")
                .setMessage("On the next screen, turn on \"Allow from this source\", then come back. Desalote will finish the update.")
                .setPositiveButton("Open settings", (d, w) -> {
                    pendingAfterSettings = true;
                    act.startActivity(new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                        Uri.parse("package:" + act.getPackageName())));
                })
                .setNegativeButton("Not now", null)
                .show();
            busy = false;
            return false;
        }
        return true;
    }

    private boolean pendingAfterSettings;

    // Back from the settings screen: try again right away
    void onResume() {
        if (pendingAfterSettings) {
            pendingAfterSettings = false;
            busy = false;
            check();
        }
    }

    private void download(String url) {
        final AlertDialog progress = new AlertDialog.Builder(act)
            .setTitle("Updating Desalote")
            .setMessage("Downloading...")
            .setCancelable(false)
            .show();
        new Thread(() -> {
            try {
                Mobile.save();
                HttpURLConnection c = (HttpURLConnection) new URL(url).openConnection();
                c.setInstanceFollowRedirects(true);
                c.setConnectTimeout(15000);
                c.setReadTimeout(30000);
                long total = c.getContentLength(); // -1 when unknown
                PackageInstaller pi = act.getPackageManager().getPackageInstaller();
                PackageInstaller.SessionParams params = new PackageInstaller.SessionParams(
                    PackageInstaller.SessionParams.MODE_FULL_INSTALL);
                params.setAppPackageName(act.getPackageName());
                int id = pi.createSession(params);
                PackageInstaller.Session s = pi.openSession(id);
                try (InputStream in = c.getInputStream(); OutputStream out = s.openWrite("Desalote.apk", 0, total)) {
                    byte[] buf = new byte[1 << 16];
                    long done = 0;
                    int n, lastPct = -1;
                    while ((n = in.read(buf)) > 0) {
                        out.write(buf, 0, n);
                        done += n;
                        int pct = total > 0 ? (int) (done * 100 / total) : -1;
                        if (pct != lastPct && pct % 5 == 0) {
                            lastPct = pct;
                            final int p = pct;
                            main.post(() -> progress.setMessage("Downloading... " + p + "%"));
                        }
                    }
                    s.fsync(out);
                }
                Intent done = new Intent(act, InstallReceiver.class);
                int flags = PendingIntent.FLAG_UPDATE_CURRENT;
                if (Build.VERSION.SDK_INT >= 31) flags |= PendingIntent.FLAG_MUTABLE;
                PendingIntent pend = PendingIntent.getBroadcast(act, id, done, flags);
                s.commit(pend.getIntentSender());
                s.close();
                main.post(() -> {
                    progress.dismiss();
                    busy = false;
                });
            } catch (Exception e) {
                Log.w("Desalote", "update failed", e);
                main.post(() -> {
                    progress.dismiss();
                    busy = false;
                    new AlertDialog.Builder(act)
                        .setTitle("Update failed")
                        .setMessage("Desalote couldn't download the update (" + e.getMessage() + "). It will try again later.")
                        .setPositiveButton("OK", null)
                        .show();
                });
            }
        }).start();
    }

    private static String readAll(InputStream in) throws java.io.IOException {
        java.io.ByteArrayOutputStream b = new java.io.ByteArrayOutputStream();
        byte[] buf = new byte[8192];
        int n;
        while ((n = in.read(buf)) > 0) b.write(buf, 0, n);
        in.close();
        return b.toString("UTF-8");
    }
}
