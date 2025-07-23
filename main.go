package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"

	"olimpo-vicedecanatura/config"
	"olimpo-vicedecanatura/database"
	"olimpo-vicedecanatura/functions"
	"olimpo-vicedecanatura/models"
)

// ensureDirectoryExists creates directory if it doesn't exist
func ensureDirectoryExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, 0755)
	}
	return nil
}

// buildDownloadURL constructs the correct download URL based on environment
func buildDownloadURL(filename string, c *gin.Context) string {
	// Get the host from the request or use default
	host := c.GetHeader("Host")
	scheme := "https" // Default to https for production
	
	// Check if we're running locally
	if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") {
		scheme = "http"
	}
	
	// If no host in header, try to determine from request
	if host == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
		host = c.Request.Host
	}
	
	return fmt.Sprintf("%s://%s/static/reports/%s", scheme, host, filename)
}

type TipologiaAsignatura string

const (
	TipologiaDisciplinarOptativa   TipologiaAsignatura = "DISCIPLINAR OPTATIVA"
	TipologiaFundamentalObligatoria TipologiaAsignatura = "FUND. OBLIGATORIA"
	TipologiaFundamentalOptativa    TipologiaAsignatura = "FUND. OPTATIVA"
	TipologiaDisciplinarObligatoria TipologiaAsignatura = "DISCIPLINAR OBLIGATORIA"
	TipologiaLibreEleccion         TipologiaAsignatura = "LIBRE ELECCIÓN"
	TipologiaTrabajoGrado          TipologiaAsignatura = "TRABAJO DE GRADO"
)

// ValidarTipologia verifica si una tipología es válida     
func ValidarTipologia(tipo string) bool {
	switch TipologiaAsignatura(tipo) {
	case TipologiaDisciplinarOptativa,
		 TipologiaFundamentalObligatoria,
		 TipologiaFundamentalOptativa,
		 TipologiaDisciplinarObligatoria,
		 TipologiaLibreEleccion,
		 TipologiaTrabajoGrado:
		return true
	default:
		return false
	}
}

type HistoriaAcademicaRequest struct {
	Historia string `json:"historia" binding:"required"`
}

type Asignatura struct {
	Nombre      string            `json:"nombre"`
	Codigo      string            `json:"codigo"`
	Creditos    int               `json:"creditos"`
	Tipo        TipologiaAsignatura `json:"tipo"`
	Periodo     string            `json:"periodo"`
	Calificacion float64           `json:"calificacion"`
	Estado      string            `json:"estado"`
}

type ResumenCreditos struct {
	Tipologia  TipologiaAsignatura `json:"tipologia"`
	Exigidos   int                 `json:"exigidos"`
	Aprobados  int                 `json:"aprobados"`
	Pendientes int                 `json:"pendientes"`
	Inscritos  int                 `json:"inscritos"`
	Cursados   int                 `json:"cursados"`
}

type HistoriaAcademicaResponse struct {
	PlanEstudios      string            `json:"plan_estudios"`
	Facultad          string            `json:"facultad"`
	PAPA              float64           `json:"papa"`
	Promedio          float64           `json:"promedio"`
	Asignaturas       []Asignatura      `json:"asignaturas"`
	ResumenCreditos   []ResumenCreditos `json:"resumen_creditos"`
	PorcentajeAvance  float64           `json:"porcentaje_avance"`
}

