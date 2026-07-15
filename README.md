# mremoteng-go

Migración de [mRemoteNG](https://github.com/mRemoteNG/mRemoteNG) (C#/WinForms,
Windows-only) a Go, con el objetivo de soportar Linux nativamente y mantener
portabilidad a Windows.

Estado: esqueleto inicial, sin funcionalidad todavía. Ver
[`docs/MIGRATION_PLAN.md`](docs/MIGRATION_PLAN.md) para el plan de fases.

## Estructura

- `cmd/mremoteng` — punto de entrada de la aplicación.
- `internal/connection` — modelo de conexiones y árbol de contenedores.
- `internal/serialize/xml`, `internal/serialize/csv` — serializadores de
  archivos de conexión (compatibles con el formato del proyecto original).
- `internal/security` — cifrado y derivación de claves.
- `internal/protocol` — implementaciones de protocolo (SSH, RDP, VNC, etc.).
- `internal/ui` — interfaz gráfica (Fyne).

## Requisitos de desarrollo

- Go 1.23+ (no instalado aún en este entorno — instalar antes de continuar).
