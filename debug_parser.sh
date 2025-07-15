#!/bin/bash

# Script de debug para el parser
echo "🔍 DEBUG DEL PARSER DE HISTORIA ACADÉMICA"
echo "========================================"

# Solo algunas materias para debug
ACADEMIC_HISTORY='Desarrollo móvil (3011171)3DISCIPLINAR OPTATIVA2024-2S Ordinaria4.7APROBADADesarrollo web I (3011019)3DISCIPLINAR OPTATIVA2024-2S Ordinaria4.8APROBADAcalidad de software (3010440)3DISCIPLINAR OBLIGATORIA2024-2S Ordinaria4.2APROBADA'

# Crear el JSON para la petición
JSON_DATA=$(jq -n \
  --arg history "$ACADEMIC_HISTORY" \
  --arg career "ISIS" \
  '{
    "academic_history_text": $history,
    "target_career_code": $career
  }')

echo "📤 Enviando petición de debug..."
echo "Historia académica de prueba: $ACADEMIC_HISTORY"
echo ""

# Realizar la petición POST al nuevo endpoint
curl -X POST http://localhost:8080/api/cambio-carrera-texto \
  -H "Content-Type: application/json" \
  -d "$JSON_DATA" \
  | jq '.parsed_subjects'

echo ""
echo "✅ Debug completado" 