func main() {
	// Cargar variables de entorno desde .env
	if err := godotenv.Load(); err != nil {
		log.Println("No se pudo cargar el archivo .env (puede que no exista o ya estén las variables en el entorno)")
	}

	// Inicializar la base de datos
	config.InitDB()

	// Verificar la conexión
	sqlDB, err := config.DB.DB()
	if err != nil {
		log.Fatalf("Error obteniendo la conexión SQL: %v", err)
	}
	defer sqlDB.Close()

	// Probar la conexión
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}
	log.Println("✅ Conexión a la base de datos establecida exitosamente")

	// Ejecutar migraciones
	database.RunMigrations(config.DB)
	log.Println("✅ Migraciones ejecutadas exitosamente")

	// Insertar datos iniciales (opcional)
	database.SeedInitialData(config.DB)
	log.Println("✅ Datos iniciales cargados (si era necesario)")

	// Crear directorio para reportes si no existe
	if err := ensureDirectoryExists("static/reports"); err != nil {
		log.Fatalf("Error creando directorio de reportes: %v", err)
	}
	log.Println("✅ Directorio de reportes verificado/creado exitosamente")

	// Configurar CORS y middlewares
    r := gin.Default()
	
	// Servir archivos estáticos para descargas de reportes
	r.Static("/static", "./static")
	
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"https://olimpo.vercel.app",
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:8080",
			"https://olimpo-app-t6qn9.ondigitalocean.app", // Dominio backend DigitalOcean
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Ruta raíz
    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{
			"message": "API de Olimpo Vicedecanatura",
			"status":  "online",
			"db":      "connected",
			"endpoints": []string{
				"GET /api/careers - Obtener todas las carreras",
				"GET /api/careers/:code/study-plans - Obtener planes de estudio de una carrera",
				"GET /api/study-plans/:id - Obtener detalles de un plan de estudio",
				"POST /api/compare - Comparar historia académica con plan de estudio",
				"POST /api/compare-by-career - Comparar por código de carrera",
				"POST /api/api-compare - Comparar historia académica en texto plano",
				"POST /api/cambio-carrera - Comparar para cambio de carrera (JSON)",
				"POST /api/cambio-carrera-texto - Comparar para cambio de carrera desde texto plano",
				"POST /api/cambio-carrera/excel - Generar reporte Excel de cambio de carrera (retorna URL de descarga)",
				"POST /api/doble-titulacion - Simulación de doble titulación",
				"POST /api/doble-titulacion/excel - Generar reporte Excel de doble titulación (retorna URL de descarga)",
				"POST /api/careers - Crear nueva carrera",
				"POST /api/study-plans - Crear nuevo plan de estudio",
				"POST /api/subjects - Crear nueva materia",
				"POST /api/complete-study-plan - Crear plan de estudio completo con materias",
				
				"GET /api/subjects - Obtener todas las asignaturas",
				"GET /api/subjects/:id - Obtener asignatura por ID",
				"PUT /api/subjects/:id/type - Cambiar tipología de asignatura",
				"PUT /api/subjects/:id - Actualizar asignatura completa",
				
				"GET /api/equivalences - Obtener todas las equivalencias",
				"GET /api/careers/:code/equivalences - Obtener equivalencias por carrera",
				"GET /api/equivalences/:id - Obtener equivalencia por ID",
				"POST /api/equivalences - Crear nueva equivalencia",
				"PUT /api/equivalences/:id - Actualizar equivalencia",
				"PUT /api/equivalences/:id/source-subject - Actualizar materia origen",
				"DELETE /api/equivalences/:id - Eliminar equivalencia",
			},
		})
	})


	// API Routes
	api := r.Group("/api")
	{
		// Obtener todas las carreras disponibles
		api.GET("/careers", getCareers)
		
		// Obtener planes de estudio de una carrera específica
		api.GET("/careers/:code/study-plans", getStudyPlansByCareer)
		
		// Obtener detalles de un plan de estudio específico
		api.GET("/study-plans/:id", getStudyPlanDetails)
		
		// Comparar historia académica con plan de estudio
		api.POST("/compare", compareAcademicHistory)
		
		// Endpoint adicional para comparar por código de carrera (más simple)
		api.POST("/compare-by-career", compareByCareerCode)
		
		// Nuevo endpoint para comparar historia académica en texto plano
		api.POST("/api-compare", compareAcademicHistoryFromText)

		// Nuevo endpoint específico para cambio de carrera (aislado de doble titulación)
		api.POST("/cambio-carrera", compareForCareerChange)

		// Nuevo endpoint para cambio de carrera desde texto plano (form-data y JSON)
		api.POST("/cambio-carrera-texto", compareCareerChangeFromText)

		// Nuevo endpoint para generar reporte Excel de cambio de carrera
		api.POST("/cambio-carrera/excel", func(c *gin.Context) {
			var academicHistoryText, targetCareerCode string

			contentType := c.GetHeader("Content-Type")
			if strings.HasPrefix(contentType, "application/json") {
				var req APICompareRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
					return
				}
				academicHistoryText = req.AcademicHistoryText
				targetCareerCode = req.TargetCareerCode
			} else if strings.HasPrefix(contentType, "multipart/form-data") || strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
				academicHistoryText = c.PostForm("academic_history_text")
				targetCareerCode = c.PostForm("target_career_code")
				if academicHistoryText == "" || targetCareerCode == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Faltan campos en el formulario: academic_history_text y target_career_code son requeridos"})
					return
				}
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type no soportado. Usa application/json o form-data."})
				return
			}

			fmt.Printf("[DEBUG EXCEL CAMBIO CARRERA] Iniciando generación de reporte Excel...\n")
			fmt.Printf("[DEBUG EXCEL CAMBIO CARRERA] Carrera objetivo: %s\n", targetCareerCode)

			// Usar exactamente el mismo flujo exitoso que el endpoint regular de cambio de carrera
			cleanedText := preprocessAcademicHistoryText(academicHistoryText)

			parsedSubjects, err := parseAcademicHistoryTextFlexible(cleanedText)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Error parseando historia académica: " + err.Error()})
				return
			}

			// Convertir a formato de entrada de la API
			var subjects []models.SubjectInput
			for _, ps := range parsedSubjects {
				subject := models.SubjectInput{
					Code:     strings.TrimSpace(ps.Code),
					Name:     ps.Name,
					Credits:  ps.Credits,
					Type:     models.TipologiaAsignatura(ps.Type),
					Grade:    ps.Grade,
					Status:   ps.Status,
					Semester: ps.Semester,
				}
				subjects = append(subjects, subject)
			}

			academicHistory := models.AcademicHistoryInput{
				CareerCode: targetCareerCode,
				Subjects:   subjects,
			}

			// Realizar la comparación usando la función específica de cambio de carrera
			result, err := functions.CompareAcademicHistoryForCareerChange(config.DB, academicHistory)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Obtener información del plan de estudio usado
			studyPlan, _ := functions.GetStudyPlanByCareerCode(config.DB, targetCareerCode)

			// Crear el archivo Excel
			f := excelize.NewFile()
			defer func() {
				if err := f.Close(); err != nil {
					fmt.Println(err)
				}
			}()

			// Configurar la hoja principal
			sheetName := "Informe Cambio de Carrera"
			index, err := f.NewSheet(sheetName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando hoja Excel: " + err.Error()})
				return
			}
			f.SetActiveSheet(index)

			// Definir estilos
			headerStyle, _ := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
				Fill: excelize.Fill{Type: "pattern", Color: []string{"366092"}, Pattern: 1},
				Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
				Border: []excelize.Border{
					{Type: "left", Color: "000000", Style: 1},
					{Type: "top", Color: "000000", Style: 1},
					{Type: "bottom", Color: "000000", Style: 1},
					{Type: "right", Color: "000000", Style: 1},
				},
			})

			titleStyle, _ := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true, Size: 16, Color: "366092"},
				Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			})

			subHeaderStyle, _ := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true, Size: 12, Color: "366092"},
				Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
			})

			dataStyle, _ := f.NewStyle(&excelize.Style{
				Border: []excelize.Border{
					{Type: "left", Color: "CCCCCC", Style: 1},
					{Type: "top", Color: "CCCCCC", Style: 1},
					{Type: "bottom", Color: "CCCCCC", Style: 1},
					{Type: "right", Color: "CCCCCC", Style: 1},
				},
				Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
			})

			approvedStyle, _ := f.NewStyle(&excelize.Style{
				Border: []excelize.Border{
					{Type: "left", Color: "CCCCCC", Style: 1},
					{Type: "top", Color: "CCCCCC", Style: 1},
					{Type: "bottom", Color: "CCCCCC", Style: 1},
					{Type: "right", Color: "CCCCCC", Style: 1},
				},
				Fill: excelize.Fill{Type: "pattern", Color: []string{"E7F7E7"}, Pattern: 1},
				Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
			})

			pendingStyle, _ := f.NewStyle(&excelize.Style{
				Border: []excelize.Border{
					{Type: "left", Color: "CCCCCC", Style: 1},
					{Type: "top", Color: "CCCCCC", Style: 1},
					{Type: "bottom", Color: "CCCCCC", Style: 1},
					{Type: "right", Color: "CCCCCC", Style: 1},
				},
				Fill: excelize.Fill{Type: "pattern", Color: []string{"FFE7E7"}, Pattern: 1},
				Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
			})

			// Título principal
			f.SetCellValue(sheetName, "A1", "INFORME TÉCNICO - CAMBIO DE CARRERA")
			f.SetCellStyle(sheetName, "A1", "A1", titleStyle)
			f.MergeCell(sheetName, "A1", "H1")

			// Información general
			row := 3
			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "INFORMACIÓN GENERAL")
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Fecha de generación:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), time.Now().Format("02/01/2006 15:04:05"))
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Carrera objetivo:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), studyPlan.Career.Name)
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Código de carrera:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), targetCareerCode)
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Plan de estudio:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), studyPlan.Version)
			row++

			// Resumen estadístico
			row++
			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "RESUMEN ESTADÍSTICO")
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Materias parseadas:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), len(parsedSubjects))
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Materias homologables:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), len(result.EquivalentSubjects))
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Créditos homologables:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), result.TotalCredits)
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Materias faltantes:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), len(result.MissingSubjects))
			row++

			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Créditos faltantes:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), result.MissingCredits)
			row++

			porcentajeAvance := calculateCompletionPercentage(result.CreditsSummary)
			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Porcentaje de avance:")
			f.SetCellValue(sheetName, "B"+strconv.Itoa(row), fmt.Sprintf("%.2f%%", porcentajeAvance))
			row++

			// Tabla de materias homologables (APROBADAS)
			row += 2
			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "MATERIAS HOMOLOGABLES")
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
			row++

			// Encabezados de la tabla
			headers := []string{"Código", "Nombre Materia", "Créditos", "Tipología", "Estado", "Equivalencia"}
			for col, header := range headers {
				cellName, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetName, cellName, header)
				f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
			}
			row++

			// Datos de materias homologables
			for _, subject := range result.EquivalentSubjects {
				equivalencia := "Directa"
				if subject.Equivalence != nil {
					equivalencia = subject.Equivalence.Notes
				}

				data := []interface{}{
					subject.Code,
					subject.Name,
					subject.Credits,
					string(subject.Type),
					subject.Status,
					equivalencia,
				}

				for col, value := range data {
					cellName, _ := excelize.CoordinatesToCellName(col+1, row)
					f.SetCellValue(sheetName, cellName, value)
					f.SetCellStyle(sheetName, cellName, cellName, approvedStyle)
				}
				row++
			}

			// Tabla de materias faltantes (PENDIENTES)
			row += 2
			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "MATERIAS FALTANTES")
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
			row++

			// Encabezados de la tabla de faltantes
			for col, header := range headers[:5] { // Solo los primeros 5 headers (sin equivalencia)
				cellName, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetName, cellName, header)
				f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
			}
			row++

			// Datos de materias faltantes
			for _, subject := range result.MissingSubjects {
				data := []interface{}{
					subject.Code,
					subject.Name,
					subject.Credits,
					string(subject.Type),
					"PENDIENTE",
				}

				for col, value := range data {
					cellName, _ := excelize.CoordinatesToCellName(col+1, row)
					f.SetCellValue(sheetName, cellName, value)
					f.SetCellStyle(sheetName, cellName, cellName, pendingStyle)
				}
				row++
			}

			// Resumen por tipología
			row += 2
			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "RESUMEN POR TIPOLOGÍA")
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
			row++

			tipologyHeaders := []string{"Tipología", "Requeridos", "Completados", "Faltantes", "% Avance"}
			for col, header := range tipologyHeaders {
				cellName, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetName, cellName, header)
				f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
			}
			row++

			// Datos por tipología
			tipologies := map[string]models.CreditTypeInfo{
				"Fundamental Obligatoria": result.CreditsSummary.FundObligatoria,
				"Fundamental Optativa":    result.CreditsSummary.FundOptativa,
				"Disciplinar Obligatoria": result.CreditsSummary.DisObligatoria,
				"Disciplinar Optativa":    result.CreditsSummary.DisOptativa,
				"Libre Elección":          result.CreditsSummary.Libre,
				"TOTAL":                   result.CreditsSummary.Total,
			}

			for tipology, info := range tipologies {
				percentage := 0.0
				if info.Required > 0 {
					percentage = (float64(info.Completed) / float64(info.Required)) * 100.0
				}

				data := []interface{}{
					tipology,
					info.Required,
					info.Completed,
					info.Missing,
					fmt.Sprintf("%.1f%%", percentage),
				}

				for col, value := range data {
					cellName, _ := excelize.CoordinatesToCellName(col+1, row)
					f.SetCellValue(sheetName, cellName, value)
					if tipology == "TOTAL" {
						f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
					} else {
						f.SetCellStyle(sheetName, cellName, cellName, dataStyle)
					}
				}
				row++
			}

			// Agregar análisis detallado
			row += 2
			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "ANÁLISIS DETALLADO")
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
			row++

			// Análisis por estado
			analisisData := [][]interface{}{
				{"Materias reconocidas:", len(result.EquivalentSubjects), fmt.Sprintf("%.1f%% del total", float64(len(result.EquivalentSubjects))/float64(len(result.EquivalentSubjects)+len(result.MissingSubjects))*100)},
				{"Materias por cursar:", len(result.MissingSubjects), fmt.Sprintf("%.1f%% del total", float64(len(result.MissingSubjects))/float64(len(result.EquivalentSubjects)+len(result.MissingSubjects))*100)},
				{"Créditos reconocidos:", result.TotalCredits, fmt.Sprintf("%.1f%% del total", porcentajeAvance)},
				{"Créditos por cursar:", result.MissingCredits, fmt.Sprintf("%.1f%% del total", 100.0-porcentajeAvance)},
			}

			for _, analisis := range analisisData {
				f.SetCellValue(sheetName, "A"+strconv.Itoa(row), analisis[0])
				f.SetCellValue(sheetName, "B"+strconv.Itoa(row), analisis[1])
				f.SetCellValue(sheetName, "C"+strconv.Itoa(row), analisis[2])
				row++
			}

			// Ajustar ancho de columnas
			f.SetColWidth(sheetName, "A", "A", 25)
			f.SetColWidth(sheetName, "B", "B", 50)
			f.SetColWidth(sheetName, "C", "C", 12)
			f.SetColWidth(sheetName, "D", "D", 25)
			f.SetColWidth(sheetName, "E", "E", 15)
			f.SetColWidth(sheetName, "F", "F", 30)

			// Generar nombre de archivo único
			timestamp := time.Now().Format("20060102_150405")
			filename := fmt.Sprintf("Informe_Cambio_Carrera_%s_%s.xlsx", targetCareerCode, timestamp)
			
			// Asegurar que el directorio existe
			reportsDir := "static/reports"
			if err := ensureDirectoryExists(reportsDir); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando directorio de reportes: " + err.Error()})
				return
			}
			
			filepath := filepath.Join(reportsDir, filename)

			// Guardar el archivo en el servidor
			if err := f.SaveAs(filepath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error guardando archivo Excel: " + err.Error()})
				return
			}

			// Construir URL de descarga usando el helper
			downloadURL := buildDownloadURL(filename, c)

			// Retornar respuesta JSON con URL de descarga
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Reporte Excel de cambio de carrera generado exitosamente",
				"download_url": downloadURL,
				"filename": filename,
				"report_info": gin.H{
					"carrera": studyPlan.Career.Name,
					"codigo_carrera": targetCareerCode,
					"materias_parseadas": len(parsedSubjects),
					"materias_homologables": len(result.EquivalentSubjects),
					"creditos_homologables": result.TotalCredits,
					"materias_faltantes": len(result.MissingSubjects),
					"creditos_faltantes": result.MissingCredits,
					"porcentaje_avance": fmt.Sprintf("%.2f%%", porcentajeAvance),
					"fecha_generacion": time.Now().Format("02/01/2006 15:04:05"),
				},
			})

			fmt.Printf("[DEBUG EXCEL CAMBIO CARRERA] Reporte Excel generado exitosamente: %s\n", filepath)
			fmt.Printf("[DEBUG EXCEL CAMBIO CARRERA] URL de descarga: %s\n", downloadURL)
		})

		//endpoint para crear carrera
		api.POST("/careers", createCareer)
		//endpoint crear plan de estudio vacio
		api.POST("/study-plans", createStudyPlan) 
		//endpoint crear materia
		api.POST("/subjects", createSubject)
		//endpoint crear plan de estudio completo
		api.POST("/complete-study-plan", createCompleteStudyPlan)
		
		// Obtener todas las asignaturas
		api.GET("/subjects", getAllSubjects)
		
		// ===== SUBJECTS CRUD ENDPOINTS =====
		// Obtener asignatura por ID
		api.GET("/subjects/:id", getSubjectByID)
		// Actualizar tipología de asignatura (cambiar tipo)
		api.PUT("/subjects/:id/type", updateSubjectType)
		// Actualizar asignatura completa
		api.PUT("/subjects/:id", updateSubject)
		
		// ===== EQUIVALENCES CRUD ENDPOINTS =====
		// Obtener todas las equivalencias
		api.GET("/equivalences", getEquivalences)
		// Obtener equivalencias por carrera
		api.GET("/careers/:code/equivalences", getEquivalencesByCareer)
		// Obtener equivalencia por ID
		api.GET("/equivalences/:id", getEquivalenceByID)
		// Crear nueva equivalencia
		api.POST("/equivalences", createEquivalence)
		// Actualizar equivalencia
		api.PUT("/equivalences/:id", updateEquivalence)
		// Actualizar materia origen de equivalencia
		api.PUT("/equivalences/:id/source-subject", updateEquivalenceSourceSubject)
		// Eliminar equivalencia
		api.DELETE("/equivalences/:id", deleteEquivalence)
	}

	// Endpoint para doble titulación - REESCRITO PARA USAR LA MISMA LÓGICA EXITOSA DE CAMBIO DE CARRERA
	r.POST("/api/doble-titulacion", func(c *gin.Context) {
		var req models.DobleTitulacionInput
		contentType := c.GetHeader("Content-Type")
		if strings.HasPrefix(contentType, "application/json") {
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
				return
			}
		} else if strings.HasPrefix(contentType, "multipart/form-data") || strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
			req.HistoriaOrigen = c.PostForm("historia_origen")
			req.HistoriaDoble = c.PostForm("historia_doble")
			req.CodigoCarreraObjetivo = c.PostForm("codigo_carrera_objetivo")
			if req.HistoriaOrigen == "" || req.HistoriaDoble == "" || req.CodigoCarreraObjetivo == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Faltan campos en el formulario: historia_origen, historia_doble y codigo_carrera_objetivo son requeridos"})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type no soportado. Usa application/json o form-data."})
			return
		}

		fmt.Printf("[DEBUG DOBLE TITULACION] Iniciando procesamiento...\n")
		fmt.Printf("[DEBUG DOBLE TITULACION] Carrera objetivo: %s\n", req.CodigoCarreraObjetivo)

		// ===== USAR EXACTAMENTE EL MISMO FLUJO EXITOSO DE CAMBIO DE CARRERA =====

		// 1. Preprocesar ambas historias usando el mismo preprocesador exitoso
		cleanedOrigen := preprocessAcademicHistoryText(req.HistoriaOrigen)
		cleanedDoble := preprocessAcademicHistoryText(req.HistoriaDoble)

		// 2. Parsear ambas historias usando el mismo parser exitoso
		parsedOrigen, err := parseAcademicHistoryTextFlexible(cleanedOrigen)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parseando historia_origen: " + err.Error()})
			return
		}
		
		parsedDoble, err := parseAcademicHistoryTextFlexible(cleanedDoble)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parseando historia_doble: " + err.Error()})
			return
		}

		// 3. Convertir ambos resultados parseados a SubjectInput (formato estándar exitoso)
		var materiasOrigen []models.SubjectInput
		var materiasDoble []models.SubjectInput
		
		// Convertir materias de origen
		for _, ps := range parsedOrigen {
			subject := models.SubjectInput{
				Code:     strings.TrimSpace(ps.Code),
				Name:     strings.TrimSpace(ps.Name),
				Credits:  ps.Credits,
				Type:     models.TipologiaAsignatura(ps.Type),
				Grade:    ps.Grade,
				Status:   ps.Status,
				Semester: ps.Semester,
			}
			materiasOrigen = append(materiasOrigen, subject)
			fmt.Printf("[DEBUG DOBLE TITULACION] Materia origen parseada: %s (%s) - %d créditos - Tipo: %s\n", 
				subject.Name, subject.Code, subject.Credits, string(subject.Type))
		}
		
		// Convertir materias de doble titulación
		for _, ps := range parsedDoble {
			subject := models.SubjectInput{
				Code:     strings.TrimSpace(ps.Code),
				Name:     strings.TrimSpace(ps.Name),
				Credits:  ps.Credits,
				Type:     models.TipologiaAsignatura(ps.Type),
				Grade:    ps.Grade,
				Status:   ps.Status,
				Semester: ps.Semester,
			}
			materiasDoble = append(materiasDoble, subject)
			fmt.Printf("[DEBUG DOBLE TITULACION] Materia doble parseada: %s (%s) - %d créditos - Tipo: %s\n", 
				subject.Name, subject.Code, subject.Credits, string(subject.Type))
		}

		fmt.Printf("[DEBUG DOBLE TITULACION] Materias origen parseadas: %d\n", len(materiasOrigen))
		fmt.Printf("[DEBUG DOBLE TITULACION] Materias doble parseadas: %d\n", len(materiasDoble))

		// 4. Usar la nueva función específica para doble titulación que maneja equivalencias correctamente
		result, err := functions.CompareDobleTitulacionCombinada(config.DB, materiasOrigen, materiasDoble, req.CodigoCarreraObjetivo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 5. Obtener información del plan de estudio (igual que cambio de carrera)
		studyPlan, _ := functions.GetStudyPlanByCareerCode(config.DB, req.CodigoCarreraObjetivo)

		// 6. Retornar EXACTAMENTE el mismo formato exitoso que cambio de carrera
		c.JSON(http.StatusOK, gin.H{
			"comparison_resultado": result,
			"study_plan_info": gin.H{
				"id":      studyPlan.ID,
				"version": studyPlan.Version,
				"career":  studyPlan.Career.Name,
			},
			"summary": gin.H{
				"total_subjects_parsed_origen": len(parsedOrigen),
				"total_subjects_parsed_doble":  len(parsedDoble),
				"total_subjects_in_plan":       len(result.EquivalentSubjects) + len(result.MissingSubjects),
				"homologable_subjects":         len(result.EquivalentSubjects),
				"homologable_credits":          result.TotalCredits,
				"missing_subjects":             len(result.MissingSubjects),
				"missing_credits":              result.MissingCredits,
				"homologation_percentage":      calculateCompletionPercentage(result.CreditsSummary),
			},
		})
	})

	// Endpoint para generar reporte Excel de doble titulación
	r.POST("/api/doble-titulacion/excel", func(c *gin.Context) {
		var req models.DobleTitulacionInput
		contentType := c.GetHeader("Content-Type")
		if strings.HasPrefix(contentType, "application/json") {
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
				return
			}
		} else if strings.HasPrefix(contentType, "multipart/form-data") || strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
			req.HistoriaOrigen = c.PostForm("historia_origen")
			req.HistoriaDoble = c.PostForm("historia_doble")
			req.CodigoCarreraObjetivo = c.PostForm("codigo_carrera_objetivo")
			if req.HistoriaOrigen == "" || req.HistoriaDoble == "" || req.CodigoCarreraObjetivo == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Faltan campos en el formulario: historia_origen, historia_doble y codigo_carrera_objetivo son requeridos"})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type no soportado. Usa application/json o form-data."})
			return
		}

		fmt.Printf("[DEBUG EXCEL] Iniciando generación de reporte Excel...\n")

		// Usar exactamente el mismo flujo que el endpoint regular de doble titulación
		cleanedOrigen := preprocessAcademicHistoryText(req.HistoriaOrigen)
		cleanedDoble := preprocessAcademicHistoryText(req.HistoriaDoble)

		parsedOrigen, err := parseAcademicHistoryTextFlexible(cleanedOrigen)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parseando historia_origen: " + err.Error()})
			return
		}
		
		parsedDoble, err := parseAcademicHistoryTextFlexible(cleanedDoble)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parseando historia_doble: " + err.Error()})
			return
		}

		var materiasOrigen []models.SubjectInput
		var materiasDoble []models.SubjectInput
		
		for _, ps := range parsedOrigen {
			subject := models.SubjectInput{
				Code:     strings.TrimSpace(ps.Code),
				Name:     strings.TrimSpace(ps.Name),
				Credits:  ps.Credits,
				Type:     models.TipologiaAsignatura(ps.Type),
				Grade:    ps.Grade,
				Status:   ps.Status,
				Semester: ps.Semester,
			}
			materiasOrigen = append(materiasOrigen, subject)
		}
		
		for _, ps := range parsedDoble {
			subject := models.SubjectInput{
				Code:     strings.TrimSpace(ps.Code),
				Name:     strings.TrimSpace(ps.Name),
				Credits:  ps.Credits,
				Type:     models.TipologiaAsignatura(ps.Type),
				Grade:    ps.Grade,
				Status:   ps.Status,
				Semester: ps.Semester,
			}
			materiasDoble = append(materiasDoble, subject)
		}

		// Obtener los resultados de la comparación
		result, err := functions.CompareDobleTitulacionCombinada(config.DB, materiasOrigen, materiasDoble, req.CodigoCarreraObjetivo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Obtener información del plan de estudio
		studyPlan, _ := functions.GetStudyPlanByCareerCode(config.DB, req.CodigoCarreraObjetivo)

		// Crear el archivo Excel
		f := excelize.NewFile()
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Println(err)
			}
		}()

		// Configurar la hoja principal
		sheetName := "Informe Doble Titulación"
		index, err := f.NewSheet(sheetName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando hoja Excel: " + err.Error()})
			return
		}
		f.SetActiveSheet(index)

		// Definir estilos
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"366092"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
			},
		})

		titleStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 16, Color: "366092"},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		})

		subHeaderStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 12, Color: "366092"},
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		})

		dataStyle, _ := f.NewStyle(&excelize.Style{
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		})

		// Título principal
		f.SetCellValue(sheetName, "A1", "INFORME TÉCNICO - SIMULACIÓN DE DOBLE TITULACIÓN")
		f.SetCellStyle(sheetName, "A1", "A1", titleStyle)
		f.MergeCell(sheetName, "A1", "H1")

		// Información general
		row := 3
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "INFORMACIÓN GENERAL")
		f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Fecha de generación:")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), time.Now().Format("02/01/2006 15:04:05"))
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Carrera objetivo:")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), studyPlan.Career.Name)
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Plan de estudio:")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), studyPlan.Version)
		row++

		// Resumen estadístico
		row++
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "RESUMEN ESTADÍSTICO")
		f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Materias parseadas (origen):")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), len(parsedOrigen))
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Materias parseadas (doble):")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), len(parsedDoble))
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Materias homologables:")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), len(result.EquivalentSubjects))
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Créditos homologables:")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), result.TotalCredits)
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Materias faltantes:")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), len(result.MissingSubjects))
		row++

		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Créditos faltantes:")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), result.MissingCredits)
		row++

		porcentajeHomologacion := calculateCompletionPercentage(result.CreditsSummary)
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "Porcentaje de homologación:")
		f.SetCellValue(sheetName, "B"+strconv.Itoa(row), fmt.Sprintf("%.2f%%", porcentajeHomologacion))
		row++

		// Tabla de materias homologables
		row += 2
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "MATERIAS HOMOLOGABLES")
		f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
		row++

		// Encabezados de la tabla
		headers := []string{"Código", "Nombre Materia", "Créditos", "Tipología", "Estado", "Equivalencia"}
		for col, header := range headers {
			cellName, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheetName, cellName, header)
			f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
		}
		row++

		// Datos de materias homologables
		for _, subject := range result.EquivalentSubjects {
			equivalencia := "Directa"
			if subject.Equivalence != nil {
				equivalencia = subject.Equivalence.Notes
			}

			data := []interface{}{
				subject.Code,
				subject.Name,
				subject.Credits,
				string(subject.Type),
				subject.Status,
				equivalencia,
			}

			for col, value := range data {
				cellName, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetName, cellName, value)
				f.SetCellStyle(sheetName, cellName, cellName, dataStyle)
			}
			row++
		}

		// Tabla de materias faltantes
		row += 2
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "MATERIAS FALTANTES")
		f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
		row++

		// Encabezados de la tabla de faltantes
		for col, header := range headers[:5] { // Solo los primeros 5 headers (sin equivalencia)
			cellName, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheetName, cellName, header)
			f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
		}
		row++

		// Datos de materias faltantes
		for _, subject := range result.MissingSubjects {
			data := []interface{}{
				subject.Code,
				subject.Name,
				subject.Credits,
				string(subject.Type),
				"FALTANTE",
			}

			for col, value := range data {
				cellName, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetName, cellName, value)
				f.SetCellStyle(sheetName, cellName, cellName, dataStyle)
			}
			row++
		}

		// Resumen por tipología
		row += 2
		f.SetCellValue(sheetName, "A"+strconv.Itoa(row), "RESUMEN POR TIPOLOGÍA")
		f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), subHeaderStyle)
		row++

		tipologyHeaders := []string{"Tipología", "Requeridos", "Completados", "Faltantes"}
		for col, header := range tipologyHeaders {
			cellName, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheetName, cellName, header)
			f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
		}
		row++

		// Datos por tipología
		tipologies := map[string]models.CreditTypeInfo{
			"Fundamental Obligatoria": result.CreditsSummary.FundObligatoria,
			"Fundamental Optativa":    result.CreditsSummary.FundOptativa,
			"Disciplinar Obligatoria": result.CreditsSummary.DisObligatoria,
			"Disciplinar Optativa":    result.CreditsSummary.DisOptativa,
			"Libre Elección":          result.CreditsSummary.Libre,
			"TOTAL":                   result.CreditsSummary.Total,
		}

		for tipology, info := range tipologies {
			data := []interface{}{
				tipology,
				info.Required,
				info.Completed,
				info.Missing,
			}

			for col, value := range data {
				cellName, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetName, cellName, value)
				if tipology == "TOTAL" {
					f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
				} else {
					f.SetCellStyle(sheetName, cellName, cellName, dataStyle)
				}
			}
			row++
		}

		// Ajustar ancho de columnas
		f.SetColWidth(sheetName, "A", "A", 25)
		f.SetColWidth(sheetName, "B", "B", 50)
		f.SetColWidth(sheetName, "C", "C", 12)
		f.SetColWidth(sheetName, "D", "D", 25)
		f.SetColWidth(sheetName, "E", "E", 15)
		f.SetColWidth(sheetName, "F", "F", 30)

		// Generar nombre de archivo único
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("Informe_Doble_Titulacion_%s_%s.xlsx", req.CodigoCarreraObjetivo, timestamp)
		
		// Asegurar que el directorio existe
		reportsDir := "static/reports"
		if err := ensureDirectoryExists(reportsDir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando directorio de reportes: " + err.Error()})
			return
		}
		
		filepath := filepath.Join(reportsDir, filename)

		// Guardar el archivo en el servidor
		if err := f.SaveAs(filepath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error guardando archivo Excel: " + err.Error()})
			return
		}

		// Construir URL de descarga usando el helper
		downloadURL := buildDownloadURL(filename, c)

		// Retornar respuesta JSON con URL de descarga
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Reporte Excel generado exitosamente",
			"download_url": downloadURL,
			"filename": filename,
			"report_info": gin.H{
				"carrera": studyPlan.Career.Name,
				"codigo_carrera": req.CodigoCarreraObjetivo,
				"materias_origen": len(parsedOrigen),
				"materias_doble": len(parsedDoble),
				"materias_homologables": len(result.EquivalentSubjects),
				"creditos_homologables": result.TotalCredits,
				"materias_faltantes": len(result.MissingSubjects),
				"creditos_faltantes": result.MissingCredits,
				"porcentaje_homologacion": fmt.Sprintf("%.2f%%", porcentajeHomologacion),
				"fecha_generacion": time.Now().Format("02/01/2006 15:04:05"),
			},
		})

		fmt.Printf("[DEBUG EXCEL] Reporte Excel generado exitosamente: %s\n", filepath)
		fmt.Printf("[DEBUG EXCEL] URL de descarga: %s\n", downloadURL)
	})

	// Ejecutar servidor
	log.Println("🚀 Servidor iniciado en http://localhost:8080")
	if err := r.Run(); err != nil {
		log.Fatalf("Error iniciando el servidor: %v", err)
	}
}

