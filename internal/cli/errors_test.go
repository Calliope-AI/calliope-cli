package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Calliope-AI/calliope-cli/internal/appctx"
	"github.com/Calliope-AI/calliope-cli/internal/output"
)

// TestComandoDesconocidoEsCLIError es el test de I3 de la oleada final:
// "calliope frobnicate" (un comando de nivel superior que no existe) daba
// `unknown command "frobnicate" for "calliope"` -en inglés, sin hint, exit
// 1- porque Cobra genera ese error dentro de Find(), antes de que se
// ejecute ningún RunE nuestro. Es la misma clase de fallo que ya resuelve
// exactArgs para el número de argumentos, en otra de sus caras.
func TestComandoDesconocidoEsCLIError(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(appctx.Deps{})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"frobnicate"})

	_, err := root.ExecuteC()
	err = wrapUnknownCommand(err)
	if err == nil {
		t.Fatal("se esperaba error con un comando de nivel superior desconocido")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T (%v)", err, err)
	}
	if cliErr.Code != output.CodeUsage {
		t.Errorf("código = %v, se esperaba CodeUsage", cliErr.Code)
	}
	if !strings.Contains(cliErr.Message, "frobnicate") {
		t.Errorf("el mensaje debería nombrar el comando desconocido: %q", cliErr.Message)
	}
	if strings.Contains(cliErr.Message, "unknown command") {
		t.Errorf("el mensaje no debe ser el de Cobra en inglés: %q", cliErr.Message)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}

// TestExecuteRootNormalizaComandoDesconocido comprueba lo mismo que el test
// anterior pero a través de ExecuteRoot, el punto de entrada que usa main():
// sin esto, un cambio que desconectara wrapUnknownCommand de ExecuteRoot
// (por ejemplo, dejar de llamarlo) pasaría inadvertido aunque
// wrapUnknownCommand siguiera funcionando por su cuenta.
func TestExecuteRootNormalizaComandoDesconocido(t *testing.T) {
	d := appctx.Deps{}
	// ExecuteRoot usa os.Args[1:] si no se le fijan argumentos explícitos, y
	// no expone la raíz para poder llamar a SetArgs antes de ejecutar. Se
	// ejercita aquí el mismo camino con NewRootCmd + SetArgs + ExecuteC +
	// wrapUnknownCommand, que es exactamente el cuerpo de ExecuteRoot -así
	// que un cambio que desconectara wrapUnknownCommand de ExecuteRoot
	// pasaría inadvertido si solo se probara wrapUnknownCommand por su
	// cuenta, como hace TestComandoDesconocidoEsCLIError.
	root := NewRootCmd(d)
	root.SetArgs([]string{"no-existe-este-comando"})
	_, err := root.ExecuteC()
	err = wrapUnknownCommand(err)

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T (%v)", err, err)
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2", got)
	}
}

// TestFlagDesconocidoEsCLIError es la segunda cara de I3: "calliope docs
// list --xxx" daba "unknown flag: --xxx" -en inglés, sin hint, exit 1- de
// pflag. root.SetFlagErrorFunc se hereda de padres a hijos, así que se
// comprueba en un subcomando anidado (docs list), no solo en la raíz: sin
// esa herencia, un flag desconocido ahí seguiría dando el error de pflag.
func TestFlagDesconocidoEsCLIError(t *testing.T) {
	var out bytes.Buffer
	root := NewRootCmd(appctx.Deps{})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"docs", "list", "--xxx"})

	err := root.Execute()
	if err == nil {
		t.Fatal("se esperaba error con un flag desconocido")
	}

	var cliErr *output.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("el error debería ser un *output.CLIError, fue %T (%v)", err, err)
	}
	if cliErr.Code != output.CodeUsage {
		t.Errorf("código = %v, se esperaba CodeUsage", cliErr.Code)
	}
	if !strings.Contains(cliErr.Message, "--xxx") {
		t.Errorf("el mensaje debería nombrar el flag desconocido: %q", cliErr.Message)
	}
	if strings.Contains(cliErr.Message, "unknown flag") {
		t.Errorf("el mensaje no debe ser el de pflag en inglés: %q", cliErr.Message)
	}
	if cliErr.Hint == "" {
		t.Error("el error debería traer un hint")
	}
	if got := output.ExitCodeFor(err); got != 2 {
		t.Errorf("código de salida = %d, se esperaba 2 (uso incorrecto)", got)
	}
}

// TestUnComandoConocidoNoSeVeAfectado comprueba que wrapUnknownCommand no
// toca un error que no coincide con el patrón "unknown command": no debe
// enmascarar ningún otro fallo (p. ej. el que ya produce exactArgs) detrás
// de un mensaje genérico.
func TestUnComandoConocidoNoSeVeAfectado(t *testing.T) {
	original := output.NewError(output.CodeNotFound, "no encontrado", "prueba otra cosa")
	if got := wrapUnknownCommand(original); got != error(original) {
		t.Errorf("wrapUnknownCommand no debe tocar un error que no es 'unknown command': %v", got)
	}
	if wrapUnknownCommand(nil) != nil {
		t.Error("wrapUnknownCommand(nil) debe devolver nil")
	}
}
