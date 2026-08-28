#!/usr/bin/env python3
"""Backend falso de Calliope Data, para desarrollo del CLI.

Implementa los endpoints que ejercita el smoke (test/e2e/smoke_test.go) con
respuestas de la forma que el CLI espera. No necesita dependencias.

    python3 test/e2e/backend-falso.py &
    CALLIOPE_E2E=1 CALLIOPE_API_KEY=cualquiera CALLIOPE_ORG=miorg \
      CALLIOPE_BASE_URL=http://127.0.0.1:8899 \
      go test -tags=e2e ./test/e2e/ -v

Sirve para dos cosas:

1. Correr el smoke sin credenciales ni backend, por ejemplo en CI.
2. Documentar de forma ejecutable las formas de respuesta. Los tipos `Me`,
   `Organization` y `SchemaResponse` de internal/sdk/models.go se dedujeron
   leyendo el cliente TypeScript de calliope-data-ui, no una respuesta real;
   lo que se devuelve aquí es lo que el CLI da por bueno. Si el backend de
   verdad usa otros nombres de campo, el síntoma no es un error: los campos
   salen vacíos con "ok": true.

No sustituye al smoke contra el backend real: un stub que devuelve lo que el
CLI espera nunca puede descubrir que el CLI espera algo equivocado.
"""
ORG = re.compile(r"^/v1/organizations/([^/]+)/(.+)$")
class H(http.server.BaseHTTPRequestHandler):
    def responde(self, obj, code=200):
        b = json.dumps(obj).encode()
        self.send_response(code); self.send_header("Content-Type","application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    def do_GET(self):
        if self.path == "/v1/auth/me":
            return self.responde({"id":"u-1","email":"prueba@calliope.so","name":"Prueba"})
        if self.path == "/v1/organizations":
            return self.responde([{"id":"o-1","name":"miorg","displayName":"Mi Org"}])
        m = ORG.match(self.path)
        if m and m.group(2).startswith("database/schema"):
            return self.responde({"tables":[{"name":"ventas","columns":[
                {"name":"id","type":"INTEGER"},{"name":"importe","type":"DOUBLE"}]}]})
        if m and m.group(2) == "rules":
            return self.responde([{"id":"r-1","name":"Margen mínimo","description":"El margen no baja del 12%"}])
        if m and m.group(2) == "knowledge/concepts":
            return self.responde({"concepts":[
                {"id":"c-1","name":"Venta","description":"Una transacción de venta"},
                {"id":"c-2","name":"Cliente","description":"Quien compra"}]})
        self.responde({"message":"no encontrado"}, 404)
    def do_POST(self):
        m = ORG.match(self.path)
        if m and m.group(2) == "ask":
            n=int(self.headers.get("Content-Length",0)); q=json.loads(self.rfile.read(n))["question"]
            return self.responde({"success":True,"text":f"Respuesta local a: {q}","rowCount":0,
                "sources":[{"citation":"ventas.parquet","documentId":"d-1"}]})
        self.responde({"message":"no encontrado"}, 404)
    def log_message(self,*a): pass
print("backend falso en :8899", flush=True)
http.server.HTTPServer(("127.0.0.1",8899),H).serve_forever()