// getCareers obtiene todas las carreras disponibles
func getCareers(c *gin.Context) {
	var careers []models.Career
	if err := config.DB.Find(&careers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo carreras"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"careers": careers,
	})
}

// getStudyPlansByCareer obtiene los planes de estudio de una carrera específica
func getStudyPlansByCareer(c *gin.Context) {
	careerCode := c.Param("code")
	
	var studyPlans []models.StudyPlan
	if err := config.DB.Preload("Career").
		Joins("JOIN careers ON careers.id = study_plans.career_id").
		Where("careers.code = ?", careerCode).
		Find(&studyPlans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo planes de estudio"})
		return
	}
	
	if len(studyPlans) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontraron planes de estudio para esta carrera"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"study_plans": studyPlans,
	})
}

// getStudyPlanDetails obtiene los detalles completos de un plan de estudio
func getStudyPlanDetails(c *gin.Context) {
	studyPlanID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de plan de estudio inválido"})
		return
	}
	
	var studyPlan models.StudyPlan
	if err := config.DB.Preload("Career").Preload("Subjects").
		First(&studyPlan, uint(studyPlanID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan de estudio no encontrado"})
		return
	}
	
	// Calcular estadísticas del plan
	subjectsByType := make(map[string][]models.Subject)
	creditsByType := make(map[string]int)
	
	for _, subject := range studyPlan.Subjects {
		subjectsByType[string(subject.Type)] = append(subjectsByType[string(subject.Type)], subject)
		creditsByType[string(subject.Type)] += subject.Credits
	}
	
	c.JSON(http.StatusOK, gin.H{
		"study_plan":        studyPlan,
		"subjects_by_type":  subjectsByType,
		"credits_by_type":   creditsByType,
		"total_subjects":    len(studyPlan.Subjects),
	})
}

