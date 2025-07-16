#!/bin/bash

# Script de prueba para generar Excel en entorno de producción
echo "🌐 PRUEBA DE GENERACIÓN DE EXCEL EN PRODUCCIÓN"
echo "=============================================="

# Configurar variables
PRODUCTION_URL="https://olimpo-app-t6qn9.ondigitalocean.app"
LOCAL_URL="http://localhost:8080"

# Detectar si estamos en local o producción
read -p "¿Probar en producción? (y/n): " TEST_PROD

if [ "$TEST_PROD" = "y" ] || [ "$TEST_PROD" = "Y" ]; then
    BASE_URL="$PRODUCTION_URL"
    echo "🚀 Probando en PRODUCCIÓN: $BASE_URL"
else
    BASE_URL="$LOCAL_URL"
    echo "🏠 Probando en LOCAL: $BASE_URL"
fi

# Historia académica de prueba
ACADEMIC_HISTORY='Portal de Servicios AcadémicosjoroblesrDatos personales Información académica Mi historia académicaMi horarioMis planesMis tutoresProceso de inscripción Buscador de cursos Catálogo prog. curriculares Información Financiera Trámites y solicitudes Evaluación docente Historia Académica  Plan de estudiosINGENIERÍA DE SISTEMAS E INFORMÁTICAFacultad: FACULTAD DE MINASHist. Acad.: 1224ESTADO ABIERTOResumen4.3 (Acumulado)Pregrado - Promedio académico2024-2S4.3 (Acumulado)Pregrado - P.A.P.A2024-2SAsignaturasAsignaturasCréditosTipoPeriodoCalificaciónDesarrollo móvil (3011171)3DISCIPLINAR OPTATIVA2024-2S Ordinaria4.7APROBADADesarrollo web I (3011019)3DISCIPLINAR OPTATIVA2024-2S Ordinaria4.8APROBADACalidad de software (3010440)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.2APROBADAEstructuración y evaluación de proyectos de ingeniería (3010407)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.7APROBADAFundamentos de analítica (3011020)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.9APROBADAIntroducción a la inteligencia artificial (3010476)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.3APROBADACátedra de sistemas: una visión histórico-cultural de la computación (3010836)3DISCIPLINAR OPTATIVA2024-1S Ordinaria4.7APROBADABASE DE DATOS I (3007847)3DISCIPLINAR OBLIGATORIA2024-1S Ordinaria5.0APROBADAINGENIERÍA DE REQUISITOS (3007852)3DISCIPLINAR OBLIGATORIA2024-1S Ordinaria5.0APROBADAIntroducción al análisis de decisiones (3010415)3DISCIPLINAR OBLIGATORIA2024-1S Ordinaria4.3APROBADAREDES Y TELECOMUNICACIONES I (3007865)3DISCIPLINAR OBLIGATORIA2024-1S Ordinaria3.7APROBADAESTADÍSTICA II (3006915)4FUND. OPTATIVA2023-2S Ordinaria4.3APROBADAESTRUCTURA DE DATOS (3007741)3DISCIPLINAR OBLIGATORIA2023-2S Ordinaria4.2APROBADAFundamentos de proyectos en ingeniería (3010408)3DISCIPLINAR OBLIGATORIA2023-2S Ordinaria4.7APROBADASIMULACIÓN DE SISTEMAS (3007331)3DISCIPLINAR OBLIGATORIA2023-2S Ordinaria4.1APROBADASISTEMAS OPERATIVOS (3007867)3DISCIPLINAR OBLIGATORIA2023-2S Ordinaria4.8APROBADAVisión Artificial (3009550)3LIBRE ELECCIÓN2023-2S Ordinaria4.5APROBADAECUACIONES DIFERENCIALES (1000007-M)4FUND. OPTATIVA2023-1S Ordinaria3.0APROBADAMÉTODOS NUMÉRICOS (3006907)4FUND. OPTATIVA2023-1S Ordinaria3.6APROBADAINGENIERÍA DE SOFTWARE (3007853)3DISCIPLINAR OBLIGATORIA2023-1S Ordinaria4.7APROBADAINVESTIGACIÓN DE OPERACIONES I (3007324)3DISCIPLINAR OBLIGATORIA2023-1S Ordinaria4.1APROBADATeoría de lenguajes de programación (3010426)3DISCIPLINAR OBLIGATORIA2023-1S Ordinaria4.7APROBADAEstadística I (3010651)3FUND. OBLIGATORIA2022-2S Ordinaria4.4APROBADAMATEMÁTICAS DISCRETAS (3006906)4FUND. OBLIGATORIA2022-2S Ordinaria3.4APROBADAQuímica general (3006829)3FUND. OPTATIVA2022-2S Ordinaria4.3APROBADAIntroducción a la ingeniería de sistemas e informática (3010438)2DISCIPLINAR OBLIGATORIA2022-2S Ordinaria5.0APROBADAPROGRAMACIÓN ORIENTADA A OBJETOS (3007744)3DISCIPLINAR OBLIGATORIA2022-2S Ordinaria4.9APROBADACATEDRA ANTIOQUIA (3007373)3LIBRE ELECCIÓN2022-2S Ordinaria4.8APROBADACátedra Ingenierías Facultad de Minas (3009511)2LIBRE ELECCIÓN2022-2S Ordinaria4.1APROBADAINGLÉS I (1000044-M)3NIVELACIÓN2022-2S Validacion por suficienciaAPROBADAINGLÉS II (1000045-M)3NIVELACIÓN2022-2S Validacion por suficienciaAPROBADAINGLÉS III (1000046-M)3NIVELACIÓN2022-2S Validacion por suficienciaAPROBADAINGLÉS IV (1000047-M)3NIVELACIÓN2022-2S Validacion por suficienciaAPROBADAFundamentos de programación (3010435)3DISCIPLINAR OBLIGATORIA2021-2S Ordinaria4.6APROBADAÁLGEBRA LINEAL (1000003-M)4FUND. OBLIGATORIA2021-1S Ordinaria4.0APROBADACÁLCULO INTEGRAL (1000005-M)4FUND. OBLIGATORIA2021-1S Ordinaria3.5APROBADAFÍSICA MECÁNICA (1000019-M)4FUND. OBLIGATORIA2021-1S Ordinaria3.7APROBADACátedra estudiantil: universidad, participación y sociedad (3010348)3LIBRE ELECCIÓN2021-1S Ordinaria4.5APROBADACÁLCULO DIFERENCIAL (1000004-M)4FUND. OBLIGATORIA2020-2S Ordinaria4.9APROBADAGEOMETRÍA VECTORIAL Y ANALÍTICA (1000008-M)4FUND. OBLIGATORIA2020-2S Ordinaria3.6APROBADACIENCIA DE LOS MATERIALES (3007309)3LIBRE ELECCIÓN2020-2S Ordinaria3.5APROBADACátedra Nacional de Inducción y Preparación para la Vida Universitaria (1000089-M)2LIBRE ELECCIÓN2020-1S OrdinariaAPROBADA'

