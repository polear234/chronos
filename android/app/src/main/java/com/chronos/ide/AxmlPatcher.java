package com.chronos.ide;

import java.io.ByteArrayOutputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;

/**
 * 修改已编译 APK 里 AndroidManifest.xml（二进制 AXML）的 package 属性。
 *
 * 思路：AXML 的 package 值是字符串池里的一个字符串。
 * 解析字符串池 → 替换包名 → 重建字符串池 → 拼接其余 chunk。
 * 这样支持任意长度的新包名（不要求与旧包名等长）。
 */
public final class AxmlPatcher {

    private static final int RES_STRING_POOL_TYPE = 0x0001;
    private static final int RES_XML_TYPE = 0x0003;
    private static final int UTF8_FLAG = 0x100;

    private AxmlPatcher() { }

    /**
     * @param axml      原始 AndroidManifest.xml（二进制）
     * @param oldPkg    旧包名（用于定位）
     * @param newPkg    新包名
     */
    public static byte[] patchPackage(byte[] axml, String oldPkg, String newPkg) {
        if (readU16(axml, 0) != RES_XML_TYPE) {
            throw new IllegalArgumentException("不是有效的 AXML");
        }
        int headerSize = readU16(axml, 2);
        int totalSize = readI32(axml, 4);

        int poolOff = headerSize;
        if (readU16(axml, poolOff) != RES_STRING_POOL_TYPE) {
            throw new IllegalArgumentException("找不到字符串池");
        }
        int poolHeaderSize = readU16(axml, poolOff + 2);
        int poolSize = readI32(axml, poolOff + 4);
        int stringCount = readI32(axml, poolOff + 8);
        int styleCount = readI32(axml, poolOff + 12);
        int flags = readI32(axml, poolOff + 16);
        int stringsStart = readI32(axml, poolOff + 20);
        boolean utf8 = (flags & UTF8_FLAG) != 0;
        boolean sorted = (flags & 0x1) != 0;

        List<String> strings = new ArrayList<>(stringCount);
        for (int i = 0; i < stringCount; i++) {
            int off = readI32(axml, poolOff + poolHeaderSize + i * 4);
            strings.add(decodeString(axml, poolOff + stringsStart + off, utf8));
        }

        // 定位并替换包名
        int idx = strings.indexOf(oldPkg);
        if (idx < 0) {
            // 兜底：找形如 a.b.c 且包含 oldPkg 最后一段的
            for (int i = 0; i < strings.size(); i++) {
                String s = strings.get(i);
                if (s != null && s.contains(".") && !s.contains("/")
                        && s.length() < 128 && looksLikePackage(s)) {
                    idx = i;
                    break;
                }
            }
        }
        if (idx < 0) {
            throw new IllegalStateException("未找到包名字符串: " + oldPkg);
        }
        strings.set(idx, newPkg);

        byte[] newPool = buildStringPool(strings, styleCount, utf8, sorted);

        // 拼接：XML header + 新字符串池 + 其余部分
        ByteArrayOutputStream out = new ByteArrayOutputStream(totalSize + 256);
        out.write(axml, 0, poolOff);
        out.write(newPool, 0, newPool.length);
        int restStart = poolOff + poolSize;
        out.write(axml, restStart, totalSize - restStart);

        byte[] result = out.toByteArray();
        writeI32(result, 4, result.length); // 更新根 chunksize
        return result;
    }

    private static boolean looksLikePackage(String s) {
        if (s.isEmpty() || s.charAt(0) == '.') return false;
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            boolean ok = (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
                    || (c >= '0' && c <= '9') || c == '_' || c == '.';
            if (!ok) return false;
        }
        return true;
    }

    // ---------------- 字符串池 ----------------

    private static byte[] buildStringPool(List<String> strings, int styleCount,
                                          boolean utf8, boolean sorted) {
        int n = strings.size();
        int[] offsets = new int[n];
        List<byte[]> datas = new ArrayList<>(n);
        int dataLen = 0;

        for (int i = 0; i < n; i++) {
            byte[] d = utf8 ? encodeUtf8(strings.get(i)) : encodeUtf16(strings.get(i));
            offsets[i] = dataLen;
            datas.add(d);
            dataLen += d.length;
        }

        int headerSize = 28;
        int offsetsSize = n * 4 + styleCount * 4;
        int stringsStart = headerSize + offsetsSize;
        int size = stringsStart + dataLen;
        // 4 字节对齐
        int padded = (size + 3) & ~3;

        byte[] out = new byte[padded];
        writeU16(out, 0, RES_STRING_POOL_TYPE);
        writeU16(out, 2, headerSize);
        writeI32(out, 4, padded);
        writeI32(out, 8, n);
        writeI32(out, 12, styleCount);
        writeI32(out, 16, (utf8 ? UTF8_FLAG : 0) | (sorted ? 0x1 : 0));
        writeI32(out, 20, stringsStart);
        writeI32(out, 24, 0); // stylesStart

        int p = headerSize;
        for (int i = 0; i < n; i++) {
            writeI32(out, p, offsets[i]);
            p += 4;
        }
        // 样式偏移（通常为 0）
        for (int i = 0; i < styleCount; i++) {
            writeI32(out, p, 0);
            p += 4;
        }
        int dp = stringsStart;
        for (int i = 0; i < n; i++) {
            byte[] d = datas.get(i);
            System.arraycopy(d, 0, out, dp, d.length);
            dp += d.length;
        }
        return out;
    }

