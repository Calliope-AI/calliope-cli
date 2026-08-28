#!/usr/bin/env bash
# Avisa al empezar la sesión si calliope no está listo, para que el agente no
# descubra el problema a mitad de una tarea. Contrato: nunca falla, nunca
# ensucia la salida y siempre termina con código 0 -- esto es un aviso, no
# una comprobación bloqueante.
set -uo pipefail

if ! command -v calliope >/dev/null 2>&1; then
  echo "calliope no está instalado. Instálalo con: brew install calliope/tap/calliope"
  exit 0
fi

# `calliope doctor` nunca devuelve código de error (ver doctor_test.go), así
# que la señal de "no listo" hay que leerla del propio diagnóstico, no del
# código de salida del proceso. Cuando hay credencial configurada, doctor
# hace además una petición de red real con un timeout interno de 10s: en un
# hook de arranque eso se nota, así que lo acotamos más si el sistema tiene
# `timeout`/`gtimeout` (coreutils GNU). Si no los tiene -macOS de fábrica no
# los trae-, se ejecuta sin acotar más: el límite interno de 10s del propio
# binario sigue garantizando que esto nunca cuelga la sesión indefinidamente,
# solo que en el peor caso (credencial configurada + red caída) el aviso
# tarda hasta 10s en lugar de 5s.
limitador=""
if command -v timeout >/dev/null 2>&1; then
  limitador="timeout 5"
elif command -v gtimeout >/dev/null 2>&1; then
  limitador="gtimeout 5"
fi

diagnostico="$($limitador calliope doctor --quiet 2>/dev/null)"
estado=$?

if [ "$estado" -ne 0 ] || [ -z "$diagnostico" ] || printf '%s' "$diagnostico" | grep -q '"status": "error"'; then
  echo "calliope está instalado pero no listo. Diagnostica con: calliope doctor"
  exit 0
fi

org="$(calliope config get org --quiet 2>/dev/null | tr -d '"')"
if [ -z "$org" ]; then
  org="sin organización activa"
fi
echo "calliope listo: $org"
exit 0
