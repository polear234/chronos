package com.chronos.ide;

import android.content.ContentProvider;
import android.content.ContentValues;
import android.database.Cursor;
import android.net.Uri;
import android.os.ParcelFileDescriptor;

import java.io.File;
import java.io.FileNotFoundException;

/**
 * 极简 ContentProvider：把打包好的 APK 以 content:// 形式暴露给系统安装器。
 *
 * Android 7.0+ 禁止用 file:// URI 触发安装（FileUriExposedException），
 * 而 AndroidX 的 FileProvider 会引入依赖，这里自己实现一个。
 */
public class ApkProvider extends ContentProvider {

    public static final String AUTHORITY = "com.chronos.ide.apk";
    public static final Uri CONTENT_URI = Uri.parse("content://" + AUTHORITY + "/app.apk");

    @Override
    public boolean onCreate() {
        return true;
    }

    @Override
    public ParcelFileDescriptor openFile(Uri uri, String mode) throws FileNotFoundException {
        File f = builtApk(getContext().getCacheDir());
        if (!f.exists()) {
            throw new FileNotFoundException("APK not built yet: " + f);
        }
        return ParcelFileDescriptor.open(f, ParcelFileDescriptor.MODE_READ_ONLY);
    }

    /** 打包产物固定路径，Provider 与 MainActivity 共用 */
    public static File builtApk(File cacheDir) {
        return new File(new File(cacheDir, "apkbuild"), "signed.apk");
    }

    // ---- 以下方法本 Provider 不需要 ----

    @Override
    public Cursor query(Uri uri, String[] projection, String selection,
                        String[] selectionArgs, String sortOrder) {
        return null;
    }

    @Override
    public String getType(Uri uri) {
        return "application/vnd.android.package-archive";
    }

    @Override
    public Uri insert(Uri uri, ContentValues values) {
        return null;
    }

    @Override
    public int delete(Uri uri, String selection, String[] selectionArgs) {
        return 0;
    }

    @Override
    public int update(Uri uri, ContentValues values, String selection, String[] selectionArgs) {
        return 0;
    }
}