// CompareRequest estructura para la solicitud de comparación
type CompareRequest struct {
	StudyPlanID     uint                        `json:"study_plan_id" binding:"required"`
	AcademicHistory models.AcademicHistoryInput `json:"academic_history" binding:"required"`
}

// compareAcademicHistory compara una historia académica con un plan de estudio específico
func compareAcademicHistory(c *gin.Context) {
	var req CompareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de entrada inválidos: " + err.Error()})
		return
	}
	
	// Realizar la comparación usando la función que creamos
	result, err := functions.CompareAcademicHistoryWithStudyPlan(config.DB, req.AcademicHistory, req.StudyPlanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Obtener información adicional del plan de estudio para el contexto
	var studyPlan models.StudyPlan
	config.DB.Preload("Career").First(&studyPlan, req.StudyPlanID)
	
	c.JSON(http.StatusOK, gin.H{
		"comparison_result": result,
		"study_plan_info": gin.H{
			"id":      studyPlan.ID,
			"version": studyPlan.Version,
			"career":  studyPlan.Career.Name,
		},
		"summary": gin.H{
			"total_subjects_in_plan":     len(result.EquivalentSubjects) + len(result.MissingSubjects),
			"approved_subjects":          len(result.EquivalentSubjects),
			"missing_subjects":           len(result.MissingSubjects),
			"completion_percentage":      calculateCompletionPercentage(result.CreditsSummary),
		},
	})
}

