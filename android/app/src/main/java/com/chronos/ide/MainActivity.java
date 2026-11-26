package com.chronos.ide;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.DialogInterface;
import android.content.Intent;
import android.database.Cursor;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.provider.Settings;
import android.provider.OpenableColumns;
import android.view.ViewGroup;
import android.webkit.JavascriptInterface;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.JsPromptResult;
import android.webkit.WebChromeClient;
import android.webkit.WebViewClient;
import android.util.Base64;
import android.util.Log;
import android.widget.EditText;
import android.widget.TextView;
import android.widget.Toast;

import com.android.apksig.ApkSigner;
import com.android.apksig.ApkSigner.SignerConfig;

import java.io.BufferedOutputStream;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.security.KeyStore;
import java.security.PrivateKey;
import java.security.cert.X509Certificate;
import java.util.Collections;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;
import java.util.zip.ZipOutputStream;

/**
 * Chronos IDE —— WebView + Go WASM 运行时。
 *
 * JS 桥（ChronosBridge）：
 *   pickFile()                    导入脚本（.chr / 旧版 .px）
 *   saveFile(name, text)          导出源码
 *   compileApk(name, script)      把脚本打包成独立 APK（模板 + 重签名）
 */
public class MainActivity extends Activity {

    private static final String BASE_URL = "https://appassets.local/";
    /** 模板 APK 的原始包名 */
    private static final String TEMPLATE_PKG = "com.chronos.script";
    private static final int REQ_OPEN = 1001;
    private static final int REQ_SAVE = 1002;
    private static final int REQ_SAVE_APK = 1003;
    private static final int REQ_SAVE_FILE = 1004;

    private WebView webView;
    private String pendingSaveText = null;
    private File pendingApkFile = null;
    private File pendingSaveFile = null;
    private String pendingSaveMime = "application/octet-stream";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // 崩溃时把信息写到文件（尽力而为）
        final Thread.UncaughtExceptionHandler prev =
                Thread.getDefaultUncaughtExceptionHandler();
        Thread.setDefaultUncaughtExceptionHandler(new Thread.UncaughtExceptionHandler() {
            @Override public void uncaughtException(Thread t, Throwable e) {
                try {
                    File dir = getExternalFilesDir(null);
                    if (dir != null) {
                        FileOutputStream fos = new FileOutputStream(new File(dir, "crash.log"), true);
                        fos.write(("\n=== " + new java.util.Date() + " ===\n").getBytes("UTF-8"));
                        fos.write(Log.getStackTraceString(e).getBytes("UTF-8"));
                        fos.close();
                    }
                } catch (Throwable ignored) { }
                if (prev != null) prev.uncaughtException(t, e);
            }
        });

        // 先用一个文本视图占位：启动失败时把堆栈直接显示在屏幕上
        final TextView tv = new TextView(this);
        tv.setBackgroundColor(0xFF0D1117);
        tv.setTextColor(0xFF3DDCAE);
        tv.setTextSize(12);
        tv.setPadding(36, 96, 36, 36);
        tv.setTextIsSelectable(true);
        tv.setText("Chronos 启动中…\n\nAndroid " + Build.VERSION.RELEASE
                + " (API " + Build.VERSION.SDK_INT + ")");
        setContentView(tv);

