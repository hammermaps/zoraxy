# Zoraxy - Schnellstart Anleitung

Willkommen bei Zoraxy! Diese Schnellstart-Anleitung hilft Ihnen, Zoraxy in wenigen Minuten einzurichten und zu verwenden.

## Inhaltsverzeichnis

- [Was ist Zoraxy?](#was-ist-zoraxy)
- [Systemanforderungen](#systemanforderungen)
- [Installation](#installation)
  - [Linux](#linux)
  - [Windows](#windows)
  - [Docker](#docker)
  - [Aus Quellcode bauen](#aus-quellcode-bauen)
- [Erste Schritte](#erste-schritte)
- [Grundkonfiguration](#grundkonfiguration)
- [Häufige Anwendungsfälle](#häufige-anwendungsfälle)
- [Fehlerbehebung](#fehlerbehebung)

## Was ist Zoraxy?

Zoraxy ist ein universeller HTTP-Reverse-Proxy und Weiterleitungstool, geschrieben in Go. Es bietet:

- 🚀 Einfache Web-Oberfläche mit detaillierten Anleitungen
- 🔄 Reverse Proxy (HTTP/2) mit WebSocket-Unterstützung
- 🔒 TLS/SSL-Setup mit automatischer Zertifikatserneuerung (ACME/Let's Encrypt)
- ⚖️ Load Balancing und Response Caching
- 🌍 Länder- und IP-basierte Zugriffskontrolle
- 📊 Integrierte Uptime-Überwachung
- 🔌 Plugin-System für Erweiterungen
- 🐳 Docker-Unterstützung

## Systemanforderungen

### Minimale Anforderungen
- **Betriebssystem**: Linux, Windows, macOS, FreeBSD
- **RAM**: 512 MB (1 GB empfohlen)
- **CPU**: 1 Core (2 Cores empfohlen)
- **Festplatte**: 100 MB für die Anwendung

### Empfohlene Anforderungen
- **RAM**: 2 GB oder mehr
- **CPU**: 2 Cores oder mehr
- **Festplatte**: 1 GB für Logs und Konfigurationen

## Installation

### Linux

#### Schritt 1: Binary herunterladen

```bash
# Für AMD64
wget https://github.com/tobychui/zoraxy/releases/latest/download/zoraxy_linux_amd64

# Für ARM64 (z.B. Raspberry Pi 4)
wget https://github.com/tobychui/zoraxy/releases/latest/download/zoraxy_linux_arm64

# Für ARM (ältere Raspberry Pi Modelle)
wget https://github.com/tobychui/zoraxy/releases/latest/download/zoraxy_linux_arm
```

#### Schritt 2: Ausführbar machen

```bash
chmod +x zoraxy_linux_amd64
```

#### Schritt 3: Starten

```bash
sudo ./zoraxy_linux_amd64 -port=:8000
```

Die Web-Oberfläche ist nun unter `http://localhost:8000` erreichbar.

#### Als Systemdienst einrichten (optional)

Erstellen Sie eine systemd-Servicedatei:

```bash
sudo nano /etc/systemd/system/zoraxy.service
```

Fügen Sie folgenden Inhalt ein:

```ini
[Unit]
Description=Zoraxy Reverse Proxy
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/zoraxy
ExecStart=/opt/zoraxy/zoraxy_linux_amd64 -port=:8000
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Service aktivieren und starten:

```bash
sudo systemctl daemon-reload
sudo systemctl enable zoraxy
sudo systemctl start zoraxy
sudo systemctl status zoraxy
```

### Windows

#### Schritt 1: Binary herunterladen

Laden Sie die neueste Windows-Version herunter:
[zoraxy_windows_amd64.exe](https://github.com/tobychui/zoraxy/releases/latest/download/zoraxy_windows_amd64.exe)

#### Schritt 2: Starten

Doppelklicken Sie auf die heruntergeladene `.exe`-Datei. Ein Konsolen-Fenster öffnet sich und Zoraxy startet automatisch.

Die Web-Oberfläche ist unter `http://localhost:8000` erreichbar.

#### Als Windows-Dienst einrichten (optional)

Verwenden Sie NSSM (Non-Sucking Service Manager):

1. NSSM herunterladen von [nssm.cc](https://nssm.cc/download)
2. Dienst installieren:

```cmd
nssm install Zoraxy "C:\path\to\zoraxy_windows_amd64.exe" -port=:8000
nssm start Zoraxy
```

### Docker

#### Schritt 1: Docker Compose verwenden

Erstellen Sie eine `docker-compose.yml`:

```yaml
version: '3.8'

services:
  zoraxy:
    image: zoraxydocker/zoraxy:latest
    container_name: zoraxy
    restart: unless-stopped
    ports:
      - "8000:8000"  # Management-Interface
      - "80:80"      # HTTP
      - "443:443"    # HTTPS
    volumes:
      - ./zoraxy-data:/opt/zoraxy/config
      - ./zoraxy-certs:/opt/zoraxy/certs
    environment:
      - TZ=Europe/Berlin
```

#### Schritt 2: Starten

```bash
docker-compose up -d
```

Die Web-Oberfläche ist unter `http://localhost:8000` erreichbar.

### Aus Quellcode bauen

#### Voraussetzungen
- Go 1.23 oder höher

#### Build-Schritte

```bash
# Repository klonen
git clone https://github.com/tobychui/zoraxy
cd zoraxy/src/

# Abhängigkeiten herunterladen
go mod tidy

# Bauen
go build

# Starten
sudo ./zoraxy -port=:8000
```

## Erste Schritte

### 1. Erste Anmeldung

1. Öffnen Sie Ihren Browser und navigieren Sie zu `http://localhost:8000`
2. Bei der ersten Anmeldung werden Sie aufgefordert, ein Administrator-Passwort zu erstellen
3. Geben Sie ein sicheres Passwort ein und merken Sie es sich gut

### 2. Übersicht der Web-Oberfläche

Nach der Anmeldung sehen Sie das Dashboard mit folgenden Hauptbereichen:

- **Dashboard**: Übersicht über Ihre Proxy-Konfiguration
- **Proxy Rules**: Reverse-Proxy-Regeln erstellen und verwalten
- **TLS/SSL**: SSL-Zertifikate verwalten
- **Access Control**: Zugriffskontrolle konfigurieren
- **Settings**: Systemeinstellungen anpassen

### 3. Erste Proxy-Regel erstellen

1. Klicken Sie auf **"Proxy Rules"** im Menü
2. Klicken Sie auf **"Add New Rule"**
3. Füllen Sie folgende Felder aus:
   - **Domain**: Ihre externe Domain (z.B. `example.com`)
   - **Target**: Internes Ziel (z.B. `http://192.168.1.100:8080`)
4. Klicken Sie auf **"Save"**

Ihre erste Proxy-Regel ist nun aktiv!

## Grundkonfiguration

### Port-Konfiguration

Standardmäßig läuft:
- Management-Interface auf Port 8000
- HTTP-Proxy auf Port 80
- HTTPS-Proxy auf Port 443

Um Ports zu ändern:

```bash
./zoraxy -port=:9000 -default_inbound_port=8080
```

### TLS/SSL-Zertifikate einrichten

#### Let's Encrypt (Automatisch)

1. Gehen Sie zu **"TLS/SSL"** → **"ACME"**
2. Geben Sie Ihre E-Mail-Adresse ein
3. Wählen Sie Ihre Domain aus
4. Klicken Sie auf **"Request Certificate"**

Das Zertifikat wird automatisch angefordert und erneuert!

#### Eigenes Zertifikat hochladen

1. Gehen Sie zu **"TLS/SSL"** → **"Certificates"**
2. Klicken Sie auf **"Upload Certificate"**
3. Laden Sie Ihr Zertifikat (`.crt`) und Schlüssel (`.key`) hoch

### Zugriffskontrolle konfigurieren

#### IP-basierte Filterung

1. Gehen Sie zu **"Access Control"** → **"IP Filter"**
2. Fügen Sie IPs zur Blacklist oder Whitelist hinzu
3. Unterstützte Formate:
   - Einzelne IP: `192.168.1.100`
   - CIDR: `192.168.1.0/24`
   - Wildcard: `192.168.1.*`

#### Länderbasierte Filterung

1. Gehen Sie zu **"Access Control"** → **"GeoIP"**
2. Wählen Sie Länder aus, die blockiert oder erlaubt werden sollen
3. Aktivieren Sie die Filterung

## Häufige Anwendungsfälle

### 1. Einfacher Reverse Proxy

**Szenario**: Einen internen Webserver nach außen verfügbar machen

```
Externe Domain: blog.example.com
Interner Server: http://192.168.1.50:3000
```

**Konfiguration**:
1. Proxy Rule erstellen: `blog.example.com` → `http://192.168.1.50:3000`
2. Optional: SSL-Zertifikat für `blog.example.com` einrichten

### 2. Load Balancing

**Szenario**: Mehrere Backend-Server für eine Anwendung

```
Domain: app.example.com
Backends:
  - http://192.168.1.10:8080
  - http://192.168.1.11:8080
  - http://192.168.1.12:8080
```

**Konfiguration**:
1. Proxy Rule für `app.example.com` erstellen
2. Mehrere Upstream-Server hinzufügen
3. Load-Balancing-Methode wählen (Round Robin, Least Connection, etc.)

### 3. Subdomain-Routing

**Szenario**: Verschiedene Dienste auf Subdomains

```
blog.example.com → http://192.168.1.10:3000
shop.example.com → http://192.168.1.11:8080
api.example.com → http://192.168.1.12:5000
```

**Konfiguration**:
Erstellen Sie für jede Subdomain eine separate Proxy Rule.

### 4. WebSocket-Proxy

**Szenario**: WebSocket-Anwendung proxyen

WebSocket-Unterstützung ist automatisch aktiviert! Erstellen Sie einfach eine normale Proxy-Regel.

### 5. Statische Website hosten

**Szenario**: HTML/CSS/JS-Dateien direkt bereitstellen

1. Gehen Sie zu **"Static Web Server"**
2. Erstellen Sie ein neues virtuelles Verzeichnis
3. Laden Sie Ihre Dateien hoch oder geben Sie einen lokalen Pfad an

## Fehlerbehebung

### Problem: Zoraxy startet nicht

**Lösung**:
- Prüfen Sie, ob Port 8000 bereits belegt ist: `netstat -tuln | grep 8000`
- Verwenden Sie einen anderen Port: `./zoraxy -port=:9000`
- Prüfen Sie Berechtigungen (Ports < 1024 benötigen root)

### Problem: SSL-Zertifikat kann nicht angefordert werden

**Mögliche Ursachen**:
- Domain zeigt nicht auf Ihren Server
- Port 80 ist nicht erreichbar
- Firewall blockiert eingehende Verbindungen

**Lösung**:
- DNS-Einträge überprüfen: `nslookup ihre-domain.com`
- Port 80 öffnen: `sudo ufw allow 80`
- Logs prüfen in der Zoraxy-Oberfläche

### Problem: Proxy funktioniert nicht

**Checkliste**:
- [ ] Ist die Proxy-Regel aktiviert?
- [ ] Ist das Backend erreichbar? `curl http://backend-ip:port`
- [ ] Sind DNS-Einträge korrekt?
- [ ] Zeigen Logs Fehler an?

### Problem: Hohe CPU-Auslastung

**Lösungen**:
- Response Caching aktivieren
- Fast GeoIP deaktivieren auf Low-End-Geräten: Starten ohne `-fastgeoip`
- Rate Limiting aktivieren

### Logs überprüfen

```bash
# Logs anzeigen
cd /pfad/zu/zoraxy
cat log/zoraxy.log

# Live-Logs verfolgen
tail -f log/zoraxy.log
```

## Nächste Schritte

Nach dieser Schnellstart-Anleitung können Sie:

1. **Erweiterte Features erkunden**:
   - Response Caching einrichten
   - Custom Headers konfigurieren
   - Redirect Rules erstellen
   - Stream Proxy (TCP/UDP) nutzen

2. **Dokumentation lesen**:
   - [Offizielle Wiki](https://github.com/tobychui/zoraxy/wiki)
   - [FAQ](https://github.com/tobychui/zoraxy/wiki/FAQ---Frequently-Asked-Questions)

3. **Community beitreten**:
   - GitHub Issues für Support
   - Reddit Community für Austausch

## Sicherheitshinweise

⚠️ **Wichtige Sicherheitstipps**:

- Verwenden Sie ein starkes Admin-Passwort
- Halten Sie Zoraxy immer auf dem neuesten Stand
- Aktivieren Sie HTTPS für das Management-Interface
- Verwenden Sie `-noauth` nur in vertrauenswürdigen Netzwerken
- Sichern Sie regelmäßig Ihre Konfiguration

## Support und Hilfe

Bei Fragen oder Problemen:

- 📖 [Dokumentation](https://github.com/tobychui/zoraxy/wiki)
- 🐛 [Issues auf GitHub](https://github.com/tobychui/zoraxy/issues)
- 💬 [Diskussionen](https://github.com/tobychui/zoraxy/discussions)

## Lizenz

Zoraxy ist Open Source unter der AGPL-Lizenz. Weitere Informationen finden Sie in der [LICENSE](LICENSE)-Datei.

---

**Viel Erfolg mit Zoraxy! 🚀**