// createCareer creates a new career
func createCareer(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	career, err := functions.CreateCareer(config.DB, req.Name, req.Code, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"career": career})
}

// createStudyPlan creates a new study plan
func createStudyPlan(c *gin.Context) {
	var req struct {
		CareerID                uint   `json:"career_id" binding:"required"`
		Version                 string `json:"version" binding:"required"`
		FundObligatoriaCredits  int    `json:"fund_obligatoria_credits" binding:"required"`
		FundOptativaCredits     int    `json:"fund_optativa_credits" binding:"required"`
		DisObligatoriaCredits   int    `json:"dis_obligatoria_credits" binding:"required"`
		DisOptativaCredits      int    `json:"dis_optativa_credits" binding:"required"`
		LibreCredits            int    `json:"libre_credits" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	studyPlan, err := functions.CreateStudyPlan(
		config.DB,
		req.CareerID,
		req.Version,
		req.FundObligatoriaCredits,
		req.FundOptativaCredits,
		req.DisObligatoriaCredits,
		req.DisOptativaCredits,
		req.LibreCredits,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"study_plan": studyPlan})
}

// createSubject creates a new subject
func createSubject(c *gin.Context) {
	var req struct {
		StudyPlanID uint   `json:"study_plan_id" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Type        string `json:"type" binding:"required"`
		Credits     int    `json:"credits" binding:"required"`
		Description string `json:"description"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	subject, err := functions.CreateSubject(
		config.DB,
		req.StudyPlanID,
		req.Code,
		req.Name,
		req.Type,
		req.Description,
		req.Credits,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"subject": subject})
}

// createCompleteStudyPlan creates a study plan with subjects in one transaction
func createCompleteStudyPlan(c *gin.Context) {
	var req struct {
		CareerID                uint   `json:"career_id" binding:"required"`
		Version                 string `json:"version" binding:"required"`
		FundObligatoriaCredits  int    `json:"fund_obligatoria_credits" binding:"required"`
		FundOptativaCredits     int    `json:"fund_optativa_credits" binding:"required"`
		DisObligatoriaCredits   int    `json:"dis_obligatoria_credits" binding:"required"`
		DisOptativaCredits      int    `json:"dis_optativa_credits" binding:"required"`
		LibreCredits            int    `json:"libre_credits" binding:"required"`
		Subjects                []struct {
			Code        string `json:"code" binding:"required"`
			Name        string `json:"name" binding:"required"`
			Type        string `json:"type" binding:"required"`
			Credits     int    `json:"credits" binding:"required"`
			Description string `json:"description"`
		} `json:"subjects" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	// Convert subjects to the format expected by the function
	subjects := make([]struct {
		Code        string
		Name        string
		Type        string
		Credits     int
		Description string
	}, len(req.Subjects))
	
	for i, s := range req.Subjects {
		subjects[i] = struct {
			Code        string
			Name        string
			Type        string
			Credits     int
			Description string
		}{
			Code:        s.Code,
			Name:        s.Name,
			Type:        s.Type,
			Credits:     s.Credits,
			Description: s.Description,
		}
	}
	
	studyPlan, err := functions.CreateCompleteStudyPlan(
		config.DB,
		req.CareerID,
		req.Version,
		req.FundObligatoriaCredits,
		req.FundOptativaCredits,
		req.DisObligatoriaCredits,
		req.DisOptativaCredits,
		req.LibreCredits,
		subjects,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"study_plan": studyPlan})
}

// compareByCareerCode compara usando el código de carrera (más simple)
func compareByCareerCode(c *gin.Context) {
	var academicHistory models.AcademicHistoryInput
	if err := c.ShouldBindJSON(&academicHistory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de entrada inválidos: " + err.Error()})
		return
	}
	
	// Realizar la comparación usando el código de carrera
	result, err := functions.CompareAcademicHistoryByCareerCode(config.DB, academicHistory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Obtener información del plan de estudio usado
	studyPlan, _ := functions.GetStudyPlanByCareerCode(config.DB, academicHistory.CareerCode)
	
	c.JSON(http.StatusOK, gin.H{
		"comparison_result": result,
		"study_plan_info": gin.H{
			"id":      studyPlan.ID,
			"version": studyPlan.Version,
			"career":  studyPlan.Career.Name,
		},
		"summary": gin.H{
			"total_subjects_in_plan":     len(result.EquivalentSubjects) + len(result.MissingSubjects),
			"approved_subjects":          len(result.EquivalentSubjects),
			"missing_subjects":           len(result.MissingSubjects),
			"completion_percentage":      calculateCompletionPercentage(result.CreditsSummary),
		},
	})
}

// calculateCompletionPercentage calcula el porcentaje de completitud basado en créditos
func calculateCompletionPercentage(summary models.CreditsSummary) float64 {
	if summary.Total.Required == 0 {
		return 0.0
	}
	return (float64(summary.Total.Completed) / float64(summary.Total.Required)) * 100.0
}

// APICompareRequest estructura para la solicitud de comparación desde texto
type APICompareRequest struct {
	AcademicHistoryText string `json:"academic_history_text" binding:"required"`
	TargetCareerCode    string `json:"target_career_code" binding:"required"`
}

// ParsedSubject representa una materia extraída del texto de historia académica
type ParsedSubject struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Credits     int     `json:"credits"`
	Type        string  `json:"type"`
	Grade       float64 `json:"grade"`
	Status      string  `json:"status"`
	Semester    string  `json:"semester"`
}

