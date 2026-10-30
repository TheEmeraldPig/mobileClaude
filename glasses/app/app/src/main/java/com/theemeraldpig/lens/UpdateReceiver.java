package com.theemeraldpig.lens;

import android.app.PendingIntent;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageInstaller;
import android.util.Log;

/**
 * Hears how an update install went, and when Lens was just updated or the phone restarted.
 * After either, Android only lets the microphone service start from the app's own screen,
 * so a notification asks you to open Lens once.
 */
public class UpdateReceiver extends BroadcastReceiver {
    @Override
    public void onReceive(Context ctx, Intent intent) {
        String action = intent.getAction();
        PendingIntent openApp = PendingIntent.getActivity(ctx, 0, new Intent(ctx, MainActivity.class), PendingIntent.FLAG_IMMUTABLE);
        if (Intent.ACTION_MY_PACKAGE_REPLACED.equals(action)) {
            Updater.notify(ctx, 2, "Lens was updated", "Now build " + Updater.currentBuild(ctx)
                    + ". Tap to open it so it can listen for the hub again.", openApp);
            return;
        }
        if (Intent.ACTION_BOOT_COMPLETED.equals(action)) {
            Updater.notify(ctx, 2, "Open Lens", "Tap so Lens can listen for the hub (Android needs the app opened once after a restart).", openApp);
            return;
        }
        int status = intent.getIntExtra(PackageInstaller.EXTRA_STATUS, PackageInstaller.STATUS_FAILURE);
        if (status == PackageInstaller.STATUS_PENDING_USER_ACTION) {
            // first update: Android wants you to confirm it once
            Intent confirm = intent.getParcelableExtra(Intent.EXTRA_INTENT);
            if (confirm == null) return;
            confirm.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            if (MainActivity.visible) {
                ctx.startActivity(confirm);
            } else {
                PendingIntent tap = PendingIntent.getActivity(ctx, 3, confirm, PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT);
                Updater.notify(ctx, 3, "Lens update ready", "Tap to install the new version.", tap);
            }
        } else if (status != PackageInstaller.STATUS_SUCCESS) {
            String msg = intent.getStringExtra(PackageInstaller.EXTRA_STATUS_MESSAGE);
            Log.w(Updater.TAG, "update failed: " + status + " " + msg);
            ctx.getSharedPreferences("lens", Context.MODE_PRIVATE).edit().putString("updateError", status + ": " + msg).apply();
        }
    }
}
