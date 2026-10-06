package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gower/internal/core"
	"gower/pkg/models"
)

// TestContractJSON fija el shape público de la salida --json de gower que
// consume gowerGUI.
//
// El contrato vive en ../contract/gower-contract.json y lo versiona gower: es
// gowerGUI quien lo descarga (del tag que tenga pineado) y verifica su lado.
// Aquí solo se vigila la parte que gower controla: que los tags `json` de los
// tipos serializados no cambien a espaldas del contrato. Un renombrado o un
// campo obligatorio que gane `omitempty` rompe este test.
//
// Alcance deliberadamente mínimo: solo los campos que gowerGUI lee de verdad.
type jsonContract struct {
	ContractVersion int `json:"contract_version"`
	WallpaperItem   struct {
		Required []string `json:"required"`
		Optional []string `json:"optional"`
	} `json:"wallpaper_item"`
	MonitorItem struct {
		Required []string `json:"required"`
	} `json:"monitor_item"`
	Status struct {
		Required []string `json:"required"`
		Paths    []string `json:"paths"`
	} `json:"status"`
}

func loadJSONContract(t *testing.T) jsonContract {
	t.Helper()

	path := filepath.Join("..", "contract", "gower-contract.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no se pudo leer el contrato %s: %v", path, err)
	}

	var c jsonContract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("contrato inválido %s: %v", path, err)
	}
	return c
}

// objectKeys serializa v y devuelve el conjunto de claves del objeto JSON.
func objectKeys(t *testing.T, v any) map[string]bool {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal a objeto: %v", err)
	}

	keys := make(map[string]bool, len(m))
	for k := range m {
		keys[k] = true
	}
	return keys
}

// TestContractWallpaperItem: las claves obligatorias se emiten incluso con el
// valor cero (no pueden llevar omitempty) y las opcionales aparecen cuando el
// campo está poblado.
func TestContractWallpaperItem(t *testing.T) {
	c := loadJSONContract(t)

	zero := objectKeys(t, models.Wallpaper{})
	for _, k := range c.WallpaperItem.Required {
		if !zero[k] {
			t.Errorf("clave obligatoria %q ausente en un Wallpaper vacío (¿se añadió omitempty o se renombró el tag?)", k)
		}
	}

	populated := objectKeys(t, models.Wallpaper{
		ID:        "id",
		URL:       "url",
		Thumbnail: "thumb",
		Path:      "/tmp/x",
		Source:    "src",
		Extension: "jpg",
		Dimension: "1920x1080",
		Ratio:     "16:9",
		Theme:     "dark",
		Color:     "#000000",
		Permalink: "https://example.com",
		Title:     "title",
		Seen:      true,
		Added:     1,
	})
	for _, k := range append(append([]string{}, c.WallpaperItem.Required...), c.WallpaperItem.Optional...) {
		if !populated[k] {
			t.Errorf("clave %q declarada en el contrato y ausente al poblar el campo", k)
		}
	}
}

// TestContractMonitorItem: core.Monitor no tiene tags json, así que las claves
// van en mayúscula. Si alguien las minúsculiza, la GUI deja de ver los monitores.
func TestContractMonitorItem(t *testing.T) {
	c := loadJSONContract(t)

	keys := objectKeys(t, core.Monitor{ID: "HDMI-1", Name: "HDMI-1", Primary: true})
	for _, k := range c.MonitorItem.Required {
		if !keys[k] {
			t.Errorf("clave de monitor %q ausente (¿se renombraron los campos o se añadieron tags json?)", k)
		}
	}
}

// TestContractStatus: las claves de nivel superior y las rutas anidadas que
// consume gowerGUI deben seguir emitiéndose.
func TestContractStatus(t *testing.T) {
	c := loadJSONContract(t)

	out := StatusOutput{
		Daemon:    &DaemonStatus{Running: true, PID: 4242},
		Providers: &ProvidersStatus{},
		Wallpaper: &CurrentWallpaperStatus{Wallpapers: []models.Wallpaper{{ID: "x"}}},
	}

	keys := objectKeys(t, out)
	for _, k := range c.Status.Required {
		if !keys[k] {
			t.Errorf("clave de status %q ausente", k)
		}
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}

	for _, path := range c.Status.Paths {
		assertJSONPath(t, m, path)
	}
}

// assertJSONPath comprueba que una ruta con puntos ("a.b") existe en el JSON.
func assertJSONPath(t *testing.T, m map[string]any, path string) {
	t.Helper()

	cur := any(m)
	for _, part := range strings.Split(path, ".") {
		obj, ok := cur.(map[string]any)
		if !ok {
			t.Errorf("ruta %q: %q no es un objeto", path, part)
			return
		}
		next, ok := obj[part]
		if !ok {
			t.Errorf("ruta %q: falta la clave %q", path, part)
			return
		}
		cur = next
	}
}