    /** UTF-8 字符串：varint(字节长) + varint(字符长) + bytes + 0 */
    private static byte[] encodeUtf8(String s) {
        byte[] data = s.getBytes(StandardCharsets.UTF_8);
        int charLen = s.codePointCount(0, s.length());
        ByteArrayOutputStream o = new ByteArrayOutputStream(data.length + 4);
        writeVarint(o, data.length);
        writeVarint(o, charLen);
        o.write(data, 0, data.length);
        o.write(0);
        return o.toByteArray();
    }

    /** UTF-16 字符串：u16(字符长，必要时两个 u16) + UTF-16LE + 0x0000 */
    private static byte[] encodeUtf16(String s) {
        int len = s.length();
        ByteArrayOutputStream o = new ByteArrayOutputStream(len * 2 + 6);
        if (len > 0x7FFF) {
            // 0x8000 | 高 16 位，再写低 16 位
            writeU16(o, 0x8000 | (len >>> 16));
            writeU16(o, len & 0xFFFF);
        } else {
            writeU16(o, len);
        }
        for (int i = 0; i < len; i++) {
            char c = s.charAt(i);
            o.write(c & 0xFF);
            o.write((c >> 8) & 0xFF);
        }
        o.write(0);
        o.write(0);
        return o.toByteArray();
    }

    private static void writeU16(ByteArrayOutputStream o, int v) {
        o.write(v & 0xFF);
        o.write((v >> 8) & 0xFF);
    }

    private static void writeVarint(ByteArrayOutputStream o, int v) {
        if (v < 0x80) {
            o.write(v);
        } else {
            o.write((v & 0x7F) | 0x80);
            o.write((v >> 7) & 0xFF);
        }
    }

    private static String decodeString(byte[] b, int off, boolean utf8) {
        if (utf8) {
            int[] p = {off};
            readVarint(b, p);        // 字节长（忽略，按 0 结尾读）
            readVarint(b, p);        // 字符长（忽略）
            int start = p[0];
            int end = start;
            while (end < b.length && b[end] != 0) end++;
            return new String(b, start, end - start, StandardCharsets.UTF_8);
        } else {
            // UTF-16：长度是 u16；若最高位为 1，则用两个 u16 表示 32 位长度
            int len = readU16(b, off);
            int p = off + 2;
            if ((len & 0x8000) != 0) {
                int low = readU16(b, p);
                len = ((len & 0x7FFF) << 16) | low;
                p += 2;
            }
            char[] cs = new char[len];
            for (int i = 0; i < len; i++) {
                cs[i] = (char) ((b[p] & 0xFF) | ((b[p + 1] & 0xFF) << 8));
                p += 2;
            }
            return new String(cs);
        }
    }

    private static int readVarint(byte[] b, int[] p) {
        int v = b[p[0]++] & 0xFF;
        if ((v & 0x80) != 0) {
            v = ((v & 0x7F) << 8) | (b[p[0]++] & 0xFF);
        }
        return v;
    }

    // ---------------- 基础读写 ----------------

    private static int readU16(byte[] b, int off) {
        return (b[off] & 0xFF) | ((b[off + 1] & 0xFF) << 8);
    }

    private static int readI32(byte[] b, int off) {
        return (b[off] & 0xFF) | ((b[off + 1] & 0xFF) << 8)
                | ((b[off + 2] & 0xFF) << 16) | ((b[off + 3] & 0xFF) << 24);
    }

    private static void writeU16(byte[] b, int off, int v) {
        b[off] = (byte) (v & 0xFF);
        b[off + 1] = (byte) ((v >> 8) & 0xFF);
    }

    private static void writeI32(byte[] b, int off, int v) {
        b[off] = (byte) (v & 0xFF);
        b[off + 1] = (byte) ((v >> 8) & 0xFF);
        b[off + 2] = (byte) ((v >> 16) & 0xFF);
        b[off + 3] = (byte) ((v >> 24) & 0xFF);
    }
}
