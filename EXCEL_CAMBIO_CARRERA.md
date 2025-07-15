# 📊 Generación de Excel para Cambio de Carrera

## 🎯 Descripción General

El endpoint `/api/cambio-carrera/excel` permite generar un informe completo en formato Excel (.xlsx) para el análisis de cambio de carrera. Este reporte incluye:

- ✅ **Materias homologables** (con códigos de colores)
- ❌ **Materias faltantes** (con códigos de colores)
- 📊 **Resumen estadístico** detallado
- 🎯 **Análisis por tipología** de materias
- 📈 **Porcentajes de avance** por categoría
- 📋 **Análisis detallado** del progreso académico

## 🚀 Endpoint

**URL:** `POST /api/cambio-carrera/excel`

**Content-Type:** `application/json` o `multipart/form-data`

### Estructura de la Petición (JSON)

```json
{
  "academic_history_text": "Historia académica completa del estudiante...",
  "target_career_code": "ISIS"
}
```

### Estructura de la Petición (Form-data)

```bash
curl -X POST http://localhost:8080/api/cambio-carrera/excel \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "academic_history_text=Historia académica..." \
  -d "target_career_code=ISIS"
```

## 📋 Respuesta Exitosa

```json
{
  "success": true,
  "message": "Reporte Excel de cambio de carrera generado exitosamente",
  "download_url": "http://localhost:8080/static/reports/Informe_Cambio_Carrera_ISIS_20241220_143025.xlsx",
  "filename": "Informe_Cambio_Carrera_ISIS_20241220_143025.xlsx",
  "report_info": {
    "carrera": "INGENIERÍA DE SISTEMAS E INFORMÁTICA",
    "codigo_carrera": "ISIS",
    "materias_parseadas": 47,
    "materias_homologables": 30,
    "creditos_homologables": 98,
    "materias_faltantes": 34,
    "creditos_faltantes": 108,
    "porcentaje_avance": "47.57%",
    "fecha_generacion": "20/12/2024 14:30:25"
  }
}
```

## 📊 Características del Excel Generado

### 1. **Información General**
- Fecha y hora de generación
- Carrera objetivo y código
- Plan de estudio utilizado

### 2. **Resumen Estadístico**
- Total de materias parseadas de la historia académica
- Materias homologables vs. faltantes
- Créditos completados vs. pendientes
- Porcentaje de avance global

### 3. **Materias Homologables** (Fondo Verde)
- Código y nombre de la materia
- Créditos asociados
- Tipología (FUND. OBLIGATORIA, DISCIPLINAR, etc.)
- Estado (APROBADA)
- Información de equivalencia (si aplica)

### 4. **Materias Faltantes** (Fondo Rojo)
- Código y nombre de la materia
- Créditos requeridos
- Tipología correspondiente
- Estado (PENDIENTE)

### 5. **Resumen por Tipología**
- **Fundamental Obligatoria**: Requeridos/Completados/Faltantes/% Avance
- **Fundamental Optativa**: Requeridos/Completados/Faltantes/% Avance
- **Disciplinar Obligatoria**: Requeridos/Completados/Faltantes/% Avance
- **Disciplinar Optativa**: Requeridos/Completados/Faltantes/% Avance
- **Libre Elección**: Requeridos/Completados/Faltantes/% Avance
- **TOTAL**: Consolidado general

### 6. **Análisis Detallado**
- Porcentaje de materias reconocidas vs. por cursar
- Porcentaje de créditos completados vs. pendientes

## 🎨 Formato Visual

- **Encabezados**: Fondo azul (`#366092`) con texto blanco
- **Materias Aprobadas**: Fondo verde claro (`#E7F7E7`)
- **Materias Pendientes**: Fondo rojo claro (`#FFE7E7`)
- **Bordes**: Grises para mejor legibilidad
- **Columnas ajustadas**: Ancho optimizado para contenido

## 🧪 Cómo Probar

### Usar el Script de Prueba
```bash
./test_excel_cambio_carrera.sh
```

### Petición Manual con curl
```bash
curl -X POST http://localhost:8080/api/cambio-carrera/excel \
  -H "Content-Type: application/json" \
  -d '{
    "academic_history_text": "Tu historia académica aquí...",
    "target_career_code": "ISIS"
  }'
```

### Usar desde JavaScript
```javascript
async function generarExcelCambioCarrera(historiaAcademica, codigoCarrera) {
  const response = await fetch('/api/cambio-carrera/excel', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      academic_history_text: historiaAcademica,
      target_career_code: codigoCarrera
    })
  });

  const result = await response.json();
  
  if (result.success) {
    // Abrir URL de descarga
    window.open(result.download_url, '_blank');
    console.log('Excel generado:', result.filename);
  } else {
    console.error('Error:', result.error);
  }
}
```

## 📁 Archivos Generados

- **Ubicación**: `static/reports/`
- **Nomenclatura**: `Informe_Cambio_Carrera_{CODIGO_CARRERA}_{TIMESTAMP}.xlsx`
- **Ejemplo**: `Informe_Cambio_Carrera_ISIS_20241220_143025.xlsx`

## ⚠️ Consideraciones

### Validaciones
- ✅ Historia académica requerida
- ✅ Código de carrera válido
- ✅ Plan de estudio activo debe existir

### Errores Comunes
- **400**: Datos faltantes o inválidos
- **500**: Error parseando historia académica
- **500**: Error generando archivo Excel

### Rendimiento
- Tiempo de generación: ~2-5 segundos
- Tamaño promedio: 50-100 KB
- Soporte para historias con 100+ materias

## 🔗 Enlaces Relacionados

- **Endpoint de comparación**: `/api/cambio-carrera-texto`
- **Endpoint JSON**: `/api/cambio-carrera`
- **Documentación general**: `CAMBIO_CARRERA_RESUMEN.md`

---

**¡El Excel de cambio de carrera proporciona un análisis visual completo y profesional del progreso académico del estudiante!** 🎓📊 