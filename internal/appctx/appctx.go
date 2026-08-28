// Package appctx monta el contexto de una invocación: configuración,
// credencial, cliente y modo de salida. Es el único punto donde se juntan las
// cuatro capas; los comandos reciben un *Context ya montado.
package appctx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/calliope/calliope-cli/internal/auth"
	"github.com/calliope/calliope-cli/internal/config"
	"github.com/calliope/calliope-cli/internal/output"
	"github.com/calliope/calliope-cli/internal/presenter"
	"github.com/calliope/calliope-cli/internal/sdk"
	"github.com/calliope/calliope-cli/internal/version"
)

// Deps son las dependencias externas de una invocación, inyectables para poder
// probar el wiring sin tocar el entorno real.
type Deps struct {
	Cwd    string
	Env    func(string) string
	Store  auth.Store
	Stdout io.Writer
	Stderr io.Writer
	IsTTY  bool
	// ReleasesURL es el endpoint que consulta el aviso de nueva versión.
	// Inyectable (en vez de leer version.ReleasesURL directamente) para que
	// los tests de cli puedan apuntarlo a un servidor de prueba sin tocar la
	// red real ni una variable global mutable.
	ReleasesURL string
}

// DefaultDeps son las dependencias reales del proceso.
func DefaultDeps() Deps {
	cwd, _ := os.Getwd()
	env := os.Getenv
	return Deps{
		Cwd:         cwd,
		Env:         env,
		Store:       auth.DefaultStore(filepath.Dir(config.GlobalPath(env))),
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		IsTTY:       term.IsTerminal(int(os.Stdout.Fd())),
		ReleasesURL: version.ReleasesURL,
	}
}

// RegisterGlobalFlags declara los flags que toda invocación entiende. Vive
// aquí y no en cli para que los tests de comandos puedan montar una raíz
// idéntica a la real sin duplicar la lista, que si no se desviaría.
func RegisterGlobalFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.String("org", "", "organización sobre la que operar")
	f.Bool("json", false, "salida JSON con envelope completo")
	f.Bool("quiet", false, "salida solo de datos, sin envelope")
	f.Bool("md", false, "salida en Markdown")
	f.String("jq", "", "filtra la salida con una expresión jq")

	// cmd.Flags() (el FlagSet local) no incluye los persistentes hasta que
	// Cobra los fusiona, y eso solo ocurre en ParseFlags/Execute/Help. Un
	// comando construido para tests que nunca pasa por Execute (como el que
	// arma commandWithFlags en appctx_test.go) no vería nunca los valores que
	// le fija con cmd.Flags().Set(...). Se fusiona aquí mismo: AddFlagSet no
	// copia los *Flag, comparte el puntero, así que Set y Get concuerden
	// tanto si el comando se ejecuta de verdad como si no.
	cmd.Flags().AddFlagSet(f)
}

// Context es todo lo que un comando necesita para hacer su trabajo.
type Context struct {
	Cfg        *config.Config
	Cred       auth.Credential
	CredSource string
	Client     *sdk.Client
	Org        string
	Present    presenter.Options
	Deps       Deps
}

// Build monta el contexto y exige credencial y organización. Es lo que usan
// todos los comandos que hablan con el backend.
func Build(cmd *cobra.Command, d Deps) (*Context, error) {
	ctx, err := BuildSinCredencial(cmd, d)
	if err != nil {
		return nil, err
	}

	cred, origen, err := auth.Resolve(d.Env, d.Store)
	if err != nil {
		return nil, err
	}
	ctx.Cred = cred
	ctx.CredSource = origen

	if ctx.Org == "" {
		ctx.Org = cred.Org
	}
	if ctx.Org == "" {
		return nil, output.NewError(output.CodeUsage,
			"No hay ninguna organización seleccionada.",
			"Elige una con: calliope orgs use <nombre>   (lista las disponibles con: calliope orgs list)")
	}

	ctx.Client = sdk.New(sdk.Options{
		BaseURL:    ctx.Cfg.BaseURL(),
		Credential: cred,
		Timeout:    timeoutOf(ctx.Cfg),
		UserAgent:  "calliope-cli/" + version.Version,
	})
	return ctx, nil
}

