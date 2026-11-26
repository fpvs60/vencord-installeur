// Installeur Vencord pour les potes : installe le build Vencord, les réglages
// des plugins et le thème embarqués dans l'exe, puis injecte Vencord dans Discord.
package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

//go:embed all:payload
var payload embed.FS

var discordFlavors = []string{"Discord", "DiscordPTB", "DiscordCanary"}

type install struct {
	name      string // Discord, DiscordPTB, DiscordCanary
	root      string // %LOCALAPPDATA%\Discord
	resources string // ...\app-x.y.z\resources
}

func main() {
	setUTF8Console()
	in := bufio.NewReader(os.Stdin)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n[ERREUR] %v\n", r)
		}
		fmt.Print("\nAppuie sur Entrée pour fermer...")
		in.ReadString('\n')
	}()

	fmt.Println("==============================================")
	fmt.Println("   Installeur Vencord : plugins + thème")
	fmt.Println("==============================================")
	fmt.Println()

	installs := findInstalls()
	if len(installs) == 0 {
		panic("aucun Discord trouvé dans " + os.Getenv("LOCALAPPDATA") + ". Installe Discord d'abord.")
	}
	for _, i := range installs {
		fmt.Printf("Discord trouvé : %s (%s)\n", i.name, filepath.Base(filepath.Dir(i.resources)))
	}

	fmt.Println()
	fmt.Println("  1. Installer / mettre à jour")
	fmt.Println("  2. Désinstaller Vencord")
	fmt.Print("\nChoix [1] : ")
	choice, _ := in.ReadString('\n')
	choice = strings.TrimSpace(choice)

	fmt.Println("\nFermeture de Discord...")
	killDiscord()

	switch choice {
	case "2":
		for _, i := range installs {
			must(uninject(i))
			fmt.Printf("  Vencord retiré de %s\n", i.name)
		}
		fmt.Println("\nC'est fait. Tes réglages Vencord restent dans %APPDATA%\\Vencord.")
	default:
		doInstall(installs)
	}

	relaunch(installs[0])
}

func doInstall(installs []install) {
	vencordDir := filepath.Join(os.Getenv("APPDATA"), "Vencord")
	stamp := time.Now().Format("20060102-150405")

	fmt.Println("Installation de Vencord...")
	distDir := filepath.Join(vencordDir, "dist")
	if exists(distDir) {
		must(os.Rename(distDir, distDir+".bak-"+stamp))
	}
	must(extractDir("payload/dist", distDir))

	fmt.Println("Installation du thème...")
	must(extractDir("payload/themes", filepath.Join(vencordDir, "themes")))

	fmt.Println("Application des réglages des plugins...")
	must(applySettings(filepath.Join(vencordDir, "settings", "settings.json"), stamp))

	patcher := filepath.Join(distDir, "patcher.js")
	for _, i := range installs {
		must(inject(i, patcher))
		fmt.Printf("  Vencord injecté dans %s\n", i.name)
	}

	fmt.Println("\nC'est fait ! Discord va se relancer avec Vencord, les plugins et le thème.")
}

// --- Discord ---

func findInstalls() []install {
	var out []install
	local := os.Getenv("LOCALAPPDATA")
	for _, name := range discordFlavors {
		root := filepath.Join(local, name)
		apps, _ := filepath.Glob(filepath.Join(root, "app-*"))
		sort.Slice(apps, func(a, b int) bool { return versionLess(apps[a], apps[b]) })
		for k := len(apps) - 1; k >= 0; k-- {
			res := filepath.Join(apps[k], "resources")
			if exists(filepath.Join(res, "app.asar")) || exists(filepath.Join(res, "_app.asar")) {
				out = append(out, install{name: name, root: root, resources: res})
				break
			}
		}
	}
	return out
}

// versionLess compare app-1.0.9270 et app-1.0.10012 numériquement.
func versionLess(a, b string) bool {
	pa := strings.Split(strings.TrimPrefix(filepath.Base(a), "app-"), ".")
	pb := strings.Split(strings.TrimPrefix(filepath.Base(b), "app-"), ".")
	for k := 0; k < len(pa) && k < len(pb); k++ {
		var x, y int
		fmt.Sscan(pa[k], &x)
		fmt.Sscan(pb[k], &y)
		if x != y {
			return x < y
		}
	}
	return len(pa) < len(pb)
}

