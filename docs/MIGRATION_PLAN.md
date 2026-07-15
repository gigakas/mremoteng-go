# Plan de migración: mRemoteNG (C#/WinForms) → Go + FreeRDP (proceso externo)

Origen: análisis y planificación realizados sobre el repositorio original en
`../mRemoteNG`. Este documento es la referencia de fases; no repetir su
contenido en otros documentos del repo, solo enlazar aquí.

## Principio arquitectónico no negociable

RDP y AnyDesk se integran **solo como procesos externos embebidos por
ventana**, nunca enlazando código (cgo contra `libfreerdp` o similar). mRemoteNG
original es GPLv2; FreeRDP es Apache-2.0 (incompatibles para *linking* directo
según la FSF). Para el embedding de ventanas:

- Linux: `github.com/BurntSushi/xgb` (protocolo X11 puro en Go, sin cgo, sin
  enlazar `libX11`).
- Windows: `golang.org/x/sys/windows` (`SetParent`/`SetWindowLong` vía
  syscall).

`xfreerdp` queda como dependencia de *runtime* (como hoy `PuTTYNG.exe` en el
proyecto original), nunca de *build*.

## Fase 0 — Spike de validación (2–3 semanas)

- Prototipo Fyne + reparent de ventana `xfreerdp` en X11.
- Mismo prototipo en Windows con `SetParent`.
- Validar comportamiento bajo Wayland (esperado: requiere XWayland).
- Criterio de salida: si el reparent no es confiable en GNOME/KDE modernos,
  decidir aquí si se acepta la limitación antes de seguir.

## Fase 1 — Núcleo de datos (paridad de modelo, sin UI ni protocolos)

Mapeo desde el C# original:

| C# original | Go destino |
|---|---|
| `Connection/AbstractConnectionRecord.cs`, `ConnectionInfo.cs` | `internal/connection` |
| `Connection/ConnectionInfoInheritance.cs` | Resolución de herencia sin reflection en runtime; usar `go:generate` o type switch explícito |
| `Container/ContainerInfo.cs` | `internal/connection` (árbol homogéneo) |
| `Config/Serializers/ConnectionSerializers/Xml/*` (v26/27/28) | `internal/serialize/xml` (mismo patrón versionado por `ConfVersion`) |
| `Config/Serializers/ConnectionSerializers/Csv/*` | `internal/serialize/csv` |
| `Security/SymmetricEncryption/AeadCryptographyProvider.cs` | `internal/security` (stdlib `crypto/cipher`, AES-GCM) |
| `Security/KeyDerivation/Pkcs5S2KeyGenerator.cs` | `internal/security` (`golang.org/x/crypto/pbkdf2`) |
| `LegacyRijndaelCryptographyProvider.cs` | `internal/security` (`crypto/aes` CBC, solo lectura legacy) |

**Prueba de aceptación crítica**: generar archivos `.xml` de conexiones con la
app C# original (distintas versiones de `ConfVersion`, cifrados y sin
cifrar) y verificar que el port en Go produce resultado idéntico al
desencriptar/leer. Bloqueante para avanzar a Fase 2.

## Fase 2 — Protocolos (orden de riesgo creciente)

1. SSH / Telnet / Rlogin / raw socket — nativo en Go (`golang.org/x/crypto/ssh`).
2. HTTP/HTTPS — webview nativo del SO (`github.com/webview/webview_go`).
3. VNC — completar sobre librería base existente (trabajo de relleno).
4. RDP — proceso externo `xfreerdp`/`wlfreerdp` + reparent. V1 sin
   redirección de discos/impresoras/portapapeles (backlog v2).
5. PowerShell remoting — WinRM vía librerías Go existentes.
6. AnyDesk — mismo patrón que RDP (protocolo propietario).

## Fase 3 — UI

- Toolkit: Fyne para árbol de conexiones, tabs, diálogos, menús.
- Docking de paneles: **v1 simplifica a layout fijo** (sin auto-hide/floating
  equivalente a `PanelBinder.cs`/`DockPanelLayoutLoader.cs` original);
  reevaluar como v2 según demanda.
- Theming: reimplementación manual de `Themes/` original como paletas Fyne.
- Credenciales externas (`ExternalConnectors/` original: AWS, 1Password,
  Vault/OpenBao, Passwordstate, Delinea) — clientes REST/CLI, portan bien.

## Fase 4 — Empaquetado

- Binario único Go por plataforma (`GOOS=linux/windows`, `GOARCH=amd64/arm64`).
- `xfreerdp` no se vendorea: dependencia de paquete en Linux
  (`.deb`/`.rpm`/Flatpak), binario oficial distribuido junto al `.zip`
  portable en Windows.
- Mantener estructura de canales Stable/Preview/Nightly del proyecto
  original durante la transición.

## Fase 5 — Migración y cutover

- Import directo de archivos de conexión existentes (cubierto por prueba de
  Fase 1).
- Opciones de registro de Windows (`Config/Settings/Registry/` original) no
  aplican en Linux; documentar equivalente en archivo de config.
- Convivencia en paralelo como canal "Preview" hasta paridad de RDP + SSH.
- Deprecar la versión C#/WinForms solo tras ciclo estable con feedback real.

## Riesgos abiertos

1. Wayland: sin soporte nativo garantizado, depende de XWayland.
2. Docking de paneles: pérdida de funcionalidad en v1 (decisión de producto).
3. Herencia por reflection: rediseño necesario, no traducción mecánica.
4. Alcance: reescritura completa: por eso vive en repo separado, no como
   rama del original.