// BuildSinCredencial monta lo que se puede montar sin estar autenticado. Lo
// usan config, version y doctor, que tienen que funcionar precisamente cuando
// la autenticación es el problema.
func BuildSinCredencial(cmd *cobra.Command, d Deps) (*Context, error) {
	flags := map[string]string{}
	if v, _ := cmd.Flags().GetString("org"); v != "" {
		flags[config.KeyOrg] = v
	}

	cfg, avisos, err := config.Load(d.Cwd, d.Env, flags)
	if err != nil {
		return nil, err
	}
	for _, a := range avisos {
		fmt.Fprintln(d.Stderr, a)
	}

	return &Context{
		Cfg:     cfg,
		Org:     cfg.Org(),
		Present: OutputMode(cmd, cfg, d),
		Deps:    d,
	}, nil
}

// ResolveOutputMode resuelve el modo de salida de esta invocación con la
// misma lógica que usan Build y BuildWithoutCredential -flags, configuración
// y TTY, en ese orden-, para que main() pueda decidir cómo mostrar un ERROR
// exactamente igual que decide cómo mostrar un éxito. cmd es el
// *cobra.Command que devuelve root.ExecuteC(): el que de verdad se ejecutó,
// con sus flags ya fusionados y parseados (incluso si la propia ejecución
// falló después).
//
// Antes, main comprobaba con slices.Contains(os.Args, "--json"),
// desconectado de esta resolución: en tubería, con --jq, --quiet, --md,
// --json=true o CALLIOPE_OUTPUT=json el éxito salía en JSON pero el error
// salía en texto plano, así que la promesa del SKILL.md de que los fallos
// salen como envelope solo era cierta si el token literal "--json" estaba en
// la línea de comandos (C2 de la oleada final).
//
// Si la configuración no carga (p. ej. un config.json corrupto es
// precisamente el tipo de fallo que estamos informando), se usa una
// configuración vacía en su lugar en vez de propagar ese error: esta
// función solo decide el FORMATO del error que ya se está informando, nunca
// debe fallar por su cuenta. Los avisos de config.Load se descartan a
// propósito: si la invocación llegó a construir un *Context antes de
// fallar, ya los habrá impreso una vez; imprimirlos aquí también los
// duplicaría en stderr.
func ResolveOutputMode(cmd *cobra.Command, d Deps) presenter.Options {
	flags := map[string]string{}
	if v, _ := cmd.Flags().GetString("org"); v != "" {
		flags[config.KeyOrg] = v
	}
	cfg, _, err := config.Load(d.Cwd, d.Env, flags)
	if err != nil {
		cfg = &config.Config{}
	}
	return OutputMode(cmd, cfg, d)
}

// Render escribe un resultado con el modo de salida de esta invocación.
func (c *Context) Render(r presenter.Result) error {
	return presenter.Render(r, c.Present)
}

// OutputMode resuelve el modo de salida de una invocación: la configuración
// puede fijar un modo por defecto, y los flags de la invocación mandan sobre
// ella. Exportada para que main() pueda resolver el modo de un ERROR con la
// misma función exacta que usan Build y BuildWithoutCredential para el éxito
// (ver ResolveOutputMode, más arriba).
func OutputMode(cmd *cobra.Command, cfg *config.Config, d Deps) presenter.Options {
	opts := presenter.Options{Mode: presenter.ModeAuto, IsTTY: d.IsTTY, Out: d.Stdout}

	// El fichero de configuración puede fijar el modo por defecto; los flags
	// mandan sobre él.
	switch cfg.Output() {
	case "json":
		opts.Mode = presenter.ModeJSON
	case "quiet":
		opts.Mode = presenter.ModeQuiet
	case "md":
		opts.Mode = presenter.ModeMarkdown
	}

	if v, _ := cmd.Flags().GetString("jq"); v != "" {
		opts.Mode, opts.JQExpr = presenter.ModeJQ, v
		return opts
	}
	if v, _ := cmd.Flags().GetBool("json"); v {
		opts.Mode = presenter.ModeJSON
	}
	if v, _ := cmd.Flags().GetBool("quiet"); v {
		opts.Mode = presenter.ModeQuiet
	}
	if v, _ := cmd.Flags().GetBool("md"); v {
		opts.Mode = presenter.ModeMarkdown
	}
	return opts
}

func timeoutOf(cfg *config.Config) time.Duration {
	d, err := time.ParseDuration(cfg.Get(config.KeyTimeout).Value)
	if err != nil || d <= 0 {
		return 60 * time.Second
	}
	return d
}
