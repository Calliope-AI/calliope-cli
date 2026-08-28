#!/usr/bin/env bash
# Avisa al empezar la sesión si calliope no está listo, para que el agente no
# descubra el problema a mitad de una tarea. Contrato: nunca falla, nunca
# ensucia la salida y siempre termina con código 0 -- esto es un aviso, no
# una comprobación bloqueante.
set -uo pipefail

if ! command -v calliope >/dev/null 2>&1; then
  echo "calliope no está instalado. Instálalo con: brew install Calliope-AI/tap/calliope"
  exit 0
fi

# `calliope doctor` nunca devuelve código de error (ver doctor_test.go), así
# que la señal de "no listo" hay que leerla del propio diagnóstico, no del
# código de salida del proceso. Se cuenta con `--jq` (el filtro jq embebido en
# el binario, sin dependencia externa -es lo que exige la primera invariante
# del propio SKILL.md) en vez de con grep sobre el texto: así no depende de
# que la salida de --quiet siga siendo JSON indentado. Si esto alguna vez
# devuelve algo que no sea exactamente "0", se trata como "no listo" -vacío
# (fallo o timeout) incluido, así el hook nunca predica éxito sin confirmarlo.
#
# Cuando hay credencial configurada, doctor hace además una petición de red
# real con un timeout interno de 10s: en un hook de arranque eso se nota, así
# que lo acotamos más si el sistema tiene `timeout`/`gtimeout` (coreutils
# GNU). Si no los tiene -macOS de fábrica no los trae-, se ejecuta sin acotar
# más: el límite interno de 10s del propio binario sigue garantizando que
# esto nunca cuelga la sesión indefinidamente, solo que en el peor caso
# (credencial configurada + red caída) el aviso tarda hasta 10s en lugar de 5s.
limitador=""
if command -v timeout >/dev/null 2>&1; then
  limitador="timeout 5"
elif command -v gtimeout >/dev/null 2>&1; then
  limitador="gtimeout 5"
fi

fallos="$($limitador calliope doctor --json --jq '[.data[]|select(.status=="error")]|length' 2>/dev/null)"
estado=$?

if [ "$estado" -ne 0 ] || [ "$fallos" != "0" ]; then
  echo "calliope está instalado pero no listo. Diagnostica con: calliope doctor"
  exit 0
fi

org="$(calliope config get org --quiet 2>/dev/null | tr -d '"')"
if [ -z "$org" ]; then
  org="sin organización activa"
fi
echo "calliope listo: $org"
exit 0
