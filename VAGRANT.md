# 🖥️ Vagrant Windows Development Environment

یک VM کامل Windows 11 با تمام ابزارهای توسعه PZ Launcher.

## 📋 پیش‌نیازها

### Windows
```powershell
# با Chocolatey
choco install virtualbox vagrant

# یا دانلود دستی:
# https://www.virtualbox.org/wiki/Downloads
# https://www.vagrantup.com/downloads
```

### macOS
```bash
brew install --cask virtualbox vagrant
```

### Linux (Ubuntu/Debian)
```bash
sudo apt update
sudo apt install virtualbox vagrant
```

---

## 🚀 شروع سریع

### ۱. ساخت VM (اولین بار ~10 دقیقه طول می‌کشد)
```bash
make vagrant-up
```

### ۲. ورود به VM
```bash
# روش 1: RDP (کامل GUI)
make vagrant-rdp

# روش 2: PowerShell (ترمینال)
make vagrant-ssh
```

### ۳. Build پروژه
```bash
# از داخل VM
make vagrant-build

# یا از هاست (cross-compile)
make vagrant-build-windows
```

---

## 📁 ساختار پوشه‌ها

| مسیر Host | مسیر VM | توضیح |
|-----------|---------|-------|
| `./` (پروژه) | `C:\Users\vagrant\project` | کد سورس (sync خودکار) |
| `./.vagrant-cache` | `C:\Users\vagrant\.cache` | کش npm/go |
| - | `C:\Users\vagrant\Desktop` | دسکتاپ VM |

---

## 🎮 دستورات اصلی

### مدیریت VM
```bash
make vagrant-up           # استارت VM
make vagrant-down         # استاپ VM
make vagrant-reload       # ری‌استارت VM
make vagrant-destroy      # حذف کامل VM
```

### دسترسی
```bash
make vagrant-rdp          # اتصال با RDP
make vagrant-ssh          # اتصال با PowerShell
```

### ساخت (Build)
```bash
make vagrant-build        # ساخت داخل VM
make vagrant-build-windows # ساخت از هاست
```

### Snapshot (ذخیره وضعیت)
```bash
make vagrant-snapshot-save     # ذخیره snapshot
make vagrant-snapshot-restore  # بازیابی snapshot
```

### راهنما
```bash
make vagrant-help         # نمایش همه دستورات
```

---

## ⚡ کار روزانه (Workflow)

### توسعه سریع
```bash
# 1. استارت VM (اگر خاموش است)
make vagrant-up

# 2. تغییر کد در IDE هاست (VSCode, Goland, ...)
# فایل‌ها خودکار sync می‌شوند!

# 3. Build در VM
make vagrant-build

# 4. تست در VM
make vagrant-rdp
# اجرای فایل: C:\Users\vagrant\project\dist\pz-launcher-windows-amd64.exe

# 5. استاپ VM (در انتهای روز)
make vagrant-down
```

### تست تمیز (Clean Test)
```bash
# Snapshot قبل از تست
make vagrant-snapshot-save
# Enter name: before-test

# تست چیزی که می‌خوای...

# برگشت به وضعیت اول
make vagrant-snapshot-restore
# Enter name: before-test
```

---

## 🔧 کانفیگ پیشرفته

### تنظیمات RAM/CPU
فایل `Vagrantfile` را ویرایش کن:
```ruby
config.vm.provider "virtualbox" do |vb|
  vb.memory = "8192"   # 8GB RAM
  vb.cpus = 6          # 6 Core CPU
end
```
بعد:
```bash
make vagrant-reload
```

### پورت‌های اضافی
```ruby
# در Vagrantfile
config.vm.network "forwarded_port", guest: 3000, host: 3000
```

### فولدر sync اضافی
```ruby
config.vm.synced_folder "./my-mods", "C:/Users/vagrant/mods"
```

---

## 🐛 عیب‌یابی

### مشکل: VM خیلی کند است
```bash
# بستن VM
make vagrant-down

# افزایش RAM/CPU در Vagrantfile
# سپس:
make vagrant-up
```

### مشکل: Sync فولدر کار نمی‌کند
```bash
# ری‌استارت سرویس sync
vagrant powershell
cd C:\Users\vagrant\project
dir  # باید فایل‌ها را ببینید
```

### مشکل: Windows guest additions قدیمی است
```bash
vagrant plugin install vagrant-vbguest
vagrant vbguest --do install
```

### مشکل: RDP کار نمی‌کند
```bash
# RDP دستی:
vagrant rdp -- /cert:ignore /w:1920 /h:1080
```

---

## 📦 فایل‌های ساخته شده

پس از `make vagrant-build`:

| فایل | مسیر |
|------|------|
| اجرایی Windows | `C:\Users\vagrant\project\dist\pz-launcher-windows-amd64.exe` |
| Backend Linux | `C:\Users\vagrant\project\dist\pz-backend-linux-amd64` |
| Agent Linux | `C:\Users\vagrant\project\dist\pz-agent-linux-amd64` |

---

## 🌐 پورت‌ها

| سرویس | پورت Host | پورت VM | توضیح |
|-------|----------|---------|-------|
| Backend API | 8080 | 8080 | API سرور |
| Dev API | 8765 | 8765 | توسعه لوکال |
| RDP | 3389 | - | دسترسی GUI |

---

## 🗑️ حذف کامل

```bash
# حذف VM (همه فایل‌ها پاک می‌شوند)
make vagrant-destroy

# یا دستی:
vagrant destroy -f
rm -rf .vagrant/
```

---

## 💡 نکات

1. **اولین بوت**: ۵-۱۰ دقیقه طول می‌کشد (Windows setup + provisioning)
2. **بوت‌های بعدی**: ۱-۲ دقیقه
3. **Suspend**: `vagrant suspend` سریع‌تر از `halt` است
4. **Snapshot**: قبل از تست‌های خطرناک snapshot بگیرید

---

## 📚 منابع

- [Vagrant Docs](https://www.vagrantup.com/docs)
- [Wails Docs](https://wails.io/docs/gettingstarted/installation)
- [VirtualBox Manual](https://www.virtualbox.org/manual/)

---

**حاضری!** فقط `make vagrant-up` بزن و منتظر Windows 11 باش! 🎉
