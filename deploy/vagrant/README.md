# PZ Platform — VM Test Environment

## Prerequisites

### روی Windows:
1. [Vagrant](https://www.vagrantup.com/downloads) نصب کن
2. VMware Workstation داری → plugin نصب کن:
   ```
   vagrant plugin install vagrant-vmware-desktop
   ```
   یا VirtualBox (رایگان):
   [https://www.virtualbox.org/wiki/Downloads](https://www.virtualbox.org/wiki/Downloads)

---

## راه‌اندازی

### مرحله ۱ — Backend رو بالا بیار (Docker)
```bash
make test-stack
```
Backend روی `http://localhost:8080` اجرا میشه.

### مرحله ۲ — VM ها رو بالا بیار
```bash
make vm-up
```
یا مستقیم:
```bash
cd deploy/vagrant
vagrant up
```

این کار:
- دو Ubuntu VM میسازه (`pz-srv-1` و `pz-srv-2`)
- SteamCMD نصب میکنه
- PZ Dedicated Server (app 380870) نصب میکنه
- `pz-agent` از آخرین GitHub release دانلود و نصب میکنه
- هر دو رو به عنوان `systemd` service راه می‌اندازه

### مرحله ۳ — وضعیت رو بررسی کن
```bash
make vm-status
```

---

## Network Map

```
Windows Host (192.168.56.1):
  ├── Docker: backend    → :8080
  ├── Docker: grafana    → :3000
  ├── Docker: prometheus → :9090
  │
  ├── VM pz-srv-1  (192.168.56.11)
  │     ├── PZ Server  → :16261 UDP (forwarded → host :16261)
  │     └── pz-agent   → pushes to http://192.168.56.1:8080
  │
  └── VM pz-srv-2  (192.168.56.12)
        ├── PZ Server  → :16261 UDP (forwarded → host :16263)
        └── pz-agent   → pushes to http://192.168.56.1:8080

Launcher (Windows native):
  └── connects to http://localhost:8080
```

---

## Commands

| Command | کار |
|---------|-----|
| `make vm-up` | VMs رو بالا میاره + provision میکنه |
| `make vm-down` | VMs رو خاموش میکنه (حذف نمیکنه) |
| `make vm-provision` | Ansible رو دوباره اجرا میکنه |
| `make vm-status` | وضعیت VM ها + backend |
| `vagrant ssh pz-srv-1` | SSH به VM 1 |
| `vagrant destroy -f` | همه VM ها رو حذف میکنه |

---

## نکته مهم: PZ Dedicated Server

PZ Dedicated Server (Steam App ID 380870) نیاز به **Steam account** داره:
```
steamcmd +login <username> +app_update 380870 +quit
```

Ansible playbook سعی میکنه anonymous install کنه — اگه fail شد:
1. `vagrant ssh pz-srv-1`
2. `sudo -u pzserver steamcmd +login <user> +app_update 380870 +quit`

Agent بدون PZ Server هم کار میکنه — اگه mods در `/home/pzserver/Zomboid/mods/` باشن، push میکنه.
