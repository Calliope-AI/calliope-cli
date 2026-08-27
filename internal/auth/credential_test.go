package auth

import "testing"

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
