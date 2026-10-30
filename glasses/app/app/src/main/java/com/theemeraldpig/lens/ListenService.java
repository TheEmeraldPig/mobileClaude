package com.theemeraldpig.lens;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.speech.RecognitionListener;
import android.speech.RecognizerIntent;
import android.speech.SpeechRecognizer;
import android.util.Log;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.InetAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

/**
 * Speech recognition for the hub. Termux's termux-speech-to-text stops at "end of speech" and never
 * reports Android's final result, so it drops the last words. This service waits for the final result.
 *
 * The hub (lens.py in Termux) talks to it on 127.0.0.1:8766, with the same token as the hub:
 *   GET  /status   -> {"ok":true,"listening":false,"available":true}
 *   POST /listen   -> {"text":"..."} once you've finished speaking
 *   POST /stop     -> finish now (the recognizer still returns everything said so far)
 */
public class ListenService extends Service {
    static final int PORT = 8766;
    static final String TAG = "LensListen";
    static final String CHANNEL = "listen";

    final Handler main = new Handler(Looper.getMainLooper());
    SpeechRecognizer recognizer;
    volatile boolean listening;
    ServerSocket server;

    @Override
    public void onCreate() {
        super.onCreate();
        NotificationManager nm = getSystemService(NotificationManager.class);
        nm.createNotificationChannel(new NotificationChannel(CHANNEL, "Listening for the glasses", NotificationManager.IMPORTANCE_LOW));
        PendingIntent open = PendingIntent.getActivity(this, 0, new Intent(this, MainActivity.class), PendingIntent.FLAG_IMMUTABLE);
        Notification n = new Notification.Builder(this, CHANNEL)
                .setSmallIcon(android.R.drawable.ic_btn_speak_now)
                .setContentTitle("Lens is ready")
                .setContentText("Hears your questions for the glasses hub.")
                .setContentIntent(open)
                .setOngoing(true)
                .build();
        if (Build.VERSION.SDK_INT >= 29) startForeground(1, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE);
        else startForeground(1, n);
        new Thread(this::serve, "lens-listen-server").start();
        main.post(updates);
    }

    /** While the service runs, look for a new version of the app every 30 minutes. */
    final Runnable updates = new Runnable() {
        @Override public void run() {
            Updater.check(ListenService.this, false, null);
            main.postDelayed(this, 30 * 60 * 1000);
        }
    };

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        return START_STICKY;
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onDestroy() {
        try {
            if (server != null) server.close();
        } catch (Exception ignored) {
        }
        main.post(() -> { if (recognizer != null) recognizer.destroy(); });
        super.onDestroy();
    }

    // ---------- tiny HTTP server, localhost only ----------
    void serve() {
        try {
            server = new ServerSocket(PORT, 4, InetAddress.getByName("127.0.0.1"));
            while (!server.isClosed()) {
                Socket s = server.accept();
                new Thread(() -> handle(s)).start();
            }
        } catch (Exception e) {
            Log.w(TAG, "server stopped: " + e);
        }
    }

