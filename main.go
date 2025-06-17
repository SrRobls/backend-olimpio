package main

import (
	"log"
	"net/http"
	"strconv"
	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	"olimpo-vicedecanatura/config"
	"olimpo-vicedecanatura/database"
	"olimpo-vicedecanatura/models"
	"olimpo-vicedecanatura/functions"
	"strings"
	"regexp"
	"fmt"
)


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

	// Configurar CORS y middlewares
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

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
	}


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
	fmt.Println("[DEBUG] Usando parser flexible")
	fmt.Println("=== INICIO DEL TEXTO ===")
	fmt.Println(text)
	fmt.Println("=== FIN DEL TEXTO ===")
	
	lines := strings.Split(text, "\n")
	var subjects []ParsedSubject
	
	// Buscar patrones de materias en el texto
	// Patrón 1: Código entre paréntesis al inicio de línea
	codePattern := regexp.MustCompile(`^([^(]+)\s*\(([^)]+)\)`)
	// Patrón 2: Línea que contiene créditos (número)
	creditsPattern := regexp.MustCompile(`^\s*(\d+)\s*$`)
	// Patrón 3: Línea que contiene calificación (número decimal)
	gradePattern := regexp.MustCompile(`^\s*(\d+\.?\d*)\s*$`)
	
	var currentSubject *ParsedSubject
	var lineCount int
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		fmt.Printf("[DEBUG] Procesando línea %d: '%s'\n", i+1, line)
		
		// Si encontramos un código de materia, empezar nueva materia
		if match := codePattern.FindStringSubmatch(line); match != nil {
			if currentSubject != nil {
				// Guardar la materia anterior si existe
				subjects = append(subjects, *currentSubject)
			}
			
			name := strings.TrimSpace(match[1])
			code := strings.TrimSpace(match[2])
			
			currentSubject = &ParsedSubject{
				Code:     code,
				Name:     name,
				Status:   "APROBADA",
				Credits:  0,
				Grade:    0.0,
				Type:     "",
				Semester: "",
			}
			lineCount = 0
			fmt.Printf("[DEBUG] Nueva materia encontrada: %s (%s)\n", name, code)
			continue
		}
		
		// Si tenemos una materia en progreso, procesar las líneas siguientes
		if currentSubject != nil {
			lineCount++
			
			switch lineCount {
			case 1: // Créditos
				if match := creditsPattern.FindStringSubmatch(line); match != nil {
					if credits, err := strconv.Atoi(match[1]); err == nil {
						currentSubject.Credits = credits
						fmt.Printf("[DEBUG] Créditos: %d\n", credits)
					}
				}
			case 2: // Tipo
				currentSubject.Type = line
				fmt.Printf("[DEBUG] Tipo: %s\n", line)
			case 3: // Período
				currentSubject.Semester = line
				fmt.Printf("[DEBUG] Período: %s\n", line)
			case 4: // Calificación
				if match := gradePattern.FindStringSubmatch(line); match != nil {
					if grade, err := strconv.ParseFloat(match[1], 64); err == nil {
						currentSubject.Grade = grade
						fmt.Printf("[DEBUG] Calificación: %.1f\n", grade)
					}
				}
				// Después de procesar la calificación, guardar la materia
				subjects = append(subjects, *currentSubject)
				currentSubject = nil
				lineCount = 0
			}
		}
	}
	
	// Guardar la última materia si existe
	if currentSubject != nil {
		subjects = append(subjects, *currentSubject)
	}
	
	fmt.Printf("[DEBUG] Total materias parseadas (flexible): %d\n", len(subjects))
	return subjects, nil
}

// Limpieza y normalización del texto de historia académica
func preprocessAcademicHistoryText(raw string) string {
	// 1. Reemplazar saltos de línea de Windows por Unix
	cleaned := strings.ReplaceAll(raw, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	// 2. Insertar salto de línea antes de cada materia (NOMBRE (CÓDIGO))
	// Esto detecta patrones como: Nombre de materia (código)
	cleaned = regexp.MustCompile(`([A-Za-zÁÉÍÓÚÑáéíóúüÜ0-9\- ]+\([0-9A-Z\-]+\))`).ReplaceAllString(cleaned, "\n$1")

	// 3. Insertar salto de línea antes de cada número de créditos (1 o 2 dígitos)
	cleaned = regexp.MustCompile(`([A-Za-zÁÉÍÓÚÑáéíóúüÜ)]+)(\d{1,2})F`).ReplaceAllString(cleaned, "$1\n$2F")
	// Y también antes de cada número de créditos suelto
	cleaned = regexp.MustCompile(`([A-Za-zÁÉÍÓÚÑáéíóúüÜ)]+)(\d{1,2})\b`).ReplaceAllString(cleaned, "$1\n$2")

	// 4. Insertar salto de línea antes de cada tipo de materia
	cleaned = regexp.MustCompile(`(\d{1,2})((FUND\. OBLIGATORIA|FUND\. OPTATIVA|DISCIPLINAR OBLIGATORIA|DISCIPLINAR OPTATIVA|LIBRE ELECCIÓN|NIVELACIÓN|TRABAJO DE GRADO))`).ReplaceAllString(cleaned, "$1\n$2")

	// 5. Insertar salto de línea antes de cada periodo (año-semestre)
	cleaned = regexp.MustCompile(`(OBLIGATORIA|OPTATIVA|ELECCIÓN|NIVELACIÓN|GRADO)(\d{4}-\dS|\d{4}-\dS|\d{4}-\d{1,2}S|\d{4}-\d{1,2})`).ReplaceAllString(cleaned, "$1\n$2")

	// 6. Insertar salto de línea antes de cada calificación (número decimal)
	cleaned = regexp.MustCompile(`(\d{4}-\dS|\d{4}-\d{1,2}S|\d{4}-\d{1,2})( Ordinaria)?([0-9]\.[0-9])`).ReplaceAllString(cleaned, "$1$2\n$3")

	// 7. Insertar salto de línea antes de cada "APROBADA" o "APROBAD" (por si hay variantes)
	cleaned = regexp.MustCompile(`([0-9]\.[0-9])((APROBADA|APROBAD))`).ReplaceAllString(cleaned, "$1\n$2")

	// 8. Reemplazar múltiples saltos de línea por uno solo
	cleaned = regexp.MustCompile(`\n+`).ReplaceAllString(cleaned, "\n")
	// 9. Quitar espacios en blanco al inicio y final de cada línea
	lines := strings.Split(cleaned, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	cleaned = strings.Join(lines, "\n")
	// 10. Quitar espacios en blanco al inicio y final del texto
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
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
