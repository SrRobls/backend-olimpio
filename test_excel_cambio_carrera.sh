#!/bin/bash

# Script de prueba para generar Excel de cambio de carrera
echo "📊 PRUEBA DE GENERACIÓN DE EXCEL PARA CAMBIO DE CARRERA"
echo "====================================================="

# Historia académica del usuario de Ingeniería de Sistemas
ACADEMIC_HISTORY='Portal de Servicios AcadémicosjoroblesrDatos personales Información académica Mi historia académicaMi horarioMis planesMis tutoresProceso de inscripción Buscador de cursos Catálogo prog. curriculares Información Financiera Trámites y solicitudes Evaluación docente Historia Académica  Plan de estudiosINGENIERÍA DE SISTEMAS E INFORMÁTICAFacultad: FACULTAD DE MINASHist. Acad.: 1224ESTADO ABIERTOResumen4.3 (Acumulado)Pregrado - Promedio académico2024-2S4.3 (Acumulado)Pregrado - P.A.P.A2024-2SAsignaturasAsignaturasCréditosTipoPeriodoCalificaciónDesarrollo móvil (3011171)3DISCIPLINAR OPTATIVA2024-2S Ordinaria4.7APROBADADesarrollo web I (3011019)3DISCIPLINAR OPTATIVA2024-2S Ordinaria4.8APROBADACalidad de software (3010440)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.2APROBADAEstructuración y evaluación de proyectos de ingeniería (3010407)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.7APROBADAFundamentos de analítica (3011020)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.9APROBADAIntroducción a la inteligencia artificial (3010476)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.3APROBADACátedra de sistemas: una visión histórico-cultural de la computación (3010836)3DISCIPLINAR OPTATIVA2024-1S Ordinaria4.7APROBADABASE DE DATOS I (3007847)3DISCIPLINAR OBLIGATORIA2024-1S Ordinaria5.0APROBADAINGENIERÍA DE REQUISITOS (3007852)3DISCIPLINAR OBLIGATORIA2024-1S Ordinaria5.0APROBADAIntroducción al análisis de decisiones (3010415)3DISCIPLINAR OBLIGATORIA2024-1S Ordinaria4.3APROBADAREDES Y TELECOMUNICACIONES I (3007865)3DISCIPLINAR OBLIGATORIA2024-1S Ordinaria3.7APROBADAESTADÍSTICA II (3006915)4FUND. OPTATIVA2023-2S Ordinaria4.3APROBADAESTRUCTURA DE DATOS (3007741)3DISCIPLINAR OBLIGATORIA2023-2S Ordinaria4.2APROBADAFundamentos de proyectos en ingeniería (3010408)3DISCIPLINAR OBLIGATORIA2023-2S Ordinaria4.7APROBADASIMULACIÓN DE SISTEMAS (3007331)3DISCIPLINAR OBLIGATORIA2023-2S Ordinaria4.1APROBADASISTEMAS OPERATIVOS (3007867)3DISCIPLINAR OBLIGATORIA2023-2S Ordinaria4.8APROBADAVisión Artificial (3009550)3LIBRE ELECCIÓN2023-2S Ordinaria4.5APROBADAECUACIONES DIFERENCIALES (1000007-M)4FUND. OPTATIVA2023-1S Ordinaria3.0APROBADAMÉTODOS NUMÉRICOS (3006907)4FUND. OPTATIVA2023-1S Ordinaria3.6APROBADAINGENIERÍA DE SOFTWARE (3007853)3DISCIPLINAR OBLIGATORIA2023-1S Ordinaria4.7APROBADAINVESTIGACIÓN DE OPERACIONES I (3007324)3DISCIPLINAR OBLIGATORIA2023-1S Ordinaria4.1APROBADATeoría de lenguajes de programación (3010426)3DISCIPLINAR OBLIGATORIA2023-1S Ordinaria4.7APROBADAEstadística I (3010651)3FUND. OBLIGATORIA2022-2S Ordinaria4.4APROBADAMATEMÁTICAS DISCRETAS (3006906)4FUND. OBLIGATORIA2022-2S Ordinaria3.4APROBADAQuímica general (3006829)3FUND. OPTATIVA2022-2S Ordinaria4.3APROBADAIntroducción a la ingeniería de sistemas e informática (3010438)2DISCIPLINAR OBLIGATORIA2022-2S Ordinaria5.0APROBADAPROGRAMACIÓN ORIENTADA A OBJETOS (3007744)3DISCIPLINAR OBLIGATORIA2022-2S Ordinaria4.9APROBADACATEDRA ANTIOQUIA (3007373)3LIBRE ELECCIÓN2022-2S Ordinaria4.8APROBADACátedra Ingenierías Facultad de Minas (3009511)2LIBRE ELECCIÓN2022-2S Ordinaria4.1APROBADAINGLÉS I (1000044-M)3NIVELACIÓN2022-2S Validacion por suficienciaAPROBADAINGLÉS II (1000045-M)3NIVELACIÓN2022-2S Validacion por suficienciaAPROBADAINGLÉS III (1000046-M)3NIVELACIÓN2022-2S Validacion por suficienciaAPROBADAINGLÉS IV (1000047-M)3NIVELACIÓN2022-2S Validacion por suficienciaAPROBADAFundamentos de programación (3010435)3DISCIPLINAR OBLIGATORIA2021-2S Ordinaria4.6APROBADAÁLGEBRA LINEAL (1000003-M)4FUND. OBLIGATORIA2021-1S Ordinaria4.0APROBADACÁLCULO INTEGRAL (1000005-M)4FUND. OBLIGATORIA2021-1S Ordinaria3.5APROBADAFÍSICA MECÁNICA (1000019-M)4FUND. OBLIGATORIA2021-1S Ordinaria3.7APROBADACátedra estudiantil: universidad, participación y sociedad (3010348)3LIBRE ELECCIÓN2021-1S Ordinaria4.5APROBADACÁLCULO DIFERENCIAL (1000004-M)4FUND. OBLIGATORIA2020-2S Ordinaria4.9APROBADAGEOMETRÍA VECTORIAL Y ANALÍTICA (1000008-M)4FUND. OBLIGATORIA2020-2S Ordinaria3.6APROBADACIENCIA DE LOS MATERIALES (3007309)3LIBRE ELECCIÓN2020-2S Ordinaria3.5APROBADACátedra Nacional de Inducción y Preparación para la Vida Universitaria (1000089-M)2LIBRE ELECCIÓN2020-1S OrdinariaAPROBADA'

