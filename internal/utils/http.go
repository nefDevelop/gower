package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// MaxJSONResponseBytes limita el tamaño de las respuestas JSON de las APIs.
	MaxJSONResponseBytes = 16 << 20 // 16 MiB
	// MaxImageBytes limita el tamaño de una imagen descargada.
	MaxImageBytes = 64 << 20 // 64 MiB
)

// newTransport crea un transporte con pooling y timeouts razonables. Sin
// MaxIdleConnsPerHost el transporte solo mantiene 2 conexiones idle por host,
// lo que provoca churn con la exploración multi-provider y los workers.
func newTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   8,
		MaxConnsPerHost:       16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}

var DefaultHTTPClient = &http.Client{
	Timeout:   30 * time.Second,
	Transport: newTransport(),
}

var ShortHTTPClient = &http.Client{
	Timeout:   5 * time.Second,
	Transport: newTransport(),
}

// DownloadHTTPClient permite descargas grandes con un timeout más holgado que
// el de las llamadas JSON.
var DownloadHTTPClient = &http.Client{
	Timeout:   5 * time.Minute,
	Transport: newTransport(),
}

// CloseResponse drena una parte pequeña del body antes de cerrarlo. Cerrar sin
// drenar impide reutilizar la conexión keep-alive.
func CloseResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
}

// ReadBodyLimit lee el body completo rechazándolo si supera limit bytes.
func ReadBodyLimit(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		limit = MaxJSONResponseBytes
	}
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes limit", limit)
	}
	return data, nil
}

// DecodeJSON decodifica JSON con un límite de tamaño para evitar que una API
// comprometida agote la memoria.
func DecodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(io.LimitReader(r, MaxJSONResponseBytes)).Decode(v)
}

// DoRequest ejecuta una petición GET con reintentos y backoff para 429 y 5xx.
// Respeta Retry-After cuando el servidor lo envía.
func DoRequest(ctx context.Context, client *http.Client, req *http.Request, attempts int) (*http.Response, error) {
	if client == nil {
		client = DefaultHTTPClient
	}
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := client.Do(req.Clone(ctx))
		if err != nil {
			if resp != nil {
				CloseResponse(resp)
			}
			lastErr = err
			if attempt == attempts-1 {
				break
			}
			if err := sleepContext(ctx, backoff(attempt)); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return resp, nil
		}

		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		// Se lee una pista del body para que el mensaje de error no sea opaco.
		body, _ := ReadBodyLimit(resp.Body, 512)
		CloseResponse(resp)
		lastErr = fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		if attempt == attempts-1 {
			break
		}
		wait := retryAfter
		if wait <= 0 {
			wait = backoff(attempt)
		}
		if err := sleepContext(ctx, wait); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func backoff(attempt int) time.Duration {
	return time.Duration(attempt+1) * 500 * time.Millisecond
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		return time.Until(t)
	}
	return 0
}

// RedactURL oculta credenciales de una URL antes de escribirla en los logs.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	changed := false
	for _, key := range []string{"api_key", "apikey", "key", "token", "access_token", "client_id", "client_secret"} {
		if q.Has(key) {
			q.Set(key, "REDACTED")
			changed = true
		}
	}
	if changed {
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// SafeFilename convierte un identificador externo en un nombre de fichero
// seguro. Evita path traversal (../), separadores y caracteres reservados en
// Windows sin cambiar los identificadores normales.
func SafeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)

	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 32 || r == 0x7f:
			b.WriteRune('_')
		case strings.ContainsRune(`:*?"<>|`, r):
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}

	result := strings.Trim(b.String(), " .")
	if result == "" || result == "." || result == ".." {
		return "wallpaper"
	}
	return result
}

// IsWithinDir indica si path se resuelve dentro de dir.
func IsWithinDir(path, dir string) bool {
	if dir == "" {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	// Resolver symlinks evita que un enlace dentro del directorio apunte fuera.
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
	}
	if resolved, err := filepath.EvalSymlinks(absDir); err == nil {
		absDir = resolved
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