echo ""
echo "📋 Preparando datos de prueba..."

# Crear JSON para la petición
JSON_DATA=$(jq -n \
  --arg history "$ACADEMIC_HISTORY" \
  --arg career "ISIS" \
  '{
    "academic_history_text": $history,
    "target_career_code": $career
  }')

echo ""
echo "🧪 PRUEBA 1: Cambio de Carrera Excel"
echo "====================================="
echo "🎯 Endpoint: $BASE_URL/api/cambio-carrera/excel"

RESPONSE1=$(curl -s -X POST "$BASE_URL/api/cambio-carrera/excel" \
  -H "Content-Type: application/json" \
  -d "$JSON_DATA")

echo "📋 Respuesta:"
echo "$RESPONSE1" | jq '.'

# Verificar éxito
SUCCESS1=$(echo "$RESPONSE1" | jq -r '.success // false')
if [ "$SUCCESS1" = "true" ]; then
    echo "✅ ¡Cambio de carrera Excel exitoso!"
    DOWNLOAD_URL1=$(echo "$RESPONSE1" | jq -r '.download_url')
    echo "🔗 URL: $DOWNLOAD_URL1"
else
    echo "❌ Error en cambio de carrera Excel"
    echo "$RESPONSE1" | jq -r '.error // "Error desconocido"'
fi

echo ""
echo "🧪 PRUEBA 2: Doble Titulación Excel"
echo "===================================="
echo "🎯 Endpoint: $BASE_URL/api/doble-titulacion/excel"

# Datos para doble titulación
JSON_DATA_DOBLE=$(jq -n \
  --arg historia_origen "$ACADEMIC_HISTORY" \
  --arg historia_doble "$ACADEMIC_HISTORY" \
  --arg codigo_carrera "ISIS" \
  '{
    "historia_origen": $historia_origen,
    "historia_doble": $historia_doble,
    "codigo_carrera_objetivo": $codigo_carrera
  }')

RESPONSE2=$(curl -s -X POST "$BASE_URL/api/doble-titulacion/excel" \
  -H "Content-Type: application/json" \
  -d "$JSON_DATA_DOBLE")

echo "📋 Respuesta:"
echo "$RESPONSE2" | jq '.'

# Verificar éxito
SUCCESS2=$(echo "$RESPONSE2" | jq -r '.success // false')
if [ "$SUCCESS2" = "true" ]; then
    echo "✅ ¡Doble titulación Excel exitoso!"
    DOWNLOAD_URL2=$(echo "$RESPONSE2" | jq -r '.download_url')
    echo "🔗 URL: $DOWNLOAD_URL2"
else
    echo "❌ Error en doble titulación Excel"
    echo "$RESPONSE2" | jq -r '.error // "Error desconocido"'
fi

echo ""
echo "📊 RESUMEN DE PRUEBAS"
echo "===================="
if [ "$SUCCESS1" = "true" ] && [ "$SUCCESS2" = "true" ]; then
    echo "✅ Todas las pruebas pasaron exitosamente"
    echo "🎉 El sistema de Excel está funcionando correctamente"
else
    echo "❌ Algunas pruebas fallaron"
    if [ "$SUCCESS1" != "true" ]; then
        echo "  - Cambio de carrera: FALLÓ"
    fi
    if [ "$SUCCESS2" != "true" ]; then
        echo "  - Doble titulación: FALLÓ"
    fi
fi

echo ""
echo "🔧 INFORMACIÓN TÉCNICA"
echo "======================"
echo "• URL base: $BASE_URL"
echo "• Content-Type: application/json"
echo "• Método: POST"
echo "• Directorios: static/reports/"
echo ""
echo "✅ Prueba completada" 