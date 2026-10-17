// Lens glasses firmware for the Seeed Studio XIAO ESP32S3 Sense.
//
// The glasses only see and listen for the button; the pocket hub does the thinking.
//   GET  http://lens-glasses.local/capture  -> a JPEG of what you're looking at (recording light on while it's taken)
//   GET  http://lens-glasses.local/status   -> {"ok":true,...}
//   On connecting (and every 30 s) it tells the hub its address: hub /hello {"ip":"..."}
//   Button: short press -> hub /button {"event":"press"} (talk / stop)
//           long press  -> hub /button {"event":"long"}  (take a photo)
//
// Arduino IDE: board "XIAO_ESP32S3", Tools -> PSRAM -> "OPI PSRAM". Copy secrets.example.h to secrets.h first.
// Wiring: button between D1 and GND; recording LED (with a 220 ohm resistor) from D0 to GND.

#include <WiFi.h>
#include <WebServer.h>
#include <ESPmDNS.h>
#include <HTTPClient.h>
#include "esp_camera.h"
#include "secrets.h"   // WIFI_SSID, WIFI_PASSWORD, HUB_HOST, HUB_PORT, HUB_TOKEN

const int BUTTON_PIN = D1;
const int LED_PIN = D0;
const unsigned long LONG_PRESS_MS = 700;

WebServer server(80);
IPAddress hubIp;

bool startCamera() {
  camera_config_t c = {};
  c.ledc_channel = LEDC_CHANNEL_0;
  c.ledc_timer = LEDC_TIMER_0;
  // XIAO ESP32S3 Sense camera pins
  c.pin_pwdn = -1;  c.pin_reset = -1;  c.pin_xclk = 10;
  c.pin_sccb_sda = 40;  c.pin_sccb_scl = 39;
  c.pin_d7 = 48;  c.pin_d6 = 11;  c.pin_d5 = 12;  c.pin_d4 = 14;
  c.pin_d3 = 16;  c.pin_d2 = 18;  c.pin_d1 = 17;  c.pin_d0 = 15;
  c.pin_vsync = 38;  c.pin_href = 47;  c.pin_pclk = 13;
  c.xclk_freq_hz = 20000000;
  c.pixel_format = PIXFORMAT_JPEG;
  c.frame_size = FRAMESIZE_SVGA;   // 800x600: sharp enough to read signs, small enough to send fast
  c.jpeg_quality = 12;
  c.fb_count = 2;
  c.fb_location = CAMERA_FB_IN_PSRAM;
  c.grab_mode = CAMERA_GRAB_LATEST;
  return esp_camera_init(&c) == ESP_OK;
}

void handleCapture() {
  digitalWrite(LED_PIN, HIGH);
  camera_fb_t *fb = esp_camera_fb_get();   // drop a possibly stale frame
  if (fb) esp_camera_fb_return(fb);
  fb = esp_camera_fb_get();
  if (!fb) {
    digitalWrite(LED_PIN, LOW);
    server.send(500, "text/plain", "camera error");
    return;
  }
  server.setContentLength(fb->len);
  server.send(200, "image/jpeg", "");
  server.client().write(fb->buf, fb->len);
  esp_camera_fb_return(fb);
  delay(150);   // keep the light visible for a moment
  digitalWrite(LED_PIN, LOW);
}

void handleStatus() {
  String s = "{\"ok\":true,\"rssi\":" + String(WiFi.RSSI()) + ",\"uptime\":" + String(millis() / 1000) + "}";
  server.send(200, "application/json", s);
}

// HUB_HOST "gateway": the hub is the phone whose hotspot we're on. Otherwise it's a name like "lens-hub" (a Pi).
bool findHub() {
  if (String(HUB_HOST) == "gateway") hubIp = WiFi.gatewayIP();
  else hubIp = MDNS.queryHost(HUB_HOST, 2000);
  return hubIp != IPAddress(0, 0, 0, 0);
}

int postToHub(const char *path, const String &body) {
  if (hubIp == IPAddress(0, 0, 0, 0) && !findHub()) return -1;
  HTTPClient http;
  http.begin("http://" + hubIp.toString() + ":" + String(HUB_PORT) + path);
  http.setTimeout(2000);
  http.addHeader("Content-Type", "application/json");
  http.addHeader("X-Lens-Token", HUB_TOKEN);
  int code = http.POST(body);
  http.end();
  if (code <= 0) hubIp = IPAddress(0, 0, 0, 0);   // the hub moved; look it up again next time
  return code;
}

void sendButton(const char *event) {
  postToHub("/button", String("{\"event\":\"") + event + "\"}");
}

// Tells the hub our address, so it can fetch pictures whatever address the hotspot gave us.
void sendHello() {
  postToHub("/hello", "{\"ip\":\"" + WiFi.localIP().toString() + "\"}");
}

void blink(int times) {
  for (int i = 0; i < times; i++) {
    digitalWrite(LED_PIN, HIGH); delay(80);
    digitalWrite(LED_PIN, LOW);  delay(120);
  }
}

void setup() {
  pinMode(BUTTON_PIN, INPUT_PULLUP);
  pinMode(LED_PIN, OUTPUT);
  Serial.begin(115200);

  if (!startCamera()) {
    Serial.println("camera failed: is PSRAM set to OPI PSRAM?");
    while (true) blink(3), delay(1000);
  }

  WiFi.mode(WIFI_STA);
  WiFi.setSleep(true);   // saves battery between requests
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  while (WiFi.status() != WL_CONNECTED) delay(250);
  Serial.println("Wi-Fi: " + WiFi.localIP().toString());

  MDNS.begin("lens-glasses");
  MDNS.addService("http", "tcp", 80);
  server.on("/capture", handleCapture);
  server.on("/status", handleStatus);
  server.begin();
  findHub();
  sendHello();
  blink(2);   // ready
}

void loop() {
  server.handleClient();

  static bool wasDown = false;
  static unsigned long downAt = 0;
  bool down = digitalRead(BUTTON_PIN) == LOW;
  if (down && !wasDown) downAt = millis();
  if (!down && wasDown) {
    unsigned long held = millis() - downAt;
    if (held > 30) sendButton(held >= LONG_PRESS_MS ? "long" : "press");
  }
  wasDown = down;

  static unsigned long lastHello = 0;
  if (millis() - lastHello > 30000) {   // also covers the hub restarting
    lastHello = millis();
    if (WiFi.status() == WL_CONNECTED) sendHello();
  }

  if (WiFi.status() != WL_CONNECTED) WiFi.reconnect(), delay(500);
  delay(5);
}