    void handle(Socket s) {
        try (Socket sock = s) {
            sock.setSoTimeout(60000);
            BufferedReader in = new BufferedReader(new InputStreamReader(sock.getInputStream(), StandardCharsets.UTF_8));
            String first = in.readLine();
            if (first == null) return;
            String[] parts = first.split(" ");
            String method = parts[0], path = parts.length > 1 ? parts[1] : "/";
            String token = "";
            for (String h; (h = in.readLine()) != null && !h.isEmpty(); ) {
                int c = h.indexOf(':');
                if (c > 0 && h.substring(0, c).trim().equalsIgnoreCase("X-Lens-Token")) token = h.substring(c + 1).trim();
            }
            String expected = getSharedPreferences("lens", MODE_PRIVATE).getString("token", "");
            String body;
            int code = 200;
            if (expected.isEmpty() || !expected.equals(token)) {
                code = 403;
                body = "{\"error\":\"bad token\"}";
            } else if (path.startsWith("/status")) {
                body = "{\"ok\":true,\"listening\":" + listening + ",\"available\":" + SpeechRecognizer.isRecognitionAvailable(this) + "}";
            } else if (path.startsWith("/listen") && method.equals("POST")) {
                if (listening) {
                    code = 409;
                    body = "{\"error\":\"already listening\"}";
                } else {
                    body = listen();
                }
            } else if (path.startsWith("/stop") && method.equals("POST")) {
                main.post(() -> { if (recognizer != null && listening) recognizer.stopListening(); });
                body = "{\"ok\":true}";
            } else {
                code = 404;
                body = "{\"error\":\"not found\"}";
            }
            byte[] out = body.getBytes(StandardCharsets.UTF_8);
            OutputStream os = sock.getOutputStream();
            os.write(("HTTP/1.1 " + code + " OK\r\nContent-Type: application/json\r\nContent-Length: " + out.length
                    + "\r\nConnection: close\r\n\r\n").getBytes(StandardCharsets.UTF_8));
            os.write(out);
            os.flush();
        } catch (Exception e) {
            Log.w(TAG, "request failed: " + e);
        }
    }

    // ---------- speech ----------
    /** Listens once and returns {"text": ...}, waiting for Android's final result rather than the last partial one. */
    String listen() throws InterruptedException {
        final String[] best = {""};
        final String[] error = {null};
        final CountDownLatch done = new CountDownLatch(1);
        listening = true;
        main.post(() -> {
            if (recognizer == null) recognizer = SpeechRecognizer.createSpeechRecognizer(this);
            recognizer.setRecognitionListener(new RecognitionListener() {
                @Override public void onResults(Bundle r) {
                    ArrayList<String> got = r.getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION);
                    if (got != null && !got.isEmpty() && !got.get(0).trim().isEmpty()) best[0] = got.get(0);
                    done.countDown();
                }
                @Override public void onPartialResults(Bundle r) {
                    ArrayList<String> got = r.getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION);
                    if (got != null && !got.isEmpty() && got.get(0).length() >= best[0].length()) best[0] = got.get(0);
                }
                @Override public void onError(int e) {
                    // "no match" and "speech timeout" just mean nothing was said
                    if (e != SpeechRecognizer.ERROR_NO_MATCH && e != SpeechRecognizer.ERROR_SPEECH_TIMEOUT) error[0] = "speech error " + e;
                    done.countDown();
                }
                @Override public void onReadyForSpeech(Bundle b) {}
                @Override public void onBeginningOfSpeech() {}
                @Override public void onRmsChanged(float v) {}
                @Override public void onBufferReceived(byte[] b) {}
                @Override public void onEndOfSpeech() {}
                @Override public void onEvent(int t, Bundle b) {}
            });
            Intent i = new Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH);
            i.putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL, RecognizerIntent.LANGUAGE_MODEL_FREE_FORM);
            i.putExtra(RecognizerIntent.EXTRA_PARTIAL_RESULTS, true);
            i.putExtra(RecognizerIntent.EXTRA_MAX_RESULTS, 1);
            // ask for a longer pause before deciding you've finished (some recognizers ignore this)
            i.putExtra(RecognizerIntent.EXTRA_SPEECH_INPUT_COMPLETE_SILENCE_LENGTH_MILLIS, 2000L);
            i.putExtra(RecognizerIntent.EXTRA_SPEECH_INPUT_POSSIBLY_COMPLETE_SILENCE_LENGTH_MILLIS, 1500L);
            try {
                recognizer.startListening(i);
            } catch (Exception e) {
                error[0] = "couldn't start listening: " + e.getMessage();
                done.countDown();
            }
        });
        boolean finished = done.await(40, TimeUnit.SECONDS);
        if (!finished) main.post(() -> recognizer.cancel());
        listening = false;
        String text = best[0].replace("\\", "\\\\").replace("\"", "\\\"").replace("\n", " ").replace("\r", " ");
        return error[0] == null ? "{\"text\":\"" + text + "\"}"
                : "{\"text\":\"" + text + "\",\"error\":\"" + error[0].replace("\"", "'") + "\"}";
    }
}
