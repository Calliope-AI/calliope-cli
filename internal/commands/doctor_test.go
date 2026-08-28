package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calliope/calliope-cli/internal/appctx"
	"github.com/calliope/calliope-cli/internal/auth"
)

func checksOf(t *testing.T, salida []byte) map[string]string {
	t.Helper()
	var env struct {
		Data []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(salida, &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, salida)
	}
	m := map[string]string{}
	for _, c := range env.Data {
		m[c.Name] = c.Status
	}
	return m
}

// checkDetail devuelve el Detail del chequeo `nombre` en la salida JSON, o
// falla el test si no aparece. Complementa a checksOf cuando lo que importa
// no es solo el estado sino el texto exacto del detalle (p. ej. que nombre un
// fichero, o que no filtre un token).
func checkDetail(t *testing.T, salida []byte, nombre string) string {
	t.Helper()
	var env struct {
		Data []struct {
			Name   string `json:"name"`
			Detail string `json:"detail"`
		} `json:"data"`
	}
	if err := json.Unmarshal(salida, &env); err != nil {
		t.Fatalf("salida no es JSON: %v (%q)", err, salida)
	}
	for _, c := range env.Data {
		if c.Name == nombre {
			return c.Detail
		}
	}
	t.Fatalf("no se encontró el chequeo %q en la salida: %s", nombre, salida)
	return ""
}

