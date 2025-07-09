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
	"github.com/gin-contrib/cors"
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
				"POST /api/careers - Crear nueva carrera",
				"POST /api/study-plans - Crear nuevo plan de estudio",
				"POST /api/subjects - Crear nueva materia",
				"POST /api/complete-study-plan - Crear plan completo con materias",
				
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
