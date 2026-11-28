# Chronospp Windows 一键安装脚本
# 用法（PowerShell）：
#   .\install.ps1                      安装到用户目录 %LOCALAPPDATA%\chpp，并写入用户 PATH
#   .\install.ps1 -Scope Machine       安装到 C:\Program Files\chpp（需要管理员）
#   .\install.ps1 -InstallDir D:\tools  安装到自定义目录
#
# 自动检测 CPU 架构（amd64/arm64），选择对应二进制。

param(
    [ValidateSet("User", "Machine")]
    [string]$Scope = "User",
    [string]$InstallDir = "",
    [switch]$Force
)

$ErrorActionPreference = "Stop"

$Version = "1.1.0"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# 优先脚本同目录（发布包），其次 ../bin（源码树）
if (Get-ChildItem -Path $ScriptDir -Filter "chpp-windows-*.exe" -ErrorAction SilentlyContinue) {
    $BinDir = $ScriptDir
} else {
    $BinDir = Join-Path $ScriptDir "bin"
}

# ---- 检测架构 ----
$Arch = $env:PROCESSOR_ARCHITECTURE
switch ($Arch) {
    "AMD64" { $Binary = "chpp-windows-amd64.exe" }
    "ARM64" { $Binary = "chpp-windows-arm64.exe" }
    default {
        Write-Host "✗ 不支持的架构: $Arch" -ForegroundColor Red
        Write-Host "  支持的架构: AMD64, ARM64"
        exit 1
    }
}

# ---- 确定安装目录 ----
if ($InstallDir -eq "") {
    if ($Scope -eq "Machine") {
        $InstallDir = Join-Path $env:ProgramFiles "chpp"
    } else {
        $InstallDir = Join-Path $env:LOCALAPPDATA "chpp"
    }
}

$Src = Join-Path $BinDir $Binary
if (-not (Test-Path $Src)) {
    Write-Host "✗ 找不到二进制文件: $Src" -ForegroundColor Red
    Write-Host "  源码安装：在项目根目录执行 make dist" -ForegroundColor Yellow
    Write-Host "  发布包：确保 install.ps1 与 chpp-windows-*.exe 在同一目录" -ForegroundColor Yellow
    exit 1
}

# ---- 校验二进制版本（版本号编译期注入，旧构建会显示旧值）----
$Expect = "Chronospp $Version"
$Actual = ""
try { $Actual = (& $Src version 2>$null | Select-Object -First 1) } catch { $Actual = "" }
if ($Actual -ne $Expect) {
    Write-Host "✗ 二进制版本不匹配，已中止安装" -ForegroundColor Red
    Write-Host "    期望: $Expect"
    Write-Host "    实际: $(if ($Actual) { $Actual } else { '（无法执行该文件）' })"
    Write-Host "    文件: $Src"
    Write-Host ""
    if (-not $Force) {
        Write-Host "  原因：版本号在编译期由 -ldflags 注入，旧二进制不会自动变新。" -ForegroundColor Yellow
        Write-Host "  修复（源码）：在项目根目录执行 make dist" -ForegroundColor Yellow
        Write-Host "  修复（发布包）：重新下载 v$Version 的文件，别混用旧目录" -ForegroundColor Yellow
        Write-Host "  确认要装旧二进制：加 -Force" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "  -Force 已指定，忽略版本不符继续安装。" -ForegroundColor Yellow
}

# ---- 安装 ----
Write-Host "==> Chronospp $Version 安装" -ForegroundColor Cyan
Write-Host "    架构: $Arch"
Write-Host "    目标: $InstallDir"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$Dest = Join-Path $InstallDir "chpp.exe"

try {
    Copy-Item -Path $Src -Destination $Dest -Force
    Write-Host "==> 已复制到 $Dest" -ForegroundColor Green
} catch {
    Write-Host "✗ 复制失败，尝试管理员权限..." -ForegroundColor Yellow
    Start-Process powershell -Verb RunAs -ArgumentList @(
        "-Command", "Copy-Item -Path '$Src' -Destination '$Dest' -Force"
    ) -Wait
}

# ---- 写入 PATH ----
function Add-ToUserPath($dir) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$dir*") {
        $newPath = if ($userPath) { "$userPath;$dir" } else { "$dir" }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host "==> 已写入用户 PATH: $dir" -ForegroundColor Green
    } else {
        Write-Host "==> PATH 已包含: $dir"
    }
}

function Add-ToMachinePath($dir) {
    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($machinePath -notlike "*$dir*") {
        $newPath = if ($machinePath) { "$machinePath;$dir" } else { "$dir" }
        try {
            [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
            Write-Host "==> 已写入系统 PATH: $dir" -ForegroundColor Green
        } catch {
            Write-Host "⚠ 写入系统 PATH 需要管理员权限" -ForegroundColor Yellow
            Write-Host "  请以管理员身份重新运行: .\install.ps1 -Scope Machine"
        }
    }
}

if ($Scope -eq "Machine") {
    Add-ToMachinePath $InstallDir
} else {
    Add-ToUserPath $InstallDir
}

# ---- 刷新当前进程 PATH ----
$env:Path = "$InstallDir;$env:Path"

# ---- 验证 ----
Write-Host ""
if (Get-Command chpp -ErrorAction SilentlyContinue) {
    Write-Host "✓ 安装成功！" -ForegroundColor Green
    & $Dest version
} else {
    Write-Host "✓ 已安装到 $Dest" -ForegroundColor Green
    Write-Host "  ⚠ 当前窗口可能未刷新 PATH，请重新打开终端后使用 chpp" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "快速开始:"
Write-Host "  chpp                      # 启动 REPL"
Write-Host "  chpp -run hello.chpp    # 运行脚本"