// writeGlobalConfig escribe contenido literal en el config.json global que
// appctx.BuildWithoutCredential (vía config.Load) intenta leer, siguiendo la
// misma resolución de ruta que config.globalDir: HOME/.config/calliope (sin
// XDG_CONFIG_HOME, que depsWithServer no fija).
func writeGlobalConfig(t *testing.T, d appctx.Deps, contenido string) {
	t.Helper()
	dir := filepath.Join(d.Env("HOME"), ".config", "calliope")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDoctorTodoCorrecto(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"u-1","email":"a@b.c"}`))
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	c := checksOf(t, stdout.Bytes())
	for _, nombre := range []string{"credencial", "organización", "conectividad"} {
		if c[nombre] != "ok" {
			t.Errorf("chequeo %q = %q, se esperaba ok", nombre, c[nombre])
		}
	}
}

func TestDoctorSinCredencialInformaEnVezDeFallar(t *testing.T) {
	d, stdout, _ := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	d.Store = auth.NewFileStore(filepath.Join(t.TempDir(), "vacio.json"))

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor nunca debe fallar, debe informar: %v", err)
	}

	c := checksOf(t, stdout.Bytes())
	if c["credencial"] != "error" {
		t.Errorf("chequeo credencial = %q, se esperaba error", c["credencial"])
	}
}

func TestDoctorConBackendCaidoLoDetecta(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	c := checksOf(t, stdout.Bytes())
	if c["conectividad"] != "error" {
		t.Errorf("chequeo conectividad = %q, se esperaba error", c["conectividad"])
	}
}

// TestDoctorNuncaImprimeElToken cubre el aviso más delicado de esta tarea: el
// token de la credencial no debe aparecer en la salida de `doctor` en ningún
// modo, ni siquiera cuando la conectividad falla y el error del backend
// podría, por descuido, arrastrar detalles de la petición. Se comprueban los
// dos modos de salida (JSON, el único que depsWithServer ejercita sin TTY, y
// el renderer de texto con IsTTY:true) para que un futuro cambio en
// checkConnectivity que interpole ctx.Cred en vez del cred resuelto no pase
// inadvertido.
func TestDoctorNuncaImprimeElToken(t *testing.T) {
	const tokenSecreto = "cal_live_no_debe_salir_nunca"

	casos := []struct {
		nombre  string
		handler http.HandlerFunc
		isTTY   bool
		args    []string
	}{
		{
			nombre: "json/todo-correcto",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"id":"u-1","email":"a@b.c"}`))
			},
			args: []string{"doctor", "--json"},
		},
		{
			nombre: "json/backend-caido",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			args: []string{"doctor", "--json"},
		},
		{
			nombre: "texto/tty",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"id":"u-1","email":"a@b.c"}`))
			},
			isTTY: true,
			args:  []string{"doctor"},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			d, stdout, st := depsWithServer(t, caso.handler)
			d.IsTTY = caso.isTTY
			if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: tokenSecreto}); err != nil {
				t.Fatal(err)
			}

			root := testRoot(NewDoctorCmd(d), stdout)
			root.SetArgs(caso.args)
			if err := root.Execute(); err != nil {
				t.Fatalf("doctor: %v", err)
			}

			if strings.Contains(stdout.String(), tokenSecreto) {
				t.Errorf("doctor imprimió el token en salida %q: %s", caso.nombre, stdout.String())
			}
		})
	}
}

// TestDoctorConConfigCorruptaInformaEnVezDeFallar cubre el Critical de la
// ronda 1/5: appctx.BuildWithoutCredential devuelve error cuando el config.json
// global no parsea, y antes doctor propagaba ese error tal cual (exit 1). Un
// config.json corrupto es justo el tipo de instalación rota que doctor tiene
// que diagnosticar, no una que lo tumbe.
func TestDoctorConConfigCorruptaInformaEnVezDeFallar(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}
	writeGlobalConfig(t, d, "{not valid json")

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor nunca debe fallar, ni con una configuración corrupta: %v", err)
	}

	c := checksOf(t, stdout.Bytes())
	if c["versión"] != "ok" {
		t.Errorf("chequeo versión = %q, se esperaba ok incluso con configuración corrupta", c["versión"])
	}
	if c["configuración"] != "error" {
		t.Errorf("chequeo configuración = %q, se esperaba error", c["configuración"])
	}

	detalle := checkDetail(t, stdout.Bytes(), "configuración")
	if !strings.Contains(detalle, "config.json") {
		t.Errorf("el detalle de configuración no nombra el fichero: %q", detalle)
	}
	if !strings.Contains(detalle, "corríge") && !strings.Contains(detalle, "bórra") {
		t.Errorf("el detalle de configuración no dice cómo arreglarlo: %q", detalle)
	}
}

// TestDoctorConConfigCorruptaFuncionaEnLosCincoModos comprueba que, en el
// camino degradado (sin *appctx.Context, por tanto sin ctx.Render), los cinco
// modos de salida de presenter.Options siguen funcionando: auto (con y sin
// TTY, que son sus dos ramas), json, quiet, md y jq.
func TestDoctorConConfigCorruptaFuncionaEnLosCincoModos(t *testing.T) {
	casos := []struct {
		nombre string
		args   []string
		isTTY  bool
		// comprueba es específico de cada modo: no basta con "escribió algo",
		// porque eso no distingue un modo que ignora su rama (p. ej. --jq
		// tratado como si no se hubiera pasado, cayendo a JSON) de uno que sí
		// se aplicó. Cada caso comprueba la forma que solo ESE modo produce.
		comprueba func(t *testing.T, salida string)
	}{
		{
			nombre: "auto-tty",
			isTTY:  true,
			comprueba: func(t *testing.T, salida string) {
				// Texto (tabla), no JSON: no debe empezar por "{" ni "[".
				if !strings.Contains(salida, "COMPROBACIÓN") {
					t.Errorf("auto-tty: no parece la tabla de texto: %q", salida)
				}
				if strings.HasPrefix(strings.TrimSpace(salida), "{") {
					t.Errorf("auto-tty: parece JSON en vez de texto: %q", salida)
				}
			},
		},
		{
			nombre: "auto-pipe",
			comprueba: func(t *testing.T, salida string) {
				var env struct {
					OK bool `json:"ok"`
				}
				if err := json.Unmarshal([]byte(salida), &env); err != nil || !env.OK {
					t.Errorf("auto-pipe: no es el envelope JSON completo (ok:true): %v (%q)", err, salida)
				}
			},
		},
		{
			nombre: "json",
			args:   []string{"--json"},
			comprueba: func(t *testing.T, salida string) {
				var env struct {
					OK bool `json:"ok"`
				}
				if err := json.Unmarshal([]byte(salida), &env); err != nil || !env.OK {
					t.Errorf("json: no es el envelope JSON completo (ok:true): %v (%q)", err, salida)
				}
			},
		},
		{
			nombre: "quiet",
			args:   []string{"--quiet"},
			comprueba: func(t *testing.T, salida string) {
				// Quiet imprime solo Data (un array), sin el envelope. Ojo:
				// no basta con buscar la subcadena `"ok"` — un chequeo con
				// Status "ok" también la contiene como VALOR; lo que
				// distingue al envelope es la CLAVE de nivel superior
				// `"ok":`, que solo existe si se filtró el objeto completo.
				if !strings.HasPrefix(strings.TrimSpace(salida), "[") {
					t.Errorf("quiet: no empieza por un array: %q", salida)
				}
				if strings.Contains(salida, "\"ok\":") {
					t.Errorf("quiet: se filtró el envelope completo en vez de solo los datos: %q", salida)
				}
			},
		},
		{
			// IsTTY:true a propósito: si la rama --md de outputModeWithoutConfig
			// se perdiera, el modo caería a ModeAuto por defecto, que SÍ
			// mira IsTTY y renderizaría la tabla de texto en vez del
			// envelope — a diferencia de ModeMarkdown, que con
			// Result.Markdown nil cae a JSON sin mirar IsTTY. Con
			// IsTTY:false (como estaba antes) ambos caminos producen el
			// mismo JSON por coincidencia, y la mutación que borra esta
			// rama pasa desapercibida.
			nombre: "md",
			args:   []string{"--md"},
			isTTY:  true,
			comprueba: func(t *testing.T, salida string) {
				// doctor no define Result.Markdown: md cae al envelope JSON
				// completo, igual que json/auto-pipe (presenter.Render, caso
				// ModeMarkdown), incluso con IsTTY:true.
				var env struct {
					OK bool `json:"ok"`
				}
				if err := json.Unmarshal([]byte(salida), &env); err != nil || !env.OK {
					t.Errorf("md: no es el envelope JSON completo (ok:true): %v (%q)", err, salida)
				}
			},
		},
		{
			// ".data | length" (en vez de ".data") para que la comprobación
			// no pueda confundirse con la de "quiet": si la rama --jq se
			// perdiera y el flujo cayera al JSON por defecto, esto NO
			// produciría "2", así que la mutación queda capturada incluso
			// cuando el resultado por accidente también empieza por "[".
			nombre: "jq",
			args:   []string{"--jq", ".data | length"},
			comprueba: func(t *testing.T, salida string) {
				if strings.TrimSpace(salida) != "2" {
					t.Errorf("jq: %q, se esperaba \"2\" (versión + configuración)", salida)
				}
			},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
			if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
				t.Fatal(err)
			}
			writeGlobalConfig(t, d, "{not valid json")
			d.IsTTY = caso.isTTY

			root := testRoot(NewDoctorCmd(d), stdout)
			root.SetArgs(append([]string{"doctor"}, caso.args...))
			if err := root.Execute(); err != nil {
				t.Fatalf("doctor (%s) con configuración corrupta: %v", caso.nombre, err)
			}
			if stdout.Len() == 0 {
				t.Fatalf("doctor (%s) con configuración corrupta no escribió nada en stdout", caso.nombre)
			}
			caso.comprueba(t, stdout.String())
		})
	}
}

// TestDoctorOrganizacionDesdeCredencialSeEtiquetaCorrectamente cubre el
// Important 1 de la ronda 1/5: cuando la organización sale del fallback
// `if org == "" { org = cred.Org }`, etiquetarla con
// ctx.Cfg.Get(config.KeyOrg).Source miente (da "default", porque la
// configuración nunca trajo ningún valor de org).
func TestDoctorOrganizacionDesdeCredencialSeEtiquetaCorrectamente(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"u-1","email":"a@b.c"}`))
	})
	// depsWithServer fija CALLIOPE_ORG=acme; se anula para que la
	// organización no pueda salir de ninguna capa de configuración y el
	// único origen posible sea la credencial.
	base := d.Env
	d.Env = func(k string) string {
		if k == "CALLIOPE_ORG" {
			return ""
		}
		return base(k)
	}
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k", Org: "org-desde-credencial"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	detalle := checkDetail(t, stdout.Bytes(), "organización")
	if detalle != "org-desde-credencial (credencial)" {
		t.Errorf("detalle de organización = %q, se esperaba %q", detalle, "org-desde-credencial (credencial)")
	}
}

