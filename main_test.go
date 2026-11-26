package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Le asar généré doit être identique à celui que Vencord a écrit sur ce PC.
func TestBuildAsarMatchesReal(t *testing.T) {
	real := os.Getenv("REAL_ASAR")
	if real == "" {
		t.Skip("REAL_ASAR non défini")
	}
	want, err := os.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	got := buildAsar(filepath.Join(os.Getenv("APPDATA"), "Vencord", "dist", "patcher.js"))
	if !bytes.Equal(got, want) {
		t.Fatalf("asar différent :\n got  %q\n want %q", got, want)
	}
}

func TestInjectUninject(t *testing.T) {
	res := filepath.Join(t.TempDir(), "app-1.0.1", "resources")
	os.MkdirAll(res, 0o755)
	os.WriteFile(filepath.Join(res, "app.asar"), []byte("ORIGINAL"), 0o644)
	i := install{name: "Discord", resources: res}

	for range 2 { // deux fois : la ré-installation ne doit pas écraser _app.asar
		if err := inject(i, `C:\x\patcher.js`); err != nil {
			t.Fatal(err)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(res, "_app.asar")); string(b) != "ORIGINAL" {
		t.Fatalf("_app.asar = %q", b)
	}
	if err := uninject(i); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(res, "app.asar")); string(b) != "ORIGINAL" {
		t.Fatalf("app.asar après désinstallation = %q", b)
	}
	if exists(filepath.Join(res, "_app.asar")) {
		t.Fatal("_app.asar devrait avoir disparu")
	}
}

func TestApplySettingsMerge(t *testing.T) {
	file := filepath.Join(t.TempDir(), "settings", "settings.json")
	os.MkdirAll(filepath.Dir(file), 0o755)
	os.WriteFile(file, []byte("\xef\xbb\xbf"+`{"enabledThemes":["Ami.css"],"plugins":{"FakeNitro":{"enabled":false,"perso":1},"Autre":{"enabled":true}},"frameless":true}`), 0o644)

	if err := applySettings(file, "test"); err != nil {
		t.Fatal(err)
	}
	var s map[string]any
	data, _ := os.ReadFile(file)
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	plugins := s["plugins"].(map[string]any)
	fn := plugins["FakeNitro"].(map[string]any)
	if fn["enabled"] != true || fn["perso"] != float64(1) || fn["emojiSize"] != float64(48) {
		t.Errorf("FakeNitro mal fusionné : %v", fn)
	}
	if plugins["Autre"].(map[string]any)["enabled"] != true {
		t.Error("le plugin de l'ami a été perdu")
	}
	if pd, ok := plugins["PinDMs"].(map[string]any); !ok || pd["userBasedCategoryList"] != nil {
		t.Errorf("PinDMs : %v", plugins["PinDMs"])
	}
	if s["frameless"] != true || s["autoUpdate"] != false {
		t.Errorf("réglages globaux : frameless=%v autoUpdate=%v", s["frameless"], s["autoUpdate"])
	}
	themes := s["enabledThemes"].([]any)
	if len(themes) != 2 || themes[0] != "Ami.css" || themes[1] != "NotAnotherAnimeTheme.theme.css" {
		t.Errorf("thèmes : %v", themes)
	}
	if !exists(file + ".bak-test") {
		t.Error("pas de sauvegarde")
	}
}

func TestPayloadComplete(t *testing.T) {
	for _, f := range []string{"payload/dist/patcher.js", "payload/dist/preload.js", "payload/dist/renderer.js",
		"payload/dist/renderer.css", "payload/dist/package.json", "payload/themes/NotAnotherAnimeTheme.theme.css"} {
		if _, err := payload.ReadFile(f); err != nil {
			t.Errorf("%s manquant", f)
		}
	}
}

func TestFindInstalls(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	for _, v := range []string{"app-1.0.9270", "app-1.0.10012"} {
		res := filepath.Join(local, "Discord", v, "resources")
		os.MkdirAll(res, 0o755)
		os.WriteFile(filepath.Join(res, "app.asar"), nil, 0o644)
	}
	got := findInstalls()
	if len(got) != 1 || filepath.Base(filepath.Dir(got[0].resources)) != "app-1.0.10012" {
		t.Fatalf("findInstalls = %+v", got)
	}
}
