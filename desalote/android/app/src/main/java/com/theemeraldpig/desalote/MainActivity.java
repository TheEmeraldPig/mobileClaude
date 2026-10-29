package com.theemeraldpig.desalote;

import android.app.Activity;
import android.content.Context;
import android.net.wifi.WifiManager;
import android.os.Bundle;
import android.view.View;

import com.theemeraldpig.desalote.mobile.Mobile;

import go.Seq;

public class MainActivity extends Activity {
    private GameView view;
    private Updater updater;
    private WifiManager.MulticastLock wifiLock;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // Ebitengine needs the app context to read the screen size and density
        Seq.setContext(getApplicationContext());
        // Tell the game where to keep its settings and save, before its first frame
        Mobile.setDataDir(getFilesDir().getAbsolutePath());
        setContentView(R.layout.main);
        view = findViewById(R.id.ebitenview);
        hideSystemBars();
        updater = new Updater(this);
        updater.start();
        // Many phones drop Wi-Fi broadcasts to save battery unless an app asks for
        // them; multiplayer finds worlds on the same Wi-Fi through them
        try {
            WifiManager wm = (WifiManager) getApplicationContext().getSystemService(Context.WIFI_SERVICE);
            if (wm != null) {
                wifiLock = wm.createMulticastLock("desalote-find-games");
                wifiLock.setReferenceCounted(false);
                wifiLock.acquire();
            }
        } catch (Exception e) {
            // finding worlds may not work, but joining by address still does
        }
    }

    @Override
    protected void onDestroy() {
        if (wifiLock != null && wifiLock.isHeld()) wifiLock.release();
        super.onDestroy();
    }

    @Override
    protected void onPause() {
        super.onPause();
        Mobile.save();
        view.suspendGame();
    }

    @Override
    protected void onResume() {
        super.onResume();
        view.resumeGame();
        hideSystemBars();
        if (updater != null) updater.onResume();
    }

    @Override
    public void onWindowFocusChanged(boolean hasFocus) {
        super.onWindowFocusChanged(hasFocus);
        if (hasFocus) hideSystemBars();
    }

    // Full screen: hide the status and navigation bars (swipe from an edge to show them briefly)
    private void hideSystemBars() {
        getWindow().getDecorView().setSystemUiVisibility(
            View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
            | View.SYSTEM_UI_FLAG_FULLSCREEN
            | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
            | View.SYSTEM_UI_FLAG_LAYOUT_STABLE
            | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
            | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN);
    }
}