// TestDoctorConectividadConBaseURLInvalidaSaleEnEspanol es la variante a
// nivel de doctor del Important 2: con un base_url que contiene un carácter
// de control, http.NewRequestWithContext falla dentro de sdk.Do. Antes de la
// corrección en internal/sdk/client.go, ese error salía crudo y en inglés
// como Detail del chequeo de conectividad.
func TestDoctorConectividadConBaseURLInvalidaSaleEnEspanol(t *testing.T) {
	d, stdout, st := depsWithServer(t, func(w http.ResponseWriter, r *http.Request) {})
	base := d.Env
	d.Env = func(k string) string {
		if k == "CALLIOPE_BASE_URL" {
			return "http://ejemplo.invalido/\x00"
		}
		return base(k)
	}
	if err := st.Save(auth.Credential{Kind: auth.KindAPIKey, Token: "k"}); err != nil {
		t.Fatal(err)
	}

	root := testRoot(NewDoctorCmd(d), stdout)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	c := checksOf(t, stdout.Bytes())
	if c["conectividad"] != "error" {
		t.Errorf("chequeo conectividad = %q, se esperaba error con un base_url inválido", c["conectividad"])
	}
	detalle := checkDetail(t, stdout.Bytes(), "conectividad")
	if strings.Contains(detalle, "net/url") || strings.Contains(detalle, "control character") {
		t.Errorf("el detalle de conectividad filtra un error crudo en inglés: %q", detalle)
	}
}