echo "📤 Enviando petición para generar Excel de cambio de carrera..."
echo ""

# Crear el JSON para la petición usando jq
JSON_DATA=$(jq -n \
  --arg history "$ACADEMIC_HISTORY" \
  --arg career "ISIS" \
  '{
    "academic_history_text": $history,
    "target_career_code": $career
  }')

echo "🔍 Probando endpoint: /api/cambio-carrera/excel"
echo "Carrera objetivo: ISIS (Ingeniería de Sistemas)"
echo ""

# Realizar la petición POST al nuevo endpoint de Excel
RESPONSE=$(curl -s -X POST http://localhost:8080/api/cambio-carrera/excel \
  -H "Content-Type: application/json" \
  -d "$JSON_DATA")

echo "📋 Respuesta del servidor:"
echo "$RESPONSE" | jq '.'
echo ""

# Extraer URL de descarga de la respuesta
DOWNLOAD_URL=$(echo "$RESPONSE" | jq -r '.download_url // empty')
FILENAME=$(echo "$RESPONSE" | jq -r '.filename // empty')
SUCCESS=$(echo "$RESPONSE" | jq -r '.success // false')

if [ "$SUCCESS" = "true" ] && [ "$DOWNLOAD_URL" != "" ]; then
    echo "✅ ¡Excel generado exitosamente!"
    echo "📁 Archivo: $FILENAME"
    echo "🔗 URL de descarga: $DOWNLOAD_URL"
    echo ""
    
    # Verificar si el archivo existe localmente
    LOCAL_PATH="static/reports/$FILENAME"
    if [ -f "$LOCAL_PATH" ]; then
        echo "✅ Archivo confirmado en el servidor: $LOCAL_PATH"
        echo "📊 Tamaño: $(du -h "$LOCAL_PATH" | cut -f1)"
    else
        echo "❌ Archivo no encontrado localmente: $LOCAL_PATH"
    fi
    
    echo ""
    echo "🌐 Para descargar el archivo, abre esta URL en tu navegador:"
    echo "$DOWNLOAD_URL"
    echo ""
    echo "📋 Información del reporte:"
    echo "$RESPONSE" | jq '.report_info'
else
    echo "❌ Error generando el Excel:"
    echo "$RESPONSE" | jq '.error // "Error desconocido"'
fi

echo ""
echo "✅ Prueba completada para Excel de cambio de carrera" 