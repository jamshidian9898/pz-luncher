# -*- mode: ruby -*-
# vi: set ft=ruby :

# PZ Launcher - Vagrant Development Environment
# Windows 11 with Go, Node.js, Wails, and build tools

Vagrant.configure("2") do |config|
  
  # ==================== VM CONFIGURATION ====================
  
  # Windows 11 Development Box (official Microsoft evaluation)
  # Alternative boxes if this is slow: "gusztavvargadr/windows-11" or "stefanscherer/windows_11"
  config.vm.box = "gusztavvargadr/windows-11-22h2-enterprise"
  config.vm.box_version = ">= 2202.0"
  
  # VM Settings
  config.vm.hostname = "pz-dev"
  config.vm.boot_timeout = 600  # 10 minutes for Windows boot
  
  # ==================== NETWORK ====================
  
  # Backend API port
  config.vm.network "forwarded_port", guest: 8080, host: 8080, host_ip: "127.0.0.1", auto_correct: true
  
  # Dev API port (if using dev-api mode)
  config.vm.network "forwarded_port", guest: 8765, host: 8765, host_ip: "127.0.0.1", auto_correct: true
  
  # Private network for internal communication
  config.vm.network "private_network", type: "dhcp"
  
  # ==================== SYNCED FOLDERS ====================
  
  # Project folder (your PZ launcher code)
  # This syncs your host code into VM automatically
  config.vm.synced_folder ".", "C:/Users/vagrant/project", 
    type: "virtualbox",
    automount: true,
    mount_options: ["iocharset=utf8"]
  
  # Optional: Cache folder for faster rebuilds
  config.vm.synced_folder "./.vagrant-cache", "C:/Users/vagrant/.cache",
    create: true,
    type: "virtualbox"
  
  # ==================== VIRTUALBOX PROVIDER ====================
  
  config.vm.provider "virtualbox" do |vb|
    vb.name = "PZ-Launcher-Dev"
    vb.memory = "6144"       # 6GB RAM (adjust based on your host)
    vb.cpus = 4              # 4 CPU cores
    vb.gui = true            # Show GUI window
    vb.linked_clone = true   # Faster cloning
    
    # Display settings
    vb.customize ["modifyvm", :id, "--vram", "256"]
    vb.customize ["modifyvm", :id, "--accelerate3d", "on"]
    vb.customize ["modifyvm", :id, "--clipboard-mode", "bidirectional"]
    vb.customize ["modifyvm", :id, "--draganddrop", "bidirectional"]
    vb.customize ["modifyvm", :id, "--audio", "none"]  # Disable audio if not needed
    
    # Performance
    vb.customize ["modifyvm", :id, "--nested-hw-virt", "off"]  # Nested virtualization off for stability
    vb.customize ["modifyvm", :id, "--pae", "on"]
    vb.customize ["modifyvm", :id, "--largepages", "on"]
    vb.customize ["modifyvm", :id, "--vtxux", "on"]
    vb.customize ["modifyvm", :id, "--vtxvpid", "on"]
    
    # Disable unused devices for speed
    vb.customize ["modifyvm", :id, "--usb", "off"]
    vb.customize ["modifyvm", :id, "--usbehci", "off"]
  end
  
  # ==================== PROVISIONING ====================
  
  # Step 1: Install Chocolatey (package manager)
  config.vm.provision "shell", privileged: true, name: "Install Chocolatey", inline: <<-SHELL
    Write-Host "Installing Chocolatey..." -ForegroundColor Green
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
    choco feature enable -n allowGlobalConfirmation
    refreshenv
    Write-Host "✅ Chocolatey installed" -ForegroundColor Green
  SHELL
  
  # Step 2: Install development tools
  config.vm.provision "shell", privileged: true, name: "Install Dev Tools", inline: <<-SHELL
    Write-Host "Installing development tools..." -ForegroundColor Cyan
    
    # Core tools
    choco install -y git
    choco install -y golang --version 1.22.0
    choco install -y nodejs-lts --version 20.11.0
    choco install -y make
    choco install -y mingw     # C compiler for Go CGO
    choco install -y 7zip
    choco install -y vscode    # Optional IDE
    
    # Wails dependencies (WebView2)
    choco install -y webview2-runtime
    
    refreshenv
    Write-Host "✅ Dev tools installed" -ForegroundColor Green
  SHELL
  
  # Step 3: Install Wails CLI and Go tools
  config.vm.provision "shell", privileged: false, name: "Install Wails", inline: <<-SHELL
    Write-Host "Installing Wails CLI..." -ForegroundColor Cyan
    
    # Ensure Go bin is in PATH
    $goBin = "$env:USERPROFILE\go\bin"
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $currentPath.Contains($goBin)) {
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$goBin", "User")
    }
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "User") + ";" + [Environment]::GetEnvironmentVariable("Path", "Machine")
    
    # Install Wails v2
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    
    # Verify installation
    wails version
    
    Write-Host "✅ Wails installed" -ForegroundColor Green
    Write-Host "Run 'wails doctor' to check dependencies" -ForegroundColor Yellow
  SHELL
  
  # Step 4: Initial project setup (runs once)
  config.vm.provision "shell", privileged: false, run: "once", name: "Project Setup", inline: <<-SHELL
    if (Test-Path "C:/Users/vagrant/project") {
      Write-Host "Setting up project..." -ForegroundColor Cyan
      cd C:/Users/vagrant/project
      
      # Install frontend dependencies
      if (Test-Path "apps/launcher-ui/frontend/package.json") {
        Write-Host "Installing npm dependencies..." -ForegroundColor Cyan
        cd apps/launcher-ui/frontend
        npm install
        cd ../../..
      }
      
      Write-Host "✅ Project setup complete" -ForegroundColor Green
      Write-Host "" 
      Write-Host "📁 Project location: C:\Users\vagrant\project" -ForegroundColor Yellow
      Write-Host "🚀 Ready to build! Run these commands:" -ForegroundColor Yellow
      Write-Host "   cd C:\Users\vagrant\project" -ForegroundColor White
      Write-Host "   make build-windows" -ForegroundColor White
      Write-Host "   OR" -ForegroundColor Gray
      Write-Host "   wails build -platform windows" -ForegroundColor White
    }
  SHELL
  
  # ==================== POST-UP MESSAGE ====================
  
  config.vm.post_up_message = <<-MSG
    ============================================
    🎮 PZ Launcher Development VM Ready!
    ============================================
    
    Access methods:
      - GUI:        VirtualBox window (auto-opens)
      - RDP:        vagrant rdp
      - PowerShell: vagrant powershell
    
    Project folder: C:\Users\vagrant\project
    (Auto-syncs with your host machine)
    
    Quick commands:
      vagrant halt       # Stop VM
      vagrant reload     # Restart VM
      vagrant suspend    # Pause VM
      vagrant destroy    # Delete VM
    
    Build commands:
      make help          # Show all commands
      make build-windows # Build launcher
    
    ============================================
  MSG
end