func killDiscord() {
	for _, name := range discordFlavors {
		exec.Command("taskkill", "/F", "/IM", name+".exe").Run()
	}
	time.Sleep(2 * time.Second)
}

func relaunch(i install) {
	update := filepath.Join(i.root, "Update.exe")
	if exists(update) {
		exec.Command(update, "--processStart", i.name+".exe").Start()
	}
}

// inject remplace app.asar par un petit asar qui charge Vencord, et garde
// l'original sous _app.asar (c'est là que Vencord va le chercher).
func inject(i install, patcher string) error {
	asar := filepath.Join(i.resources, "app.asar")
	orig := filepath.Join(i.resources, "_app.asar")
	if !exists(orig) {
		if err := os.Rename(asar, orig); err != nil {
			return fmt.Errorf("impossible de renommer app.asar (Discord est encore ouvert ?) : %w", err)
		}
	}
	return os.WriteFile(asar, buildAsar(patcher), 0o644)
}

func uninject(i install) error {
	asar := filepath.Join(i.resources, "app.asar")
	orig := filepath.Join(i.resources, "_app.asar")
	if !exists(orig) {
		return nil // pas injecté
	}
	if err := os.Remove(asar); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return os.Rename(orig, asar)
}

// buildAsar écrit une archive asar avec index.js (require du patcher) et package.json.
func buildAsar(patcher string) []byte {
	quoted, _ := json.Marshal(patcher)
	index := []byte("require(" + string(quoted) + ")")
	pkg := []byte("{\n\t\"name\": \"discord\",\n\t\"main\": \"index.js\"\n}")
	header := fmt.Sprintf(`{"files":{"index.js":{"size":%d,"offset":"0"},"package.json":{"size":%d,"offset":"%d"}}}`,
		len(index), len(pkg), len(index))

	strLen := len(header)
	aligned := (strLen + 3) &^ 3
	payloadSize := 4 + aligned
	pickleSize := 4 + payloadSize

	var b bytes.Buffer
	for _, v := range []int{4, pickleSize, payloadSize, strLen} {
		binary.Write(&b, binary.LittleEndian, uint32(v))
	}
	b.WriteString(header)
	b.Write(make([]byte, aligned-strLen))
	b.Write(index)
	b.Write(pkg)
	return b.Bytes()
}

// --- Réglages ---

// applySettings fusionne preset.json dans settings.json : les plugins du preset
// écrasent leurs réglages, les autres réglages de la personne restent.
func applySettings(file, stamp string) error {
	raw, err := payload.ReadFile("payload/preset.json")
	if err != nil {
		return err
	}
	var preset map[string]any
	if err := json.Unmarshal(raw, &preset); err != nil {
		return fmt.Errorf("preset.json : %w", err)
	}

	current := map[string]any{}
	if data, err := os.ReadFile(file); err == nil {
		os.WriteFile(file+".bak-"+stamp, data, 0o644)
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		if err := json.Unmarshal(data, &current); err != nil {
			current = map[string]any{} // fichier cassé : on repart de zéro (sauvegarde gardée)
		}
	}

	for key, val := range preset {
		switch key {
		case "plugins":
			plugins, _ := current["plugins"].(map[string]any)
			if plugins == nil {
				plugins = map[string]any{}
			}
			for name, p := range val.(map[string]any) {
				existing, _ := plugins[name].(map[string]any)
				if existing == nil {
					existing = map[string]any{}
				}
				for k, v := range p.(map[string]any) {
					existing[k] = v
				}
				plugins[name] = existing
			}
			current["plugins"] = plugins
		case "enabledThemes":
			current["enabledThemes"] = union(current["enabledThemes"], val)
		default:
			current[key] = val
		}
	}

	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(current, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, out, 0o644)
}

func union(a, b any) []any {
	seen := map[string]bool{}
	var out []any
	for _, list := range []any{a, b} {
		items, _ := list.([]any)
		for _, it := range items {
			if s, ok := it.(string); ok && !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}

// --- Utilitaires ---

func extractDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := payload.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := payload.ReadFile(path.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func setUTF8Console() {
	k := syscall.NewLazyDLL("kernel32.dll")
	k.NewProc("SetConsoleOutputCP").Call(65001)
	k.NewProc("SetConsoleCP").Call(65001)
}