// Parser alternativo más flexible para historia académica
func parseAcademicHistoryTextFlexible(text string) ([]ParsedSubject, error) {
	fmt.Println("[DEBUG PARSER] Usando parser flexible")
	fmt.Println("=== INICIO DEL TEXTO ===")
	fmt.Println(text)
	fmt.Println("=== FIN DEL TEXTO ===")
	fmt.Printf("[DEBUG PARSER] Longitud del texto: %d caracteres\n", len(text))
	
	lines := strings.Split(text, "\n")
	fmt.Printf("[DEBUG PARSER] Total líneas después de split: %d\n", len(lines))
	
	var subjects []ParsedSubject
	
	// Patrones mejorados
	// Patrón 1: Código entre paréntesis - captura mejor el nombre y código
	codePattern := regexp.MustCompile(`^(.+?)\s*\(([^)]+)\)\s*$`)
	// Patrón 2: Línea que contiene solo créditos (número)
	creditsPattern := regexp.MustCompile(`^\s*(\d+)\s*$`)
	// Patrón 3: Para limpiar nombres que tienen prefijos como "4APROBADA", "7APROBADA", etc. - MEJORADO
	nameCleanPattern := regexp.MustCompile(`^(\d+)?APROBAD?A?(.+)$`)
	// Patrón 4: Para casos más complejos como "7APROBADAEstadística I"
	complexNameCleanPattern := regexp.MustCompile(`^(\d+[A-Z]+)(.+)$`)
	
	var currentSubject *ParsedSubject
	var lineCount int
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Printf("[DEBUG PARSER] Línea %d: vacía, saltando\n", i+1)
			continue
		}
		
		fmt.Printf("[DEBUG PARSER] Procesando línea %d: '%s'\n", i+1, line)
		
		// Si encontramos un código de materia, empezar nueva materia
		if match := codePattern.FindStringSubmatch(line); match != nil {
			if currentSubject != nil {
				// Guardar la materia anterior si existe
				fmt.Printf("[DEBUG PARSER] Guardando materia anterior: %s (%s)\n", currentSubject.Name, currentSubject.Code)
				subjects = append(subjects, *currentSubject)
			}
			
			name := strings.TrimSpace(match[1])
			code := strings.TrimSpace(match[2])
			
			fmt.Printf("[DEBUG PARSER] COINCIDENCIA CÓDIGO: nombre='%s', código='%s'\n", name, code)
			
			// Limpiar el nombre si tiene prefijos problemáticos - LÓGICA MEJORADA
			originalName := name
			if cleanMatch := nameCleanPattern.FindStringSubmatch(name); cleanMatch != nil {
				name = strings.TrimSpace(cleanMatch[2])
				fmt.Printf("[DEBUG PARSER] Nombre limpiado con patrón básico de '%s' a '%s'\n", originalName, name)
			} else if cleanMatch := complexNameCleanPattern.FindStringSubmatch(name); cleanMatch != nil {
				// Verificar si el prefijo contiene "APROBADA" o similar
				prefix := cleanMatch[1]
				if strings.Contains(strings.ToUpper(prefix), "APROBAD") {
					name = strings.TrimSpace(cleanMatch[2])
					fmt.Printf("[DEBUG PARSER] Nombre limpiado con patrón complejo de '%s' a '%s'\n", originalName, name)
				}
			}
			
			currentSubject = &ParsedSubject{
				Code:     code,
				Name:     name,
				Status:   "APROBADA",
				Credits:  0,
				Grade:    0.0, // No procesaremos calificaciones por ahora
				Type:     "",
				Semester: "",
			}
			lineCount = 0
			fmt.Printf("[DEBUG PARSER] Nueva materia creada: %s (%s)\n", name, code)
			continue
		} else {
			fmt.Printf("[DEBUG PARSER] Línea NO coincide con patrón de código\n")
		}
		
		// Si tenemos una materia en progreso, procesar las líneas siguientes
		if currentSubject != nil {
			lineCount++
			fmt.Printf("[DEBUG PARSER] Procesando línea %d para materia '%s'\n", lineCount, currentSubject.Name)
			
			switch lineCount {
			case 1: // Créditos
				if match := creditsPattern.FindStringSubmatch(line); match != nil {
					if credits, err := strconv.Atoi(match[1]); err == nil {
						currentSubject.Credits = credits
						fmt.Printf("[DEBUG PARSER] Créditos asignados: %d\n", credits)
					}
				} else {
					fmt.Printf("[DEBUG PARSER] Línea NO coincide con patrón de créditos: '%s'\n", line)
				}
			case 2: // Tipo
				currentSubject.Type = line
				fmt.Printf("[DEBUG PARSER] Tipo asignado: %s\n", line)
			case 3: // Período
				currentSubject.Semester = line
				fmt.Printf("[DEBUG PARSER] Período asignado: %s\n", line)
				// Después de procesar el período, guardar la materia (saltamos calificación)
				fmt.Printf("[DEBUG PARSER] Guardando materia completa: %s (%s) - %d créditos - %s\n", currentSubject.Name, currentSubject.Code, currentSubject.Credits, currentSubject.Type)
				subjects = append(subjects, *currentSubject)
				currentSubject = nil
				lineCount = 0
			}
		} else {
			fmt.Printf("[DEBUG PARSER] No hay materia actual, línea ignorada\n")
		}
	}
	
	// Guardar la última materia si existe
	if currentSubject != nil {
		fmt.Printf("[DEBUG PARSER] Guardando última materia: %s (%s)\n", currentSubject.Name, currentSubject.Code)
		subjects = append(subjects, *currentSubject)
	}
	
	fmt.Printf("[DEBUG PARSER] === RESULTADO PARSING ===\n")
	fmt.Printf("[DEBUG PARSER] Total materias parseadas: %d\n", len(subjects))
	for i, subject := range subjects {
		fmt.Printf("[DEBUG PARSER] Materia %d: %s (%s) - %d créditos - %s\n", i+1, subject.Name, subject.Code, subject.Credits, subject.Type)
	}
	
	return subjects, nil
}

// mapearTipologiaCompleta convierte las tipologías del texto parseado a TipologiaAsignatura
func mapearTipologiaCompleta(tipo string) models.TipologiaAsignatura {
	tipo = strings.ToUpper(strings.TrimSpace(tipo))
	
	switch {
	case strings.Contains(tipo, "FUNDAMENTACIÓN OBLIGATORIA") || strings.Contains(tipo, "FUND. OBLIGATORIA") || strings.Contains(tipo, "FUND OBLIGATORIA"):
		return models.TipologiaFundamentalObligatoria
	case strings.Contains(tipo, "FUNDAMENTACIÓN OPTATIVA") || strings.Contains(tipo, "FUND. OPTATIVA") || strings.Contains(tipo, "FUND OPTATIVA"):
		return models.TipologiaFundamentalOptativa
	case strings.Contains(tipo, "DISCIPLINAR OBLIGATORIA"):
		return models.TipologiaDisciplinarObligatoria
	case strings.Contains(tipo, "DISCIPLINAR OPTATIVA"):
		return models.TipologiaDisciplinarOptativa
	case strings.Contains(tipo, "LIBRE ELECCIÓN") || strings.Contains(tipo, "LIBRE ELECCION"):
		return models.TipologiaLibreEleccion
	case strings.Contains(tipo, "TRABAJO DE GRADO"):
		return models.TipologiaTrabajoGrado
	case strings.Contains(tipo, "NIVELACIÓN") || strings.Contains(tipo, "NIVELACION"):
		return models.TipologiaLibreEleccion // Mapear nivelación como libre elección
	default:
		// Si no coincide con ninguno, intentar usar el validador existente
		if ValidarTipologia(tipo) {
			return models.TipologiaAsignatura(tipo)
		}
		return models.TipologiaLibreEleccion // Por defecto
	}
}

