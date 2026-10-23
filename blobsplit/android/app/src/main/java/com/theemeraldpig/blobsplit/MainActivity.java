package com.theemeraldpig.blobsplit;

import android.app.Activity;
import android.os.Bundle;
import android.view.View;

import com.theemeraldpig.blobsplit.mobile.Mobile;

import go.Seq;

public class MainActivity extends Activity {
    private GameView view;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // Ebitengine needs the app context to read the screen size and density
        Seq.setContext(getApplicationContext());
        // Tell the game where to keep its save, before its first frame
        Mobile.setDataDir(getFilesDir().getAbsolutePath());
        setContentView(R.layout.main);
        view = findViewById(R.id.ebitenview);
        hideSystemBars();
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
