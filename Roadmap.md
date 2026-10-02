# Gower Roadmap

Hoja de ruta para mejorar Gower, organizada por fases e impacto.  
Cada fase puede entregarse de forma independiente.

---

## Fase 1: Calidad y Robustez (Prioridad Alta)

| # | Tarea | Archivos afectados | Esfuerzo |
|---|-------|--------------------|----------|
| 1.1 | HTTP Clients con timeouts en todos los providers | `internal/providers/wallhaven.go`, `reddit.go`, `nasa.go`, `bing.go`, `generic.go`, `internal/core/controller.go` | 1-2h |
| 1.2 | Eliminar `rand.Seed()` deprecated (Go 1.25+) | `cmd/root.go` | 5min |
| 1.3 | Error handling consistente: no ignorar errores silenciosamente | `internal/core/controller.go`, `cmd/daemon.go` | 2-3h |
| 1.4 | Añadir `context.Context` a métodos de Provider y Controller | `internal/providers/provider.go`, todos los providers, `internal/core/controller.go` | 3-4h |
| 1.5 | Implementar rate limiting real en providers | `internal/providers/` | 3-4h |
| 1.6 | Manejo de señales OS (SIGTERM/SIGINT) en el daemon | `cmd/daemon.go`, `main.go` | 2h |

## Fase 2: Refactor Arquitectónico (Prioridad Alta)

| # | Tarea | Archivos afectados | Esfuerzo |
|---|-------|--------------------|----------|
| ~~2.1~~ | ~~Dividir `controller.go` (2189 líneas) en servicios~~ **Hecho** en `1a63e38` | `internal/core/` | 8-12h |
| | - Extraer `FeedService` (feed CRUD, cache, sync) | | |
| | - Extraer `DownloadService` (download, validación) | | |
| | - Extraer `AnalysisService` (análisis de color, thumbnails) | | |
| | - Extraer `LocalIndexService` (indexación de directorios) | | |
| | - Controller como facade | | |
| 2.2 | Refactor `wallpaper_changer.go` (679 líneas): eliminar switch duplicado | `internal/core/wallpaper_changer.go` | 4-6h |
| | - Extraer command builder por DE (strategy pattern) | | |
| | - Unificar lógica clone/distinct | | |
| 2.3 | Reemplazar señales de daemon por archivos planos con Unix signals | `cmd/daemon.go` | 4h |

## Fase 3: Testing y Mantenibilidad (Prioridad Media)

| # | Tarea | Archivos afectados | Esfuerzo |
|---|-------|--------------------|----------|
| 3.1 | Migrar de `utils.Log` global a dependency injection | `internal/utils/logger.go`, `internal/core/`, `cmd/`, `internal/providers/` | 3-4h |
| 3.2 | Tests para controller (cubrir sync, analyze, indexación) | `internal/core/controller_test.go` | 6-8h |
| ~~3.3~~ | ~~Tests para wallpaper_changer (mock DE commands)~~ **Hecho** en `e4d48e7` | `internal/core/wallpaper_changer_test.go` | 3-4h |
| 3.4 | CI/CD: GitHub Actions con lint + test | `.github/workflows/ci.yml` | 2h |
| 3.5 | Configurar `golangci-lint` formalmente | `.golangci.yml` | 1h |

## Fase 4: Features y Limpieza (Prioridad Media-Baja)

| # | Tarea | Archivos afectados | Esfuerzo |
|---|-------|--------------------|----------|
| 4.1 | Config vía reflection → code generation tipado | `cmd/config.go`, `pkg/models/config.go` + `generate.go` | 4-6h |
| 4.2 | Eliminar código muerto (`pkg/storage/`) | `pkg/storage/` | 30min |
| 4.3 | Worker pool size configurable (no hardcoded 5) | `internal/core/controller.go`, `pkg/models/config.go` | 1h |
| 4.4 | Estandarizar idioma a inglés (comentarios, errores) | Todo el código | 3-4h |
| 4.5 | README bilingüe (español + inglés) | `README.md` | 2h |

## Fase 5: Visión (Futuro)

| # | Tarea | Descripción | Esfuerzo |
|---|-------|-------------|----------|
| 5.1 | **Más providers** | Pexels, local filesystem watcher | 3-4h c/u |
| 5.2 | **Sync cloud** | Sincronizar config + favoritos entre máquinas | 10-15h |
| 5.3 | **Plugins** | Providers custom sin recompilar (Go plugin o WASM) | 15-20h |
| 5.4 | **Colecciones** | Agrupar wallpapers en colecciones temáticas | 6-8h |

**Completados en esta sesión:**
- ✅ **Unsplash provider** (`internal/providers/unsplash.go`)
- ✅ **Watch mode** (`gower feed watch` — polling)
- ✅ **Estadísticas** (`gower stats` — dashboard by provider/theme/resolution)
- ✅ **Integración systemd** (`gower daemon install/uninstall`)

---

## Timeline sugerido

```
Semana 1-2:  Fase 1 (calidad)
Semana 3-4:  Fase 2 (refactor arquitectónico)
Semana 5-6:  Fase 3 (testing, DI, CI)
Semana 7-8:  Fase 4 (features + limpieza)
Semana 9+:   Fase 5 (visión)
```

---

## Métricas de éxito (estado actual)

- `controller.go` ~1870 líneas (de 2210 original)
- `wallpaper_changer.go` ~540 líneas (de 679 original)
- 3 servicios extraídos: `DownloadService`, `FeedService`, `AnalysisService`
- `go vet ./...` y `golangci-lint` pasan sin warnings
- Daemon responde a SIGTERM/SIGINT + señales por archivos
- Rate limiting funcional en todos los providers
- Config type-safe: 0 reflection, 41 paths generados con `go generate`
- `pkg/storage/` eliminado (código muerto)
- README bilingüe español/inglés