// Limpieza y normalización del texto de historia académica
func preprocessAcademicHistoryText(raw string) string {
	fmt.Printf("[DEBUG PREPROCESSOR] === INICIANDO PREPROCESAMIENTO ===\n")
	fmt.Printf("[DEBUG PREPROCESSOR] Texto original longitud: %d\n", len(raw))
	
	// NUEVO: Patrón para separar materias cuando están todas en una línea continua
	// Formato: Nombre (Código)CréditosTipoPeriodoCalificaciónAPROBADA
	continuousMateriaPattern := regexp.MustCompile(`([^(]+)\s*\(([^)]+)\)\s*(\d+)\s*(FUND\.|DISCIPLINAR|LIBRE|TRABAJO|NIVELACIÓN)[^A-Z]*([A-Z\s]+?)\s*(\d{4}-\d+S?\s+[^0-9]*)\s*([0-9]*\.?[0-9]*)\s*APROBADA`)
	
	// Si el texto no tiene saltos de línea significativos, intentar separar materias continuas
	if !strings.Contains(raw, "\n") || len(strings.Split(raw, "\n")) < 5 {
		fmt.Printf("[DEBUG PREPROCESSOR] Detectado texto continuo, intentando separar materias...\n")
		
		matches := continuousMateriaPattern.FindAllStringSubmatch(raw, -1)
		if len(matches) > 0 {
			fmt.Printf("[DEBUG PREPROCESSOR] Encontradas %d materias en texto continuo\n", len(matches))
			
			var processedLines []string
			for i, match := range matches {
				if len(match) >= 6 {
					nombre := strings.TrimSpace(match[1])
					codigo := strings.TrimSpace(match[2])
					creditos := strings.TrimSpace(match[3])
					tipoCompleto := strings.TrimSpace(match[4] + " " + match[5])
					periodo := strings.TrimSpace(match[6])
					calificacion := ""
					if len(match) >= 8 && match[7] != "" {
						calificacion = strings.TrimSpace(match[7])
					}
					
					// Crear entrada estructurada para cada materia
					materiaBlock := fmt.Sprintf("%s (%s)\n%s\n%s\n%s", nombre, codigo, creditos, tipoCompleto, periodo)
					if calificacion != "" {
						materiaBlock += "\n" + calificacion
					}
					materiaBlock += "\nAPROBADA\n"
					
					processedLines = append(processedLines, materiaBlock)
					fmt.Printf("[DEBUG PREPROCESSOR] ✓ Materia %d: %s (%s) - %s créditos - %s\n", i+1, nombre, codigo, creditos, tipoCompleto)
				}
			}
			
			if len(processedLines) > 0 {
				result := strings.Join(processedLines, "\n")
				fmt.Printf("[DEBUG PREPROCESSOR] === TEXTO PROCESADO EXITOSAMENTE ===\n")
				fmt.Printf("[DEBUG PREPROCESSOR] Materias extraídas: %d\n", len(processedLines))
				return result
			}
		}
	}
	
	// Si no es texto continuo, usar el procesamiento original
	// Patrón para detectar líneas de materias válidas
	// Formato: Nombre (Código) \t Créditos \t Tipo \t Periodo \t Calificación+APROBADA
	materiaPattern := regexp.MustCompile(`([^(\n\r\t]+)\s*\(([^)]+)\)\s+(\d+)\s+(FUND\.|DISCIPLINAR|LIBRE|TRABAJO|NIVELACIÓN)[^A-Z]*([A-Z\s]+?)\s+(\d{4}-\d+S?\s+[^0-9]*)\s*([0-9]+\.?[0-9]*)?[A-Z]*APROBADA?`)
	
	lines := strings.Split(raw, "\n")
	var validLines []string
	
	fmt.Printf("[DEBUG PREPROCESSOR] Procesando %d líneas...\n", len(lines))
	
	for i, line := range lines {
		// Limpiar la línea
		cleanLine := strings.TrimSpace(line)
		if cleanLine == "" {
			continue
		}
		
		fmt.Printf("[DEBUG PREPROCESSOR] Línea %d: '%s'\n", i+1, cleanLine)
		
		// Buscar coincidencias con el patrón de materia
		if matches := materiaPattern.FindAllStringSubmatch(cleanLine, -1); len(matches) > 0 {
			for _, match := range matches {
				if len(match) >= 6 {
					nombre := strings.TrimSpace(match[1])
					codigo := strings.TrimSpace(match[2])
					creditos := strings.TrimSpace(match[3])
					tipoCompleto := strings.TrimSpace(match[4] + " " + match[5]) // Agregar espacio entre partes del tipo
					calificacion := ""
					if len(match) >= 8 && match[7] != "" {
						calificacion = strings.TrimSpace(match[7])
					}
					
					// Construir línea estructurada
					validLine := fmt.Sprintf("%s (%s)\n%s\n%s", nombre, codigo, creditos, tipoCompleto)
					if calificacion != "" {
						validLine += "\n" + calificacion
					}
					validLine += "\nAPROBADA"
					
					validLines = append(validLines, validLine)
					fmt.Printf("[DEBUG PREPROCESSOR] ✓ Materia extraída: %s (%s) - %s créditos\n", nombre, codigo, creditos)
				}
			}
		} else {
			// Intentar búsqueda más flexible
			simplePattern := regexp.MustCompile(`([^(\n\r\t]+)\s*\(([A-Z0-9\-]+)\)\s+(\d+)\s+(.*?)APROBADA?`)
			if matches := simplePattern.FindStringSubmatch(cleanLine); len(matches) >= 4 {
				nombre := strings.TrimSpace(matches[1])
				codigo := strings.TrimSpace(matches[2])
				creditos := strings.TrimSpace(matches[3])
				resto := strings.TrimSpace(matches[4])
				
				// Dividir el resto para extraer tipo y periodo
				partes := strings.Fields(resto)
				if len(partes) >= 2 {
					// Los primeros campos probablemente sean el tipo
					tipo := ""
					
					// Buscar el tipo (FUND., DISCIPLINAR, etc.)
					for j, parte := range partes {
						if strings.Contains(parte, "FUND") || strings.Contains(parte, "DISCIPLINAR") || strings.Contains(parte, "LIBRE") {
							// Tomar hasta 3 palabras para el tipo
							endIndex := j + 3
							if endIndex > len(partes) {
								endIndex = len(partes)
							}
							tipo = strings.Join(partes[j:endIndex], " ")
							break
						}
					}
					
					if tipo != "" {
						validLine := fmt.Sprintf("%s (%s)\n%s\n%s\nAPROBADA", nombre, codigo, creditos, tipo)
						validLines = append(validLines, validLine)
						fmt.Printf("[DEBUG PREPROCESSOR] ✓ Materia extraída (simple): %s (%s) - %s créditos\n", nombre, codigo, creditos)
					}
				}
			} else {
				fmt.Printf("[DEBUG PREPROCESSOR] ✗ Línea no coincide con patrones\n")
			}
		}
	}
	
	result := strings.Join(validLines, "\n\n")
	
	fmt.Printf("[DEBUG PREPROCESSOR] === RESULTADO ===\n")
	fmt.Printf("[DEBUG PREPROCESSOR] Materias extraídas: %d\n", len(validLines))
	fmt.Printf("[DEBUG PREPROCESSOR] Texto estructurado:\n%s\n", result)
	
	return result
}

// compareAcademicHistoryFromText compara historia académica en texto con el pensum
func compareAcademicHistoryFromText(c *gin.Context) {
	var academicHistoryText, targetCareerCode string

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var req APICompareRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de entrada inválidos: " + err.Error()})
			return
		}
		academicHistoryText = req.AcademicHistoryText
		targetCareerCode = req.TargetCareerCode
	} else if strings.HasPrefix(contentType, "multipart/form-data") || strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		// Leer desde form-data o x-www-form-urlencoded
		academicHistoryText = c.PostForm("academic_history_text")
		targetCareerCode = c.PostForm("target_career_code")
		fmt.Printf("[DEBUG] academic_history_text recibido: '%s'\n", academicHistoryText)
		fmt.Printf("[DEBUG] target_career_code recibido: '%s'\n", targetCareerCode)
		if academicHistoryText == "" || targetCareerCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Faltan campos en el formulario: academic_history_text y target_career_code son requeridos"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type no soportado. Usa application/json o form-data."})
		return
	}

	// Limpieza y normalización del texto
	cleanedText := preprocessAcademicHistoryText(academicHistoryText)

	// Parsear la historia académica del texto limpio
	parsedSubjects, err := parseAcademicHistoryTextFlexible(cleanedText)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error parseando historia académica: " + err.Error()})
		return
	}

	// Convertir a formato de entrada de la API
	var subjects []models.SubjectInput
	for _, ps := range parsedSubjects {
		subject := models.SubjectInput{
			Code:     strings.TrimSpace(ps.Code),
			Name:     ps.Name,
			Credits:  ps.Credits,
			Type:     models.TipologiaAsignatura(ps.Type),
			Grade:    ps.Grade,
			Status:   ps.Status,
			Semester: ps.Semester,
		}
		subjects = append(subjects, subject)
	}
	fmt.Printf("[DEBUG] Subjects parseados para comparar: %+v\n", subjects)

	academicHistory := models.AcademicHistoryInput{
		CareerCode: targetCareerCode,
		Subjects:   subjects,
	}
	fmt.Printf("[DEBUG] DTO enviado a comparación: %+v\n", academicHistory)

	// Realizar la comparación
	result, err := functions.CompareAcademicHistoryByCareerCode(config.DB, academicHistory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Obtener información del plan de estudio usado
	studyPlan, _ := functions.GetStudyPlanByCareerCode(config.DB, targetCareerCode)

	c.JSON(http.StatusOK, gin.H{
		"parsed_subjects": parsedSubjects,
		"comparison_result": result,
		"study_plan_info": gin.H{
			"id":      studyPlan.ID,
			"version": studyPlan.Version,
			"career":  studyPlan.Career.Name,
		},
		"summary": gin.H{
			"total_subjects_parsed":     len(parsedSubjects),
			"total_subjects_in_plan":    len(result.EquivalentSubjects) + len(result.MissingSubjects),
			"approved_subjects":         len(result.EquivalentSubjects),
			"missing_subjects":          len(result.MissingSubjects),
			"completion_percentage":     calculateCompletionPercentage(result.CreditsSummary),
		},
	})
}

