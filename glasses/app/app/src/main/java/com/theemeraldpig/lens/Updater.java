package com.theemeraldpig.lens;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.PackageInstaller;
import android.os.Build;
import android.util.Log;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;

/**
 * Keeps the app up to date, like Mote's Windows updater: asks GitHub for the newest lens-app-<n>
 * release and, when n is newer than this build, downloads Lens.apk and installs it.
 * The first update asks you to confirm; after that (Android 12+) updates install by themselves,
 * because Lens is then the app that installed itself.
 */
final class Updater {
    static final String RELEASES = "https://api.github.com/repos/TheEmeraldPig/mobileClaude/releases?per_page=40";
    static final String TAG = "LensUpdate";
    static final String CHANNEL = "updates";
    static final long MIN_GAP_MS = 10 * 60 * 1000;   // GitHub allows 60 unauthenticated checks an hour
    static volatile boolean running;

    interface Report { void say(String message); }

    static long currentBuild(Context ctx) {
        try {
            return ctx.getPackageManager().getPackageInfo(ctx.getPackageName(), 0).getLongVersionCode();
        } catch (Exception e) {
            return 0;
        }
    }

    /** Checks in the background. force skips the 10-minute gap; report (may be null) hears how it went. */
    static void check(Context ctx, boolean force, Report report) {
        Context app = ctx.getApplicationContext();
        SharedPreferences prefs = app.getSharedPreferences("lens", Context.MODE_PRIVATE);
        long now = System.currentTimeMillis();
        if (running || (!force && now - prefs.getLong("lastUpdateCheck", 0) < MIN_GAP_MS)) return;
        running = true;
        prefs.edit().putLong("lastUpdateCheck", now).apply();
        new Thread(() -> {
            try {
                String msg = checkNow(app);
                if (report != null) report.say(msg);
            } catch (Exception e) {
                Log.w(TAG, "update check failed", e);
                if (report != null) report.say("Couldn't check for updates: " + e.getMessage());
            } finally {
                running = false;
            }
        }, "lens-update").start();
    }

    static String checkNow(Context ctx) throws Exception {
        long have = currentBuild(ctx);
        JSONArray releases = new JSONArray(read(RELEASES));
        long best = -1;
        String url = null;
        for (int i = 0; i < releases.length(); i++) {
            JSONObject r = releases.getJSONObject(i);
            String tag = r.optString("tag_name");
            if (!tag.startsWith("lens-app-") || r.optBoolean("draft")) continue;
            long n;
            try {
                n = Long.parseLong(tag.substring("lens-app-".length()));
            } catch (NumberFormatException e) {
                continue;
            }
            JSONArray assets = r.optJSONArray("assets");
            for (int j = 0; assets != null && j < assets.length(); j++) {
                JSONObject a = assets.getJSONObject(j);
                if ("Lens.apk".equals(a.optString("name")) && n > best) {
                    best = n;
                    url = a.optString("browser_download_url");
                }
            }
        }
        if (url == null || best <= have) return "Lens is up to date (build " + have + ").";
        File apk = new File(ctx.getCacheDir(), "Lens-" + best + ".apk");
        download(url, apk);
        install(ctx, apk);
        return "Installing build " + best + "…";
    }

    static String read(String url) throws Exception {
        HttpURLConnection c = open(url);
        try (InputStream in = c.getInputStream()) {
            java.io.ByteArrayOutputStream out = new java.io.ByteArrayOutputStream();
            byte[] buf = new byte[16 * 1024];
            for (int n; (n = in.read(buf)) > 0; ) out.write(buf, 0, n);
            return out.toString("UTF-8");
        } finally {
            c.disconnect();
        }
    }

    static void download(String url, File to) throws Exception {
        HttpURLConnection c = open(url);
        try (InputStream in = c.getInputStream(); OutputStream out = new FileOutputStream(to)) {
            byte[] buf = new byte[64 * 1024];
            for (int n; (n = in.read(buf)) > 0; ) out.write(buf, 0, n);
        } finally {
            c.disconnect();
        }
    }

    static HttpURLConnection open(String url) throws Exception {
        HttpURLConnection c = (HttpURLConnection) new URL(url).openConnection();
        c.setConnectTimeout(15000);
        c.setReadTimeout(60000);
        c.setInstanceFollowRedirects(true);
        c.setRequestProperty("User-Agent", "Lens-app-updater");   // GitHub's API needs one
        c.setRequestProperty("Accept", "application/vnd.github+json");
        if (c.getResponseCode() >= 400) throw new Exception("GitHub answered " + c.getResponseCode());
        return c;
    }

    static void install(Context ctx, File apk) throws Exception {
        PackageInstaller pi = ctx.getPackageManager().getPackageInstaller();
        PackageInstaller.SessionParams params = new PackageInstaller.SessionParams(PackageInstaller.SessionParams.MODE_FULL_INSTALL);
        params.setAppPackageName(ctx.getPackageName());
        if (Build.VERSION.SDK_INT >= 31) params.setRequireUserAction(PackageInstaller.SessionParams.USER_ACTION_NOT_REQUIRED);
        int id = pi.createSession(params);
        try (PackageInstaller.Session session = pi.openSession(id)) {
            try (InputStream in = new FileInputStream(apk); OutputStream out = session.openWrite("Lens.apk", 0, apk.length())) {
                byte[] buf = new byte[64 * 1024];
                for (int n; (n = in.read(buf)) > 0; ) out.write(buf, 0, n);
                session.fsync(out);
            }
            int flags = PendingIntent.FLAG_UPDATE_CURRENT | (Build.VERSION.SDK_INT >= 31 ? PendingIntent.FLAG_MUTABLE : 0);
            PendingIntent done = PendingIntent.getBroadcast(ctx, 7, new Intent(ctx, UpdateReceiver.class), flags);
            session.commit(done.getIntentSender());
        }
    }

    static void notify(Context ctx, int id, String title, String text, PendingIntent tap) {
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        nm.createNotificationChannel(new NotificationChannel(CHANNEL, "App updates", NotificationManager.IMPORTANCE_DEFAULT));
        Notification n = new Notification.Builder(ctx, CHANNEL)
                .setSmallIcon(android.R.drawable.stat_sys_download_done)
                .setContentTitle(title)
                .setContentText(text)
                .setContentIntent(tap)
                .setAutoCancel(true)
                .build();
        nm.notify(id, n);
    }
}