        try {
            boot();
        } catch (Throwable e) {
            Log.e("Chronos", "启动失败", e);
            tv.setText("启动失败:\n\n" + Log.getStackTraceString(e));
            tv.setTextColor(0xFFFF7B72);
            setContentView(tv);
        }
    }

    private void boot() {
        webView = new WebView(this);
        WebSettings s = webView.getSettings();
        s.setJavaScriptEnabled(true);
        s.setDomStorageEnabled(true);
        s.setAllowFileAccess(true);
        s.setAllowContentAccess(true);
        s.setLoadWithOverviewMode(true);
        s.setUseWideViewPort(true);
        s.setTextZoom(100);
        webView.setBackgroundColor(0xFF0D1117);
        webView.addJavascriptInterface(new Bridge(), "ChronosBridge");

        webView.setWebViewClient(new WebViewClient() {
            @Override
            public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
                String host = request.getUrl().getHost();
                if (host == null || !host.equals("appassets.local")) {
                    return null;
                }
                String path = request.getUrl().getPath();
                if (path == null || path.isEmpty() || "/".equals(path)) {
                    path = "/index.html";
                }
                String assetPath = path.substring(1);
                try {
                    InputStream in = getAssets().open(assetPath);
                    String mime = mimeOf(assetPath);
                    String enc = (mime.startsWith("text/") || mime.contains("javascript")
                            || mime.contains("json")) ? "UTF-8" : null;
                    return new WebResourceResponse(mime, enc, in);
                } catch (IOException e) {
                    return null;
                }
            }
        });

        // JS prompt() → 原生输入框（供 Chronos 的 input() 使用）
        webView.setWebChromeClient(new WebChromeClient() {
            @Override
            public boolean onJsPrompt(WebView view, String url, String message,
                                      String defaultValue, final JsPromptResult result) {
                final EditText et = new EditText(MainActivity.this);
                et.setText(defaultValue == null ? "" : defaultValue);
                et.setSelection(et.getText().length());
                new AlertDialog.Builder(MainActivity.this)
                        .setTitle("Chronos 输入")
                        .setMessage(message == null ? "" : message)
                        .setView(et)
                        .setPositiveButton("确定", new DialogInterface.OnClickListener() {
                            @Override public void onClick(DialogInterface d, int w) {
                                result.confirm(et.getText().toString());
                            }
                        })
                        .setNegativeButton("取消", new DialogInterface.OnClickListener() {
                            @Override public void onClick(DialogInterface d, int w) {
                                result.cancel();
                            }
                        })
                        .setOnCancelListener(new DialogInterface.OnCancelListener() {
                            @Override public void onCancel(DialogInterface d) { result.cancel(); }
                        })
                        .show();
                return true;
            }
        });

        webView.loadUrl(BASE_URL + "index.html");
        setContentView(webView, new ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT));
    }

    // ---------------- JS 桥 ----------------

    private class Bridge {
        @JavascriptInterface
        public void pickFile() {
            runOnUiThread(new Runnable() {
                @Override public void run() {
                    Intent i = new Intent(Intent.ACTION_OPEN_DOCUMENT);
                    i.addCategory(Intent.CATEGORY_OPENABLE);
                    i.setType("*/*");
                    try {
                        startActivityForResult(i, REQ_OPEN);
                    } catch (Exception e) {
                        toast("无法打开文件选择器");
                    }
                }
            });
        }

        @JavascriptInterface
        public void saveFile(final String suggestedName, final String text) {
            runOnUiThread(new Runnable() {
                @Override public void run() {
                    pendingSaveText = text;
                    Intent i = new Intent(Intent.ACTION_CREATE_DOCUMENT);
                    i.addCategory(Intent.CATEGORY_OPENABLE);
                    i.setType("text/x-chronos");
                    i.putExtra(Intent.EXTRA_TITLE, suggestedName);
                    try {
                        startActivityForResult(i, REQ_SAVE);
                    } catch (Exception e) {
                        toast("无法打开保存对话框");
                    }
                }
            });
        }

        @JavascriptInterface
        public void compileApk(final String baseName, final String pkgName, final String script) {
            new Thread(new Runnable() {
                @Override public void run() {
                    try {
                        File dir = new File(getCacheDir(), "apkbuild");
                        if (!dir.exists() && !dir.mkdirs()) {
                            throw new IOException("无法创建缓存目录");
                        }
                        File unsigned = new File(dir, "unsigned.apk");
                        File signed = new File(dir, "signed.apk");
                        if (signed.exists()) {
                            //noinspection ResultOfMethodCallIgnored
                            signed.delete();
                        }

                        String pkg = normalizePkg(pkgName);
                        rewriteTemplate(unsigned, script, pkg);
                        signApk(unsigned, signed);

                        final File out = signed;
                        runOnUiThread(new Runnable() {
                            @Override public void run() { autoInstall(out); }
                        });
                    } catch (final Exception e) {
                        postJs("onApkResult", false, "打包失败: " + e);
                    }
                }
            }).start();
        }

        /** 生成自包含的 sh 脚本（内嵌 arm64 解释器 + 源码） */
        @JavascriptInterface
        public void compileShell(final String baseName, final String script) {
            new Thread(new Runnable() {
                @Override public void run() {
                    try {
                        InputStream in = getAssets().open("chr-linux-arm64");
                        byte[] bin = readAllBytes(in);
                        in.close();
                        String b64 = Base64.encodeToString(bin, Base64.NO_WRAP);

                        StringBuilder sb = new StringBuilder(b64.length() + script.length() + 512);
                        sb.append("#!/bin/sh\n");
                        sb.append("# 由 Chronos IDE 生成：自包含可执行脚本\n");
                        sb.append("set -e\n");
                        sb.append("WORK=\"$(mktemp -d \"${TMPDIR:-/tmp}/chronos.XXXXXX\")\"\n");
                        sb.append("trap 'rm -rf \"$WORK\"' EXIT INT TERM\n");
                        sb.append("base64 -d > \"$WORK/chr\" <<'__CHRONOS_BIN__'\n");
                        sb.append(b64).append("\n");
                        sb.append("__CHRONOS_BIN__\n");
                        sb.append("chmod +x \"$WORK/chr\"\n");
                        sb.append("cat > \"$WORK/app.chr\" <<'__CHRONOS_SRC__'\n");
                        sb.append(script).append("\n");
                        sb.append("__CHRONOS_SRC__\n");
                        sb.append("\"$WORK/chr\" -run \"$WORK/app.chr\"\n");

                        File out = new File(getCacheDir(), "app.sh");
                        FileOutputStream fos = new FileOutputStream(out);
                        fos.write(sb.toString().getBytes(StandardCharsets.UTF_8));
                        fos.close();

                        final File f = out;
                        final String name = shellName(baseName);
                        runOnUiThread(new Runnable() {
                            @Override public void run() {
                                startSaveFile(f, name, "text/x-shellscript");
                            }
                        });
                    } catch (final Exception e) {
                        postJs("onShellResult", false, "生成失败: " + e);
                    }
                }
            }).start();
        }

        @JavascriptInterface
        public void toastMsg(final String msg) {
            runOnUiThread(new Runnable() {
                @Override public void run() { toast(msg); }
            });
        }
    }

    // ---------------- APK 打包 ----------------

    /**
     * 复制模板 APK，替换 assets/script.chr（兼容旧版 script.px），并剔除旧的签名文件。
     *
     * 关键：Android 11+ 要求 targetSdk>=30 的 APK 中 resources.arsc
     * 必须是 STORED（不压缩）且 4 字节对齐。做法是把它作为第一个条目、
     * 用 STORED 且不加 extra field —— 此时数据偏移 = 30(local header)
     * + 14("resources.arsc") = 44，天然是 4 的倍数。
     */
    private void rewriteTemplate(File outFile, String script, String pkgName) throws IOException {
        // 1) 全部读入内存（模板约 1.3M，可接受）
        java.util.LinkedHashMap<String, byte[]> entries = new java.util.LinkedHashMap<>();
        try (ZipInputStream zin = new ZipInputStream(getAssets().open("template.apk"))) {
            byte[] buf = new byte[16384];
            ZipEntry e;
            while ((e = zin.getNextEntry()) != null) {
                String name = e.getName();
                // 丢弃旧签名，稍后重做
                if (name.startsWith("META-INF/")
                        && (name.equals("META-INF/MANIFEST.MF")
                        || name.endsWith(".SF") || name.endsWith(".RSA")
                        || name.endsWith(".DSA") || name.endsWith(".EC"))) {
                    continue;
                }
                ByteArrayOutputStream bos = new ByteArrayOutputStream(
                        e.getSize() > 0 ? (int) e.getSize() : 8192);
                int n;
                while ((n = zin.read(buf)) > 0) {
                    bos.write(buf, 0, n);
                }
                byte[] data = bos.toByteArray();
                if ("AndroidManifest.xml".equals(name) && pkgName != null
                        && !pkgName.isEmpty() && !pkgName.equals(TEMPLATE_PKG)) {
                    data = AxmlPatcher.patchPackage(data, TEMPLATE_PKG, pkgName);
                }
                entries.put(name, data);
            }
        }
        entries.put("assets/script.chr", script.getBytes(StandardCharsets.UTF_8));

        // 2) 写出：resources.arsc 优先且 STORED
        try (ZipOutputStream zout = new ZipOutputStream(
                new BufferedOutputStream(new FileOutputStream(outFile)))) {
            byte[] arsc = entries.remove("resources.arsc");
            if (arsc != null) {
                writeStored(zout, "resources.arsc", arsc);
            }
            for (java.util.Map.Entry<String, byte[]> en : entries.entrySet()) {
                ZipEntry ne = new ZipEntry(en.getKey());
                zout.putNextEntry(ne);
                zout.write(en.getValue());
                zout.closeEntry();
            }
        }
    }

    /** 以 STORED（不压缩）方式写入一个条目 */
    private static void writeStored(ZipOutputStream zout, String name, byte[] data) throws IOException {
        ZipEntry ne = new ZipEntry(name);
        ne.setMethod(ZipEntry.STORED);
        ne.setSize(data.length);
        ne.setCompressedSize(data.length);
        java.util.zip.CRC32 crc = new java.util.zip.CRC32();
        crc.update(data);
        ne.setCrc(crc.getValue());
        zout.putNextEntry(ne);
        zout.write(data);
        zout.closeEntry();
    }

    /** 用内置 keystore 重新签名（v1 + v2） */
    private void signApk(File in, File out) throws Exception {
        KeyStore ks = KeyStore.getInstance("PKCS12");
        InputStream ksIn = getAssets().open("chronos.keystore");
        try {
            ks.load(ksIn, "android".toCharArray());
        } finally {
            ksIn.close();
        }
        PrivateKey key = (PrivateKey) ks.getKey("chronos", "android".toCharArray());
        X509Certificate cert = (X509Certificate) ks.getCertificate("chronos");

        // 注意：SignerConfig.Builder.setMinSdkVersion 是「密钥轮换 lineage 的目标 SDK」，
        // 要求 >= 28；这里不需要，APK 的 minSdk 由 ApkSigner.Builder 设置。
        SignerConfig sc = new SignerConfig.Builder(
                "chronos", key, Collections.singletonList(cert))
                .build();

        new ApkSigner.Builder(Collections.singletonList(sc))
                .setInputApk(in)
                .setOutputApk(out)
                .setV1SigningEnabled(true)
                .setV2SigningEnabled(true)
                .setV3SigningEnabled(false)
                .setMinSdkVersion(26)
                .build()
                .sign();
    }

    /** 校验并规范化包名；非法则回退到模板包名 */
    static String normalizePkg(String raw) {
        if (raw == null) return TEMPLATE_PKG;
        String p = raw.trim();
        if (p.isEmpty()) return TEMPLATE_PKG;
        if (p.length() > 100) return TEMPLATE_PKG;
        int segStart = 0;
        for (int i = 0; i <= p.length(); i++) {
            if (i == p.length() || p.charAt(i) == '.') {
                if (i == segStart) return TEMPLATE_PKG;        // 空段
                char c0 = p.charAt(segStart);
                if (!Character.isLetter(c0)) return TEMPLATE_PKG;
                for (int k = segStart; k < i; k++) {
                    char c = p.charAt(k);
                    if (!(Character.isLetterOrDigit(c) || c == '_')) return TEMPLATE_PKG;
                }
                segStart = i + 1;
            }
        }
        if (!p.contains(".")) return TEMPLATE_PKG;
        return p;
    }

    private static String apkName(String base) {
        if (base == null || base.trim().isEmpty()) {
            return "chronos-app.apk";
        }
        String n = base.trim();
        if (n.endsWith(".chr")) {
            n = n.substring(0, n.length() - 4);
        } else if (n.endsWith(".px")) {
            n = n.substring(0, n.length() - 3);
        }
        if (!n.endsWith(".apk")) {
            n = n + ".apk";
        }
        return n;
    }

    /** 编译完成后直接拉起系统安装器 */
    private void autoInstall(File apk) {
        if (!apk.exists()) {
            postJs("onApkResult", false, "APK 文件不存在");
            return;
        }
        // Android 8+ 需要「安装未知应用」权限
        if (Build.VERSION.SDK_INT >= 26 && !getPackageManager().canRequestPackageInstalls()) {
            postJs("onApkResult", false, "请先允许「安装未知应用」");
            try {
                Intent s = new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                        Uri.parse("package:" + getPackageName()));
                startActivity(s);
            } catch (Exception ignored) { }
            return;
        }
        try {
            Intent i = new Intent(Intent.ACTION_VIEW);
            i.setDataAndType(ApkProvider.CONTENT_URI, "application/vnd.android.package-archive");
            i.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
            i.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            startActivity(i);
            postJs("onApkResult", true, "已生成 APK，请在安装界面确认");
        } catch (Exception e) {
            postJs("onApkResult", false, "无法启动安装: " + e);
        }
    }

    private static String shellName(String base) {
        if (base == null || base.trim().isEmpty()) return "app.sh";
        String n = base.trim();
        if (n.endsWith(".chr")) n = n.substring(0, n.length() - 4);
        else if (n.endsWith(".px")) n = n.substring(0, n.length() - 3);
        if (!n.endsWith(".sh")) n = n + ".sh";
        return n;
    }

    private void startSaveFile(File f, String name, String mime) {
        pendingSaveFile = f;
        pendingSaveMime = mime;
        Intent i = new Intent(Intent.ACTION_CREATE_DOCUMENT);
        i.addCategory(Intent.CATEGORY_OPENABLE);
        i.setType(mime);
        i.putExtra(Intent.EXTRA_TITLE, name);
        try {
            startActivityForResult(i, REQ_SAVE_FILE);
        } catch (Exception e) {
            postJs("onShellResult", false, "无法打开保存对话框");
        }
    }

    private void startSaveApk(File apk, String name) {
        pendingApkFile = apk;
        Intent i = new Intent(Intent.ACTION_CREATE_DOCUMENT);
        i.addCategory(Intent.CATEGORY_OPENABLE);
        i.setType("application/vnd.android.package-archive");
        i.putExtra(Intent.EXTRA_TITLE, name);
        try {
            startActivityForResult(i, REQ_SAVE_APK);
        } catch (Exception e) {
            postJs("onApkResult", false, "无法打开保存对话框");
        }
    }

    // ---------------- 回调 ----------------

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (resultCode != RESULT_OK || data == null) {
            return;
        }
        Uri uri = data.getData();
        if (uri == null) {
            return;
        }

        if (requestCode == REQ_OPEN) {
            try (InputStream in = getContentResolver().openInputStream(uri)) {
                String content = readAll(in);
                String name = displayName(uri);
                postJs2("onFileImported", name, content);
            } catch (Exception e) {
                toast("导入失败: " + e.getMessage());
            }
        } else if (requestCode == REQ_SAVE) {
            String text = pendingSaveText == null ? "" : pendingSaveText;
            pendingSaveText = null;
            try (OutputStream out = getContentResolver().openOutputStream(uri)) {
                out.write(text.getBytes(StandardCharsets.UTF_8));
                out.flush();
                toast("已导出（" + text.length() + " 字节）");
            } catch (Exception e) {
                toast("导出失败: " + e.getMessage());
            }
        } else if (requestCode == REQ_SAVE_FILE) {
            File f = pendingSaveFile;
            pendingSaveFile = null;
            if (f == null) return;
            try (OutputStream out = getContentResolver().openOutputStream(uri)) {
                copyFile(f, out);
                out.flush();
                toast("已导出 shell 脚本");
                postJs("onShellResult", true, "已保存（" + f.length() + " 字节）");
            } catch (Exception e) {
                postJs("onShellResult", false, "保存失败: " + e.getMessage());
            }
        } else if (requestCode == REQ_SAVE_APK) {
            File apk = pendingApkFile;
            pendingApkFile = null;
            if (apk == null) {
                return;
            }
            try (OutputStream out = getContentResolver().openOutputStream(uri)) {
                copyFile(apk, out);
                out.flush();
                toast("APK 已保存");
                postJs("onApkResult", true, "APK 已保存（" + apk.length() + " 字节）");
            } catch (Exception e) {
                postJs("onApkResult", false, "保存失败: " + e.getMessage());
            }
        }
    }

    private static void copyFile(File f, OutputStream out) throws IOException {
        try (InputStream in = new java.io.FileInputStream(f)) {
            byte[] buf = new byte[16384];
            int n;
            while ((n = in.read(buf)) > 0) {
                out.write(buf, 0, n);
            }
        }
    }

    private static byte[] readAllBytes(InputStream in) throws IOException {
        ByteArrayOutputStream bos = new ByteArrayOutputStream();
        byte[] buf = new byte[16384];
        int n;
        while ((n = in.read(buf)) > 0) {
            bos.write(buf, 0, n);
        }
        return bos.toByteArray();
    }

    private static String readAll(InputStream in) throws IOException {
        ByteArrayOutputStream bos = new ByteArrayOutputStream();
        byte[] buf = new byte[8192];
        int n;
        while ((n = in.read(buf)) > 0) {
            bos.write(buf, 0, n);
        }
        return new String(bos.toByteArray(), StandardCharsets.UTF_8);
    }

    private String displayName(Uri uri) {
        String name = null;
        try (Cursor c = getContentResolver().query(uri, null, null, null, null)) {
            if (c != null && c.moveToFirst()) {
                int idx = c.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                if (idx >= 0) {
                    name = c.getString(idx);
                }
            }
        } catch (Exception ignored) { }
        if (name == null) {
            name = uri.getLastPathSegment();
        }
        return name == null ? "untitled.chr" : name;
    }

    private void postJs(String fn, boolean ok, String msg) {
        final String js = "window." + fn + "(" + ok + "," + jsStr(msg) + ")";
        webView.post(new Runnable() {
            @Override public void run() { webView.evaluateJavascript(js, null); }
        });
    }

    private void postJs2(String fn, String a, String b) {
        final String js = "window." + fn + "(" + jsStr(a) + "," + jsStr(b) + ")";
        webView.post(new Runnable() {
            @Override public void run() { webView.evaluateJavascript(js, null); }
        });
    }

    private static String jsStr(String s) {
        StringBuilder sb = new StringBuilder(s.length() + 16);
        sb.append('"');
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            switch (c) {
                case '"':  sb.append("\\\""); break;
                case '\\': sb.append("\\\\"); break;
                case '\n': sb.append("\\n"); break;
                case '\r': sb.append("\\r"); break;
                case '\t': sb.append("\\t"); break;
                case '\u2028': sb.append("\\u2028"); break;
                case '\u2029': sb.append("\\u2029"); break;
                default:
                    if (c < 0x20) {
                        sb.append(String.format("\\u%04x", (int) c));
                    } else {
                        sb.append(c);
                    }
            }
        }
        sb.append('"');
        return sb.toString();
    }

    private void toast(String msg) {
        Toast.makeText(this, msg, Toast.LENGTH_SHORT).show();
    }

    private static String mimeOf(String p) {
        if (p.endsWith(".html")) return "text/html";
        if (p.endsWith(".js")) return "application/javascript";
        if (p.endsWith(".wasm")) return "application/wasm";
        if (p.endsWith(".css")) return "text/css";
        if (p.endsWith(".json")) return "application/json";
        if (p.endsWith(".png")) return "image/png";
        if (p.endsWith(".svg")) return "image/svg+xml";
        if (p.endsWith(".apk")) return "application/vnd.android.package-archive";
        return "application/octet-stream";
    }

    @Override
    public void onBackPressed() {
        if (webView != null && webView.canGoBack()) {
            webView.goBack();
        } else {
            super.onBackPressed();
        }
    }
}
