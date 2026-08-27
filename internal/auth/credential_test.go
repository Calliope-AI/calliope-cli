package auth

import (
	"fmt"
	"strings"
	"testing"
)

func TestCabeceraDeAPIKey(t *testing.T) {
	c := Credential{Kind: KindAPIKey, Token: "cal_live_123"}
	k, v := c.Header()
	if k != "X-API-Key" || v != "cal_live_123" {
		t.Errorf("Header() = (%q, %q), se esperaba (X-API-Key, cal_live_123)", k, v)
	}
}

func TestCabeceraDeOAuth(t *testing.T) {
	c := Credential{Kind: KindOAuth, Token: "tok"}
	k, v := c.Header()
	if k != "Authorization" || v != "Bearer tok" {
		t.Errorf("Header() = (%q, %q), se esperaba (Authorization, Bearer tok)", k, v)
	}
}

func TestCredencialSinTokenNoEsValida(t *testing.T) {
	if (Credential{Kind: KindAPIKey}).Valid() {
		t.Error("una credencial sin token no debe ser válida")
	}
}

func TestCredentialStringRedactaElToken(t *testing.T) {
	c := Credential{Kind: KindAPIKey, Token: "cal_live_supersecreto", Org: "acme"}

	s := fmt.Sprintf("%v", c)
	if strings.Contains(s, c.Token) {
		t.Errorf("String() filtra el token: %q", s)
	}

	// Un %s directo también debe pasar por String(), no solo %v.
	s2 := fmt.Sprintf("%s", c)
	if strings.Contains(s2, c.Token) {
		t.Errorf("String() filtra el token con %%s: %q", s2)
	}

	if err := fmt.Errorf("fallo con credencial %v", c); strings.Contains(err.Error(), c.Token) {
		t.Errorf("un error formateado con la credencial filtra el token: %q", err.Error())
	}
}