// compareForCareerChange compara historia académica para un cambio de carrera
func compareForCareerChange(c *gin.Context) {
	var academicHistory models.AcademicHistoryInput
	if err := c.ShouldBindJSON(&academicHistory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de entrada inválidos: " + err.Error()})
		return
	}

	// Realizar la comparación usando la función de cambio de carrera
	result, err := functions.CompareAcademicHistoryForCareerChange(config.DB, academicHistory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Obtener información del plan de estudio usado
	studyPlan, _ := functions.GetStudyPlanByCareerCode(config.DB, academicHistory.CareerCode)

	c.JSON(http.StatusOK, gin.H{
		"comparison_result": result,
		"study_plan_info": gin.H{
			"id":      studyPlan.ID,
			"version": studyPlan.Version,
			"career":  studyPlan.Career.Name,
		},
		"summary": gin.H{
			"total_subjects_in_plan":     len(result.EquivalentSubjects) + len(result.MissingSubjects),
			"approved_subjects":          len(result.EquivalentSubjects),
			"missing_subjects":           len(result.MissingSubjects),
			"completion_percentage":      calculateCompletionPercentage(result.CreditsSummary),
		},
	})
}

// compareCareerChangeFromText maneja cambio de carrera desde texto plano
func compareCareerChangeFromText(c *gin.Context) {
	var academicHistoryText, targetCareerCode string

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var req APICompareRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de entrada inválidos: " + err.Error()})
			return
		}
		academicHistoryText = req.AcademicHistoryText
		targetCareerCode = req.TargetCareerCode
	} else if strings.HasPrefix(contentType, "multipart/form-data") || strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		// Leer desde form-data o x-www-form-urlencoded
		academicHistoryText = c.PostForm("academic_history_text")
		targetCareerCode = c.PostForm("target_career_code")
		fmt.Printf("[DEBUG CAMBIO CARRERA TEXTO] academic_history_text recibido: '%s'\n", academicHistoryText)
		fmt.Printf("[DEBUG CAMBIO CARRERA TEXTO] target_career_code recibido: '%s'\n", targetCareerCode)
		if academicHistoryText == "" || targetCareerCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Faltan campos en el formulario: academic_history_text y target_career_code son requeridos"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type no soportado. Usa application/json o form-data."})
		return
	}

	// Limpieza y normalización del texto
	cleanedText := preprocessAcademicHistoryText(academicHistoryText)

	// Parsear la historia académica del texto limpio
	parsedSubjects, err := parseAcademicHistoryTextFlexible(cleanedText)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error parseando historia académica: " + err.Error()})
		return
	}

	// Convertir a formato de entrada de la API
	var subjects []models.SubjectInput
	for _, ps := range parsedSubjects {
		subject := models.SubjectInput{
			Code:     strings.TrimSpace(ps.Code),
			Name:     ps.Name,
			Credits:  ps.Credits,
			Type:     models.TipologiaAsignatura(ps.Type),
			Grade:    ps.Grade,
			Status:   ps.Status,
			Semester: ps.Semester,
		}
		subjects = append(subjects, subject)
	}
	fmt.Printf("[DEBUG CAMBIO CARRERA TEXTO] Subjects parseados para comparar: %+v\n", subjects)

	academicHistory := models.AcademicHistoryInput{
		CareerCode: targetCareerCode,
		Subjects:   subjects,
	}
	fmt.Printf("[DEBUG CAMBIO CARRERA TEXTO] DTO enviado a comparación: %+v\n", academicHistory)

	// Realizar la comparación usando la función específica de cambio de carrera
	result, err := functions.CompareAcademicHistoryForCareerChange(config.DB, academicHistory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Obtener información del plan de estudio usado
	studyPlan, _ := functions.GetStudyPlanByCareerCode(config.DB, targetCareerCode)

	c.JSON(http.StatusOK, gin.H{
		"parsed_subjects": parsedSubjects,
		"comparison_result": result,
		"study_plan_info": gin.H{
			"id":      studyPlan.ID,
			"version": studyPlan.Version,
			"career":  studyPlan.Career.Name,
		},
		"summary": gin.H{
			"total_subjects_parsed":     len(parsedSubjects),
			"total_subjects_in_plan":    len(result.EquivalentSubjects) + len(result.MissingSubjects),
			"approved_subjects":         len(result.EquivalentSubjects),
			"missing_subjects":          len(result.MissingSubjects),
			"completion_percentage":     calculateCompletionPercentage(result.CreditsSummary),
		},
	})
}

// getEquivalences obtiene todas las equivalencias
func getEquivalences(c *gin.Context) {
	equivalences, err := functions.GetAllEquivalences(config.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo equivalencias: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"equivalences": equivalences,
	})
}

// getEquivalencesByCareer obtiene equivalencias por carrera
func getEquivalencesByCareer(c *gin.Context) {
	careerCode := c.Param("code")
	
	equivalences, err := functions.GetEquivalencesByCareerCode(config.DB, careerCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo equivalencias: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"equivalences": equivalences,
	})
}

// getEquivalenceByID obtiene una equivalencia por ID
func getEquivalenceByID(c *gin.Context) {
	equivalenceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de equivalencia inválido"})
		return
	}
	
	equivalence, err := functions.GetEquivalenceByID(config.DB, uint(equivalenceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Equivalencia no encontrada: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"equivalence": equivalence,
	})
}

// createEquivalence crea una nueva equivalencia
func createEquivalence(c *gin.Context) {
	var req struct {
		SourceSubject struct {
			Code        string `json:"code" binding:"required"`
			Name        string `json:"name" binding:"required"`
			Type        string `json:"type" binding:"required"`
			Credits     int    `json:"credits" binding:"required"`
			Description string `json:"description"`
		} `json:"source_subject" binding:"required"`
		TargetSubjectID uint   `json:"target_subject_id" binding:"required"`
		CareerID        uint   `json:"career_id" binding:"required"`
		Type            string `json:"type" binding:"required"`
		Notes           string `json:"notes"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	equivalence, err := functions.CreateEquivalence(
		config.DB,
		req.SourceSubject,
		req.TargetSubjectID,
		req.CareerID,
		req.Type,
		req.Notes,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"equivalence": equivalence})
}

// updateEquivalence actualiza una equivalencia
func updateEquivalence(c *gin.Context) {
	equivalenceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de equivalencia inválido"})
		return
	}
	
	var req struct {
		Type            string `json:"type"`
		Notes           string `json:"notes"`
		TargetSubjectID uint   `json:"target_subject_id"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	equivalence, err := functions.UpdateEquivalence(config.DB, uint(equivalenceID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"equivalence": equivalence})
}

// updateEquivalenceSourceSubject actualiza la materia origen de una equivalencia
func updateEquivalenceSourceSubject(c *gin.Context) {
	equivalenceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de equivalencia inválido"})
		return
	}
	
	var req struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Credits     int    `json:"credits"`
		Description string `json:"description"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	equivalence, err := functions.UpdateSourceSubject(config.DB, uint(equivalenceID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"equivalence": equivalence})
}

// deleteEquivalence elimina una equivalencia
func deleteEquivalence(c *gin.Context) {
	equivalenceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de equivalencia inválido"})
		return
	}
	
	if err := functions.DeleteEquivalence(config.DB, uint(equivalenceID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Equivalencia eliminada exitosamente"})
}

// getAllSubjects obtiene todas las asignaturas de la base de datos
func getAllSubjects(c *gin.Context) {
	var subjects []models.Subject
	if err := config.DB.Find(&subjects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo asignaturas: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"subjects": subjects,
	})
}

// getSubjectByID obtiene una asignatura por ID
func getSubjectByID(c *gin.Context) {
	subjectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de asignatura inválido"})
		return
	}
	
	var subject models.Subject
	if err := config.DB.First(&subject, uint(subjectID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asignatura no encontrada: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"subject": subject,
	})
}

// updateSubjectType actualiza el tipo de una asignatura
func updateSubjectType(c *gin.Context) {
	subjectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de asignatura inválido"})
		return
	}
	
	var req struct {
		Type string `json:"type" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	subject, err := functions.UpdateSubjectType(config.DB, uint(subjectID), req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"subject": subject,
	})
}

// updateSubject actualiza una asignatura completa
func updateSubject(c *gin.Context) {
	subjectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de asignatura inválido"})
		return
	}
	
	var req struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Credits     int    `json:"credits"`
		Description string `json:"description"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	
	subject, err := functions.UpdateSubject(config.DB, uint(subjectID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"subject": subject,
		"message": "Asignatura actualizada exitosamente",
	})
}