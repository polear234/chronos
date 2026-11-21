# PicoX Windows 一键安装脚本
# 用法（PowerShell）：
#   .\install.ps1                      安装到用户目录 %LOCALAPPDATA%\picox，并写入用户 PATH
#   .\install.ps1 -Scope Machine       安装到 C:\Program Files\picox（需要管理员）
#   .\install.ps1 -InstallDir D:\tools  安装到自定义目录
#
# 自动检测 CPU 架构（amd64/arm64），选择对应二进制。

param(
    [ValidateSet("User", "Machine")]
    [string]$Scope = "User",
    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

$Version = "0.1.0"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BinDir = Join-Path $ScriptDir "bin"

# ---- 检测架构 ----
$Arch = $env:PROCESSOR_ARCHITECTURE
switch ($Arch) {
    "AMD64" { $Binary = "picox-windows-amd64.exe" }
    "ARM64" { $Binary = "picox-windows-arm64.exe" }
    default {
        Write-Host "✗ 不支持的架构: $Arch" -ForegroundColor Red
        Write-Host "  支持的架构: AMD64, ARM64"
        exit 1
    }
}

# ---- 确定安装目录 ----
if ($InstallDir -eq "") {
    if ($Scope -eq "Machine") {
        $InstallDir = Join-Path $env:ProgramFiles "picox"
    } else {
        $InstallDir = Join-Path $env:LOCALAPPDATA "picox"
    }
}

$Src = Join-Path $BinDir $Binary
if (-not (Test-Path $Src)) {
    Write-Host "✗ 找不到二进制文件: $Src" -ForegroundColor Red
    Write-Host "  请先在项目根目录执行: make dist" -ForegroundColor Yellow
    exit 1
}

# ---- 安装 ----
Write-Host "==> PicoX $Version 安装" -ForegroundColor Cyan
Write-Host "    架构: $Arch"
Write-Host "    目标: $InstallDir"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$Dest = Join-Path $InstallDir "picox.exe"

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
if (Get-Command picox -ErrorAction SilentlyContinue) {
    Write-Host "✓ 安装成功！" -ForegroundColor Green
    & $Dest version
} else {
    Write-Host "✓ 已安装到 $Dest" -ForegroundColor Green
    Write-Host "  ⚠ 当前窗口可能未刷新 PATH，请重新打开终端后使用 picox" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "快速开始:"
Write-Host "  picox                  # 启动 REPL"
Write-Host "  picox -run hello.px    # 运行脚本"
