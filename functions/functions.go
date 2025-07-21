package functions 

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"olimpo-vicedecanatura/models"
	"strings"
	"regexp"
	"strconv"
)

// CompareAcademicHistoryWithStudyPlan compara la historia académica de un estudiante con un plan de estudio
func CompareAcademicHistoryWithStudyPlan(db *gorm.DB, academicHistory models.AcademicHistoryInput, studyPlanID uint) (*models.ComparisonResult, error) {
	// 1. Obtener el plan de estudio con sus materias
	var studyPlan models.StudyPlan
	if err := db.Preload("Subjects").Preload("Career").First(&studyPlan, studyPlanID).Error; err != nil {
		return nil, errors.New("plan de estudio no encontrado")
	}

	// 2. Obtener todas las equivalencias relevantes para las materias del plan
	var studyPlanSubjectIDs []uint
	for _, subject := range studyPlan.Subjects {
		studyPlanSubjectIDs = append(studyPlanSubjectIDs, subject.ID)
	}

	var equivalences []models.Equivalence
	db.Preload("SourceSubject").Preload("TargetSubject").Where(
		"source_subject_id IN ? OR target_subject_id IN ?", 
		studyPlanSubjectIDs, studyPlanSubjectIDs,
	).Find(&equivalences)

	// 3. Crear mapas para facilitar las búsquedas
	studyPlanSubjectsMap := make(map[string]*models.Subject)
	for i := range studyPlan.Subjects {
		studyPlanSubjectsMap[studyPlan.Subjects[i].Code] = &studyPlan.Subjects[i]
	}

	// Crear mapa de equivalencias
	equivalenceMap := make(map[string][]string) // código -> códigos equivalentes
	for _, equiv := range equivalences {
		// Si la materia origen está en el plan, agregar la destino como equivalente
		if _, exists := studyPlanSubjectsMap[equiv.SourceSubject.Code]; exists {
			equivalenceMap[equiv.SourceSubject.Code] = append(equivalenceMap[equiv.SourceSubject.Code], equiv.TargetSubject.Code)
		}
		// Si la materia destino está en el plan, agregar la origen como equivalente
		if _, exists := studyPlanSubjectsMap[equiv.TargetSubject.Code]; exists {
			equivalenceMap[equiv.TargetSubject.Code] = append(equivalenceMap[equiv.TargetSubject.Code], equiv.SourceSubject.Code)
		}
	}

	// 4. Procesar la historia académica
	approvedSubjects := make(map[string]bool) // códigos de materias aprobadas
	for _, historySubject := range academicHistory.Subjects {
		// Asumir que todas las materias en la historia académica están aprobadas
		// ya que están en la historia académica del estudiante
		cleanCode := strings.TrimSpace(historySubject.Code)
		approvedSubjects[cleanCode] = true
		fmt.Printf("[DEBUG COMPARACION] Materia aprobada agregada: %s (%s)\n", historySubject.Name, cleanCode)
	}
	fmt.Printf("[DEBUG COMPARACION] Total materias aprobadas: %d\n", len(approvedSubjects))
	fmt.Printf("[DEBUG COMPARACION] Materias aprobadas en historia académica: %+v\n", approvedSubjects)
	fmt.Printf("[DEBUG COMPARACION] Materias del plan: ")
	for _, planSubject := range studyPlan.Subjects {
		fmt.Printf("%s, ", planSubject.Code)
	}
	fmt.Println()
	fmt.Printf("[DEBUG COMPARACION] Equivalencias cargadas: %+v\n", equivalenceMap)

	// 5. Determinar qué materias del plan están aprobadas (directa o por equivalencia)
	var equivalentSubjects []models.SubjectResult
	var missingSubjects []models.SubjectResult
	
	creditsByType := map[string]int{
		"fund.obligatoria": 0,
		"fund.optativa":    0,
		"dis.obligatoria":  0,
		"dis.optativa":     0,
		"libre":            0,
	}

	for _, planSubject := range studyPlan.Subjects {
		isApproved := false
		var equivalenceInfo *models.EquivalenceResult

		fmt.Printf("[DEBUG COMPARACION] === Verificando materia del plan: %s (%s) ===\n", planSubject.Name, planSubject.Code)
		
		// Verificar si está aprobada directamente
		if approvedSubjects[planSubject.Code] {
			isApproved = true
			fmt.Printf("[DEBUG COMPARACION] ✅ ENCONTRADA directamente: %s (%s)\n", planSubject.Name, planSubject.Code)
		} else {
			fmt.Printf("[DEBUG COMPARACION] ❌ NO encontrada directamente: %s (%s)\n", planSubject.Name, planSubject.Code)
			// Verificar si está aprobada por equivalencia
			if equivalentCodes, hasEquivalences := equivalenceMap[planSubject.Code]; hasEquivalences {
				fmt.Printf("[DEBUG COMPARACION] Verificando equivalencias para %s: %v\n", planSubject.Code, equivalentCodes)
				for _, equivCode := range equivalentCodes {
					if approvedSubjects[equivCode] {
						isApproved = true
						equivalenceInfo = &models.EquivalenceResult{
							Type:  "total", // Asumimos equivalencia total por simplicidad
							Notes: "Aprobada por equivalencia con " + equivCode,
						}
						fmt.Printf("[DEBUG COMPARACION] ✅ ENCONTRADA por equivalencia: %s (%s) → %s\n", planSubject.Name, planSubject.Code, equivCode)
						break
					} else {
						fmt.Printf("[DEBUG COMPARACION] ❌ Equivalencia %s no encontrada en historial\n", equivCode)
					}
				}
			} else {
				fmt.Printf("[DEBUG COMPARACION] ❌ No hay equivalencias definidas para %s\n", planSubject.Code)
			}
		}

		subjectResult := models.SubjectResult{
			Code:        planSubject.Code,
			Name:        planSubject.Name,
			Credits:     planSubject.Credits,
			Type:        planSubject.Type,
			Equivalence: equivalenceInfo,
		}

		if isApproved {
			subjectResult.Status = "APROBADA"
			equivalentSubjects = append(equivalentSubjects, subjectResult)
			
			// CORREGIR EL MAPEO DE TIPOS PARA CONTAR CRÉDITOS CORRECTAMENTE
			typeKey := strings.ToLower(strings.ReplaceAll(string(planSubject.Type), " ", "."))
			if strings.Contains(typeKey, "fund") && strings.Contains(typeKey, "obligatoria") {
				typeKey = "fund.obligatoria"
			} else if strings.Contains(typeKey, "fund") && strings.Contains(typeKey, "optativa") {
				typeKey = "fund.optativa"
			} else if strings.Contains(typeKey, "disciplinar") && strings.Contains(typeKey, "obligatoria") {
				typeKey = "dis.obligatoria"
			} else if strings.Contains(typeKey, "disciplinar") && strings.Contains(typeKey, "optativa") {
				typeKey = "dis.optativa"
			} else if strings.Contains(typeKey, "libre") {
				typeKey = "libre"
			} else {
				typeKey = "libre" // default fallback
			}
			
			creditsByType[typeKey] += planSubject.Credits
		} else {
			subjectResult.Status = "PENDIENTE"
			missingSubjects = append(missingSubjects, subjectResult)
		}
	}

	// 6. Calcular resumen de créditos
	creditsSummary := models.CreditsSummary{
		FundObligatoria: models.CreditTypeInfo{
			Required:  studyPlan.FundObligatoriaCredits,
			Completed: creditsByType["fund.obligatoria"],
			Missing:   studyPlan.FundObligatoriaCredits - creditsByType["fund.obligatoria"],
		},
		FundOptativa: models.CreditTypeInfo{
			Required:  studyPlan.FundOptativaCredits,
			Completed: creditsByType["fund.optativa"],
			Missing:   studyPlan.FundOptativaCredits - creditsByType["fund.optativa"],
		},
		DisObligatoria: models.CreditTypeInfo{
			Required:  studyPlan.DisObligatoriaCredits,
			Completed: creditsByType["dis.obligatoria"],
			Missing:   studyPlan.DisObligatoriaCredits - creditsByType["dis.obligatoria"],
		},
		DisOptativa: models.CreditTypeInfo{
			Required:  studyPlan.DisOptativaCredits,
			Completed: creditsByType["dis.optativa"],
			Missing:   studyPlan.DisOptativaCredits - creditsByType["dis.optativa"],
		},
		Libre: models.CreditTypeInfo{
			Required:  studyPlan.LibreCredits,
			Completed: creditsByType["libre"],
			Missing:   studyPlan.LibreCredits - creditsByType["libre"],
		},
	}

	// Calcular totales
	totalCompleted := creditsByType["fund.obligatoria"] + creditsByType["fund.optativa"] + 
					  creditsByType["dis.obligatoria"] + creditsByType["dis.optativa"] + creditsByType["libre"]
	
	creditsSummary.Total = models.CreditTypeInfo{
		Required:  studyPlan.TotalCredits,
		Completed: totalCompleted,
		Missing:   studyPlan.TotalCredits - totalCompleted,
	}

	// Asegurar que los valores faltantes no sean negativos
	if creditsSummary.FundObligatoria.Missing < 0 {
		creditsSummary.FundObligatoria.Missing = 0
	}
	if creditsSummary.FundOptativa.Missing < 0 {
		creditsSummary.FundOptativa.Missing = 0
	}
	if creditsSummary.DisObligatoria.Missing < 0 {
		creditsSummary.DisObligatoria.Missing = 0
	}
	if creditsSummary.DisOptativa.Missing < 0 {
		creditsSummary.DisOptativa.Missing = 0
	}
	if creditsSummary.Libre.Missing < 0 {
		creditsSummary.Libre.Missing = 0
	}
	if creditsSummary.Total.Missing < 0 {
		creditsSummary.Total.Missing = 0
	}

	return &models.ComparisonResult{
		EquivalentSubjects: equivalentSubjects,
		MissingSubjects:    missingSubjects,
		CreditsSummary:     creditsSummary,
	}, nil
}

// GetStudyPlanByCareerCode obtiene el plan de estudio activo de una carrera por su código
func GetStudyPlanByCareerCode(db *gorm.DB, careerCode string) (*models.StudyPlan, error) {
	var studyPlan models.StudyPlan
	err := db.Preload("Subjects").Preload("Career").
		Joins("JOIN careers ON careers.id = study_plans.career_id").
		Where("careers.code = ? AND study_plans.is_active = ?", careerCode, true).
		First(&studyPlan).Error
	
	if err != nil {
		return nil, errors.New("plan de estudio activo no encontrado para la carrera: " + careerCode)
	}
	
	return &studyPlan, nil
}

// CompareAcademicHistoryByCareerCode compara la historia académica usando el código de carrera
func CompareAcademicHistoryByCareerCode(db *gorm.DB, academicHistory models.AcademicHistoryInput) (*models.ComparisonResult, error) {
	// Obtener el plan de estudio activo de la carrera
	studyPlan, err := GetStudyPlanByCareerCode(db, academicHistory.CareerCode)
	if err != nil {
		return nil, err
	}
	
	// Realizar la comparación
	return CompareAcademicHistoryWithStudyPlan(db, academicHistory, studyPlan.ID)
}

// CompareAcademicHistoryForCareerChange - Nueva función específica para cambio de carrera
// Esta función maneja correctamente las materias de la misma carrera sin necesidad de equivalencias
func CompareAcademicHistoryForCareerChange(db *gorm.DB, academicHistory models.AcademicHistoryInput) (*models.ComparisonResult, error) {
	fmt.Printf("[DEBUG CAMBIO CARRERA] === INICIANDO COMPARACIÓN PARA CAMBIO DE CARRERA ===\n")
	fmt.Printf("[DEBUG CAMBIO CARRERA] Carrera objetivo: %s\n", academicHistory.CareerCode)
	fmt.Printf("[DEBUG CAMBIO CARRERA] Total materias en historia: %d\n", len(academicHistory.Subjects))

	// 1. Obtener el plan de estudio con sus materias
	var studyPlan models.StudyPlan
	err := db.Preload("Subjects").Preload("Career").
		Joins("JOIN careers ON careers.id = study_plans.career_id").
		Where("careers.code = ? AND study_plans.is_active = ?", academicHistory.CareerCode, true).
		First(&studyPlan).Error
	
	if err != nil {
		return nil, errors.New("plan de estudio activo no encontrado para la carrera: " + academicHistory.CareerCode)
	}

	fmt.Printf("[DEBUG CAMBIO CARRERA] Plan de estudio encontrado: %s (ID: %d)\n", studyPlan.Career.Name, studyPlan.ID)
	fmt.Printf("[DEBUG CAMBIO CARRERA] Total materias en plan: %d\n", len(studyPlan.Subjects))

	// 2. Crear mapas para facilitar las búsquedas
	studyPlanSubjectsMap := make(map[string]*models.Subject)
	for i := range studyPlan.Subjects {
		code := strings.ToUpper(strings.TrimSpace(studyPlan.Subjects[i].Code))
		studyPlanSubjectsMap[code] = &studyPlan.Subjects[i]
		fmt.Printf("[DEBUG CAMBIO CARRERA] Materia en plan: %s (%s)\n", studyPlan.Subjects[i].Name, code)
	}

	// 3. Procesar la historia académica - crear mapa de materias aprobadas
	approvedSubjects := make(map[string]models.SubjectInput)
	for _, historySubject := range academicHistory.Subjects {
		cleanCode := strings.ToUpper(strings.TrimSpace(historySubject.Code))
		if cleanCode != "" {
			approvedSubjects[cleanCode] = historySubject
			fmt.Printf("[DEBUG CAMBIO CARRERA] Materia aprobada: %s (%s) - %d créditos - %s\n", 
				historySubject.Name, cleanCode, historySubject.Credits, string(historySubject.Type))
		}
	}

	fmt.Printf("[DEBUG CAMBIO CARRERA] Total materias aprobadas: %d\n", len(approvedSubjects))

	// 4. Obtener equivalencias relevantes para el plan (para casos donde sí existen)
	var studyPlanSubjectIDs []uint
	for _, subject := range studyPlan.Subjects {
		studyPlanSubjectIDs = append(studyPlanSubjectIDs, subject.ID)
	}

	var equivalences []models.Equivalence
	db.Preload("SourceSubject").Preload("TargetSubject").Where(
		"source_subject_id IN ? OR target_subject_id IN ?", 
		studyPlanSubjectIDs, studyPlanSubjectIDs,
	).Find(&equivalences)

	// Crear mapa de equivalencias
	equivalenceMap := make(map[string][]string)
	for _, equiv := range equivalences {
		sourceCode := strings.ToUpper(strings.TrimSpace(equiv.SourceSubject.Code))
		targetCode := strings.ToUpper(strings.TrimSpace(equiv.TargetSubject.Code))
		
		// Si la materia origen está en el plan, agregar la destino como equivalente
		if _, exists := studyPlanSubjectsMap[sourceCode]; exists {
			equivalenceMap[sourceCode] = append(equivalenceMap[sourceCode], targetCode)
		}
		// Si la materia destino está en el plan, agregar la origen como equivalente
		if _, exists := studyPlanSubjectsMap[targetCode]; exists {
			equivalenceMap[targetCode] = append(equivalenceMap[targetCode], sourceCode)
		}
	}

	fmt.Printf("[DEBUG CAMBIO CARRERA] Equivalencias cargadas: %+v\n", equivalenceMap)

	// 5. Determinar qué materias del plan están aprobadas
	var equivalentSubjects []models.SubjectResult
	var missingSubjects []models.SubjectResult
	
	creditsByType := map[string]int{
		"fund.obligatoria": 0,
		"fund.optativa":    0,
		"dis.obligatoria":  0,
		"dis.optativa":     0,
		"libre":            0,
	}

	for _, planSubject := range studyPlan.Subjects {
		planCode := strings.ToUpper(strings.TrimSpace(planSubject.Code))
		isApproved := false
		var equivalenceInfo *models.EquivalenceResult

		fmt.Printf("[DEBUG CAMBIO CARRERA] === Verificando materia del plan: %s (%s) ===\n", planSubject.Name, planCode)
		
		// PRIORIDAD 1: Verificar si está aprobada directamente por código
		if _, found := approvedSubjects[planCode]; found {
			isApproved = true
			fmt.Printf("[DEBUG CAMBIO CARRERA] ✅ ENCONTRADA directamente: %s (%s)\n", planSubject.Name, planCode)
		} else {
			fmt.Printf("[DEBUG CAMBIO CARRERA] ❌ NO encontrada directamente: %s (%s)\n", planSubject.Name, planCode)
			
			// PRIORIDAD 2: Verificar si está aprobada por equivalencia
			if equivalentCodes, hasEquivalences := equivalenceMap[planCode]; hasEquivalences {
				fmt.Printf("[DEBUG CAMBIO CARRERA] Verificando equivalencias para %s: %v\n", planCode, equivalentCodes)
				for _, equivCode := range equivalentCodes {
					equivCodeUpper := strings.ToUpper(strings.TrimSpace(equivCode))
					if historySubject, found := approvedSubjects[equivCodeUpper]; found {
						isApproved = true
						equivalenceInfo = &models.EquivalenceResult{
							Type:  "total",
							Notes: fmt.Sprintf("Aprobada por equivalencia con %s (%s)", historySubject.Name, historySubject.Code),
						}
						fmt.Printf("[DEBUG CAMBIO CARRERA] ✅ ENCONTRADA por equivalencia: %s (%s) → %s (%s)\n", 
							planSubject.Name, planCode, historySubject.Name, equivCodeUpper)
						break
					} else {
						fmt.Printf("[DEBUG CAMBIO CARRERA] ❌ Equivalencia %s no encontrada en historial\n", equivCodeUpper)
					}
				}
			} else {
				fmt.Printf("[DEBUG CAMBIO CARRERA] ❌ No hay equivalencias definidas para %s\n", planCode)
			}
		}

		subjectResult := models.SubjectResult{
			Code:        planSubject.Code,
			Name:        planSubject.Name,
			Credits:     planSubject.Credits,
			Type:        planSubject.Type,
			Equivalence: equivalenceInfo,
		}

		if isApproved {
			subjectResult.Status = "APROBADA"
			equivalentSubjects = append(equivalentSubjects, subjectResult)
			
			// Mapear tipos correctamente para contar créditos
			typeKey := strings.ToLower(strings.ReplaceAll(string(planSubject.Type), " ", "."))
			if strings.Contains(typeKey, "fund") && strings.Contains(typeKey, "obligatoria") {
				typeKey = "fund.obligatoria"
			} else if strings.Contains(typeKey, "fund") && strings.Contains(typeKey, "optativa") {
				typeKey = "fund.optativa"
			} else if strings.Contains(typeKey, "disciplinar") && strings.Contains(typeKey, "obligatoria") {
				typeKey = "dis.obligatoria"
			} else if strings.Contains(typeKey, "disciplinar") && strings.Contains(typeKey, "optativa") {
				typeKey = "dis.optativa"
			} else if strings.Contains(typeKey, "libre") {
				typeKey = "libre"
			} else {
				typeKey = "libre" // default fallback
			}
			
			creditsByType[typeKey] += planSubject.Credits
			fmt.Printf("[DEBUG CAMBIO CARRERA] Créditos agregados para %s: %d (total: %d)\n", typeKey, planSubject.Credits, creditsByType[typeKey])
		} else {
			subjectResult.Status = "PENDIENTE"
			missingSubjects = append(missingSubjects, subjectResult)
			fmt.Printf("[DEBUG CAMBIO CARRERA] ❌ Materia faltante: %s (%s)\n", planSubject.Name, planCode)
		}
	}

	// 6. Calcular resumen de créditos
	creditsSummary := models.CreditsSummary{
		FundObligatoria: models.CreditTypeInfo{
			Required:  studyPlan.FundObligatoriaCredits,
			Completed: creditsByType["fund.obligatoria"],
			Missing:   studyPlan.FundObligatoriaCredits - creditsByType["fund.obligatoria"],
		},
		FundOptativa: models.CreditTypeInfo{
			Required:  studyPlan.FundOptativaCredits,
			Completed: creditsByType["fund.optativa"],
			Missing:   studyPlan.FundOptativaCredits - creditsByType["fund.optativa"],
		},
		DisObligatoria: models.CreditTypeInfo{
			Required:  studyPlan.DisObligatoriaCredits,
			Completed: creditsByType["dis.obligatoria"],
			Missing:   studyPlan.DisObligatoriaCredits - creditsByType["dis.obligatoria"],
		},
		DisOptativa: models.CreditTypeInfo{
			Required:  studyPlan.DisOptativaCredits,
			Completed: creditsByType["dis.optativa"],
			Missing:   studyPlan.DisOptativaCredits - creditsByType["dis.optativa"],
		},
		Libre: models.CreditTypeInfo{
			Required:  studyPlan.LibreCredits,
			Completed: creditsByType["libre"],
			Missing:   studyPlan.LibreCredits - creditsByType["libre"],
		},
	}

	// Calcular totales
	totalCompleted := creditsByType["fund.obligatoria"] + creditsByType["fund.optativa"] + 
					  creditsByType["dis.obligatoria"] + creditsByType["dis.optativa"] + creditsByType["libre"]
	
	creditsSummary.Total = models.CreditTypeInfo{
		Required:  studyPlan.TotalCredits,
		Completed: totalCompleted,
		Missing:   studyPlan.TotalCredits - totalCompleted,
	}

	// Asegurar que los valores faltantes no sean negativos
	if creditsSummary.FundObligatoria.Missing < 0 {
		creditsSummary.FundObligatoria.Missing = 0
	}
	if creditsSummary.FundOptativa.Missing < 0 {
		creditsSummary.FundOptativa.Missing = 0
	}
	if creditsSummary.DisObligatoria.Missing < 0 {
		creditsSummary.DisObligatoria.Missing = 0
	}
	if creditsSummary.DisOptativa.Missing < 0 {
		creditsSummary.DisOptativa.Missing = 0
	}
	if creditsSummary.Libre.Missing < 0 {
		creditsSummary.Libre.Missing = 0
	}
	if creditsSummary.Total.Missing < 0 {
		creditsSummary.Total.Missing = 0
	}

	fmt.Printf("[DEBUG CAMBIO CARRERA] === RESULTADO FINAL ===\n")
	fmt.Printf("[DEBUG CAMBIO CARRERA] Materias equivalentes: %d\n", len(equivalentSubjects))
	fmt.Printf("[DEBUG CAMBIO CARRERA] Materias faltantes: %d\n", len(missingSubjects))
	fmt.Printf("[DEBUG CAMBIO CARRERA] Créditos completados: %d/%d\n", totalCompleted, studyPlan.TotalCredits)

	return &models.ComparisonResult{
		EquivalentSubjects: equivalentSubjects,
		MissingSubjects:    missingSubjects,
		CreditsSummary:     creditsSummary,
		TotalCredits:       totalCompleted,
		MissingCredits:     studyPlan.TotalCredits - totalCompleted,
	}, nil
}

// CreateCareer crea una carrera vacia (Sin planes de estudio)
func CreateCareer(db *gorm.DB, name, code, description string) (*models.Career, error) {
	// Validate required fields
	if name == "" || code == "" {
		return nil, errors.New("name and code are required")
	}

	// Check if career code already exists
	var existingCareer models.Career
	if err := db.Where("code = ?", code).First(&existingCareer).Error; err == nil {
		return nil, errors.New("career with this code already exists")
	}

	// Create new career
	career := models.Career{
		Name:        name,
		Code:        code,
		Description: description,
	}

	if err := db.Create(&career).Error; err != nil {
		return nil, errors.New("failed to create career: " + err.Error())
	}

	return &career, nil
}

// CreateStudyPlan crea un plan de estudio vacio (Sin subjects) y lo asocia a una carrera
func CreateStudyPlan(db *gorm.DB, careerID uint, version string, fundObligatoriaCredits, fundOptativaCredits, disObligatoriaCredits, disOptativaCredits, libreCredits int) (*models.StudyPlan, error) {
	// Validate required fields
	if version == "" {
		return nil, errors.New("version is required")
	}

	// Check if career exists
	var career models.Career
	if err := db.First(&career, careerID).Error; err != nil {
		return nil, errors.New("career not found")
	}

	// Check if study plan version already exists for this career
	var existingPlan models.StudyPlan
	if err := db.Where("career_id = ? AND version = ?", careerID, version).First(&existingPlan).Error; err == nil {
		return nil, errors.New("study plan with this version already exists for this career")
	}

	// Calculate total credits
	totalCredits := fundObligatoriaCredits + fundOptativaCredits + disObligatoriaCredits + disOptativaCredits + libreCredits

	// Create new study plan
	studyPlan := models.StudyPlan{
		CareerID:                careerID,
		Version:                 version,
		IsActive:                true, // New plans are active by default
		FundObligatoriaCredits:  fundObligatoriaCredits,
		FundOptativaCredits:     fundOptativaCredits,
		DisObligatoriaCredits:   disObligatoriaCredits,
		DisOptativaCredits:      disOptativaCredits,
		LibreCredits:            libreCredits,
		TotalCredits:            totalCredits,
	}

	if err := db.Create(&studyPlan).Error; err != nil {
		return nil, errors.New("failed to create study plan: " + err.Error())
	}

	// Load the career relationship
	db.Preload("Career").First(&studyPlan, studyPlan.ID)

	return &studyPlan, nil
}

// CreateSubject crea un nuevo subject y lo asocia a un plan de estudios
func CreateSubject(db *gorm.DB, studyPlanID uint, code, name, subjectType, description string, credits int) (*models.Subject, error) {
	// Validate required fields
	if code == "" || name == "" || subjectType == "" {
		return nil, errors.New("code, name, and type are required")
	}

	// Validate subject type using the model's validation function
	if !models.ValidarTipologia(subjectType) {
		return nil, errors.New("invalid subject type. Must be one of: FUND. OBLIGATORIA, FUND. OPTATIVA, DISCIPLINAR OBLIGATORIA, DISCIPLINAR OPTATIVA, LIBRE ELECCIÓN, TRABAJO DE GRADO")
	}

	// Validate credits
	if credits <= 0 {
		return nil, errors.New("credits must be greater than 0")
	}

	// Check if study plan exists
	var studyPlan models.StudyPlan
	if err := db.First(&studyPlan, studyPlanID).Error; err != nil {
		return nil, errors.New("study plan not found")
	}

	// Check if subject code already exists
	var existingSubject models.Subject
	if err := db.Where("code = ?", code).First(&existingSubject).Error; err == nil {
		return nil, errors.New("subject with this code already exists")
	}

	// Create new subject
	subject := models.Subject{
		Code:        code,
		Name:        name,
		Credits:     credits,
		Type:        models.TipologiaAsignatura(subjectType),
		Description: description,
	}

	if err := db.Create(&subject).Error; err != nil {
		return nil, errors.New("failed to create subject: " + err.Error())
	}

	// Associate subject with study plan (many-to-many relationship)
	if err := db.Model(&studyPlan).Association("Subjects").Append(&subject); err != nil {
		return nil, errors.New("failed to associate subject with study plan: " + err.Error())
	}

	return &subject, nil
}

// Helper function to create a complete study plan with subjects in one go
func CreateCompleteStudyPlan(db *gorm.DB, careerID uint, version string, fundObligatoriaCredits, fundOptativaCredits, disObligatoriaCredits, disOptativaCredits, libreCredits int, subjects []struct {
	Code        string
	Name        string
	Type        string
	Credits     int
	Description string
}) (*models.StudyPlan, error) {
	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create study plan
	studyPlan, err := CreateStudyPlan(tx, careerID, version, fundObligatoriaCredits, fundOptativaCredits, disObligatoriaCredits, disOptativaCredits, libreCredits)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Create and associate subjects
	for _, subjectData := range subjects {
		_, err := CreateSubject(tx, studyPlan.ID, subjectData.Code, subjectData.Name, subjectData.Type, subjectData.Description, subjectData.Credits)
		if err != nil {
			tx.Rollback()
			return nil, errors.New("failed to create subject " + subjectData.Code + ": " + err.Error())
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, errors.New("failed to commit transaction: " + err.Error())
	}

	// Reload study plan with subjects
	db.Preload("Career").Preload("Subjects").First(studyPlan, studyPlan.ID)

	return studyPlan, nil
}

// ===== CRUD FUNCTIONS FOR EQUIVALENCES =====

// CreateEquivalence crea una nueva equivalencia entre materias
func CreateEquivalence(db *gorm.DB, sourceSubjectData struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Credits     int    `json:"credits" binding:"required"`
	Description string `json:"description"`
}, targetSubjectID uint, careerID uint, equivalenceType, notes string) (*models.Equivalence, error) {
	// Validar campos requeridos
	if sourceSubjectData.Code == "" || sourceSubjectData.Name == "" || sourceSubjectData.Type == "" {
		return nil, errors.New("code, name, and type are required for source subject")
	}
	if targetSubjectID == 0 {
		return nil, errors.New("target subject ID is required")
	}
	if careerID == 0 {
		return nil, errors.New("career ID is required")
	}
	if equivalenceType == "" {
		return nil, errors.New("equivalence type is required")
	}

	// Validar que la carrera existe
	var career models.Career
	if err := db.First(&career, careerID).Error; err != nil {
		return nil, errors.New("career not found")
	}

	// Validar que la materia destino existe
	var targetSubject models.Subject
	if err := db.First(&targetSubject, targetSubjectID).Error; err != nil {
		return nil, errors.New("target subject not found")
	}

	// Validar tipo de materia origen
	if !models.ValidarTipologia(sourceSubjectData.Type) {
		return nil, errors.New("invalid source subject type. Must be one of: FUND. OBLIGATORIA, FUND. OPTATIVA, DISCIPLINAR OBLIGATORIA, DISCIPLINAR OPTATIVA, LIBRE ELECCIÓN, TRABAJO DE GRADO")
	}

	// Validar créditos
	if sourceSubjectData.Credits <= 0 {
		return nil, errors.New("credits must be greater than 0")
	}

	// Buscar si ya existe la materia de origen por código
	var sourceSubject models.Subject
	err := db.Where("code = ?", sourceSubjectData.Code).First(&sourceSubject).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No existe, crearla
			sourceSubject = models.Subject{
				Code:        sourceSubjectData.Code,
				Name:        sourceSubjectData.Name,
				Credits:     sourceSubjectData.Credits,
				Type:        models.TipologiaAsignatura(sourceSubjectData.Type),
				Description: sourceSubjectData.Description,
			}
			if err := db.Create(&sourceSubject).Error; err != nil {
				return nil, errors.New("failed to create source subject: " + err.Error())
			}
		} else {
			return nil, errors.New("failed to check source subject: " + err.Error())
		}
	}
	// Si ya existe, simplemente la reutilizamos (no la actualizamos aquí)

	// Crear la equivalencia
	equivalence := models.Equivalence{
		SourceSubjectID: sourceSubject.ID,
		TargetSubjectID: targetSubjectID,
		Type:            equivalenceType,
		Notes:           notes,
		CareerID:        careerID,
	}

	if err := db.Create(&equivalence).Error; err != nil {
		return nil, errors.New("failed to create equivalence: " + err.Error())
	}

	// Cargar las relaciones
	db.Preload("SourceSubject").Preload("TargetSubject").Preload("Career").First(&equivalence, equivalence.ID)

	return &equivalence, nil
}

// GetEquivalenceByID obtiene una equivalencia por su ID
func GetEquivalenceByID(db *gorm.DB, equivalenceID uint) (*models.Equivalence, error) {
	var equivalence models.Equivalence
	if err := db.Preload("SourceSubject").Preload("TargetSubject").Preload("Career").
		First(&equivalence, equivalenceID).Error; err != nil {
		return nil, errors.New("equivalence not found")
	}
	return &equivalence, nil
}

// GetAllEquivalences obtiene todas las equivalencias
func GetAllEquivalences(db *gorm.DB) ([]models.Equivalence, error) {
	var equivalences []models.Equivalence
	if err := db.Preload("SourceSubject").Preload("TargetSubject").Preload("Career").
		Find(&equivalences).Error; err != nil {
		return nil, errors.New("failed to fetch equivalences: " + err.Error())
	}
	return equivalences, nil
}

// GetEquivalencesByCareer obtiene todas las equivalencias de una carrera específica
func GetEquivalencesByCareer(db *gorm.DB, careerID uint) ([]models.Equivalence, error) {
	var equivalences []models.Equivalence
	if err := db.Preload("SourceSubject").Preload("TargetSubject").Preload("Career").
		Where("career_id = ?", careerID).Find(&equivalences).Error; err != nil {
		return nil, errors.New("failed to fetch equivalences for career: " + err.Error())
	}
	return equivalences, nil
}

// GetEquivalencesByCareerCode obtiene todas las equivalencias de una carrera por su código
func GetEquivalencesByCareerCode(db *gorm.DB, careerCode string) ([]models.Equivalence, error) {
	var equivalences []models.Equivalence
	if err := db.Preload("SourceSubject").Preload("TargetSubject").Preload("Career").
		Joins("JOIN careers ON careers.id = equivalences.career_id").
		Where("careers.code = ?", careerCode).Find(&equivalences).Error; err != nil {
		return nil, errors.New("failed to fetch equivalences for career code: " + err.Error())
	}
	return equivalences, nil
}

// UpdateEquivalence actualiza una equivalencia existente
func UpdateEquivalence(db *gorm.DB, equivalenceID uint, updates struct {
	Type            string `json:"type"`
	Notes           string `json:"notes"`
	TargetSubjectID uint   `json:"target_subject_id"`
}) (*models.Equivalence, error) {
	// Verificar que la equivalencia existe
	var equivalence models.Equivalence
	if err := db.First(&equivalence, equivalenceID).Error; err != nil {
		return nil, errors.New("equivalence not found")
	}

	// Actualizar campos
	if updates.Type != "" {
		equivalence.Type = updates.Type
	}
	if updates.Notes != "" {
		equivalence.Notes = updates.Notes
	}
	if updates.TargetSubjectID != 0 {
		// Validar que la materia destino existe
		var targetSubject models.Subject
		if err := db.First(&targetSubject, updates.TargetSubjectID).Error; err != nil {
			return nil, errors.New("target subject not found")
		}
		equivalence.TargetSubjectID = updates.TargetSubjectID
	}

	if err := db.Save(&equivalence).Error; err != nil {
		return nil, errors.New("failed to update equivalence: " + err.Error())
	}

	// Cargar las relaciones
	db.Preload("SourceSubject").Preload("TargetSubject").Preload("Career").First(&equivalence, equivalence.ID)

	return &equivalence, nil
}

// UpdateSourceSubject actualiza la materia origen de una equivalencia
func UpdateSourceSubject(db *gorm.DB, equivalenceID uint, updates struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Credits     int    `json:"credits"`
	Description string `json:"description"`
}) (*models.Equivalence, error) {
	// Verificar que la equivalencia existe y obtener el ID de la materia origen
	var equivalence models.Equivalence
	if err := db.First(&equivalence, equivalenceID).Error; err != nil {
		return nil, errors.New("equivalence not found")
	}

	// Buscar la materia origen por ID
	var subject models.Subject
	if err := db.First(&subject, equivalence.SourceSubjectID).Error; err != nil {
		return nil, errors.New("source subject not found")
	}

	// Validar y actualizar campos
	updateFields := make(map[string]interface{})
	if updates.Code != "" {
		updateFields["code"] = updates.Code
	}
	if updates.Name != "" {
		updateFields["name"] = updates.Name
	}
	if updates.Type != "" {
		if !models.ValidarTipologia(updates.Type) {
			return nil, errors.New("invalid subject type")
		}
		updateFields["type"] = models.TipologiaAsignatura(updates.Type)
	}
	if updates.Credits > 0 {
		updateFields["credits"] = updates.Credits
	}
	if updates.Description != "" {
		updateFields["description"] = updates.Description
	}

	if len(updateFields) > 0 {
		if err := db.Model(&subject).Updates(updateFields).Error; err != nil {
			return nil, errors.New("failed to update source subject: " + err.Error())
		}
	}

	// Recargar equivalence con la materia actualizada
	db.Preload("SourceSubject").Preload("TargetSubject").Preload("Career").First(&equivalence, equivalence.ID)
	return &equivalence, nil
}

// DeleteEquivalence elimina una equivalencia (pero NO elimina la materia de origen)
func DeleteEquivalence(db *gorm.DB, equivalenceID uint) error {
	// Verificar que la equivalencia existe
	var equivalence models.Equivalence
	if err := db.First(&equivalence, equivalenceID).Error; err != nil {
		return errors.New("equivalence not found")
	}

	// Eliminar solo la equivalencia
	if err := db.Delete(&equivalence).Error; err != nil {
		return errors.New("failed to delete equivalence: " + err.Error())
	}

	return nil
}

// GetEquivalencesBySubject obtiene todas las equivalencias donde una materia específica aparece
func GetEquivalencesBySubject(db *gorm.DB, subjectID uint) ([]models.Equivalence, error) {
	var equivalences []models.Equivalence
	if err := db.Preload("SourceSubject").Preload("TargetSubject").Preload("Career").
		Where("source_subject_id = ? OR target_subject_id = ?", subjectID, subjectID).
		Find(&equivalences).Error; err != nil {
		return nil, errors.New("failed to fetch equivalences for subject: " + err.Error())
	}
	return equivalences, nil
}

// ===== FUNCIONES PARA DOBLE TITULACIÓN =====

// CompareDobleTitulacion compara dos historias académicas para determinar materias homologables
// Ahora recibe el código de la carrera objetivo y busca el plan activo
func CompareDobleTitulacion(db *gorm.DB, historiaOrigen, historiaDoble, codigoCarreraObjetivo string) (*models.DobleTitulacionResult, error) {
	// 1. Obtener el plan de estudio activo de la carrera objetivo
	planObjetivo, err := GetStudyPlanByCareerCode(db, codigoCarreraObjetivo)
	if err != nil {
		return nil, errors.New("plan de estudio objetivo no encontrado para la carrera: " + codigoCarreraObjetivo)
	}

	// 2. Procesar ambas historias académicas
	materiasOrigen := procesarHistoriaAcademicaTexto(historiaOrigen)
	materiasDoble := procesarHistoriaAcademicaTexto(historiaDoble)

	// 3. Obtener equivalencias relevantes para el plan objetivo
	var equivalencias []models.Equivalence
	db.Preload("SourceSubject").Preload("TargetSubject").Where("career_id = ?", planObjetivo.CareerID).Find(&equivalencias)

	// Crear mapa de equivalencias: código origen -> código objetivo
	equivalenciaMap := make(map[string]string)
	for _, equiv := range equivalencias {
		equivalenciaMap[equiv.SourceSubject.Code] = equiv.TargetSubject.Code
	}

	// 4. Crear mapas de materias cursadas para búsqueda rápida
	materiasCursadasOrigen := make(map[string]models.SubjectInput)
	for _, materia := range materiasOrigen {
		materiasCursadasOrigen[materia.Code] = materia
	}

	materiasCursadasDoble := make(map[string]models.SubjectInput)
	for _, materia := range materiasDoble {
		materiasCursadasDoble[materia.Code] = materia
	}

	// 4.5. Crear mapa para controlar créditos por componente académico
	creditosPorComponente := make(map[string]int)

	// 5. ONLY iterate through subjects that are actually in the study plan
	var materiasHomologables []models.MateriaHomologable
	var warnings []models.Warning
	totalCreditos := 0

	for _, materiaPlan := range planObjetivo.Subjects {
		// Check if this study plan subject can be homologated
		var materiaOrigen *models.SubjectInput
		var codigoOrigen string
		var nombreOrigen string
		var tipologiaOrigen string
		var equivalenciaInfo *models.EquivalenceResult

		// First, check direct match in origen
		if materia, existe := materiasCursadasOrigen[materiaPlan.Code]; existe {
			materiaOrigen = &materia
			codigoOrigen = materia.Code
			nombreOrigen = materia.Name
			tipologiaOrigen = string(materia.Type)
		} else {
			// Check for equivalence: find if any subject in origen is equivalent to this study plan subject
			for codigoOrig, codigoObj := range equivalenciaMap {
				if codigoObj == materiaPlan.Code { // This origen subject is equivalent to our study plan subject
					if materia, existe := materiasCursadasOrigen[codigoOrig]; existe {
						materiaOrigen = &materia
						codigoOrigen = materia.Code
						nombreOrigen = materia.Name
						tipologiaOrigen = string(materia.Type)
						equivalenciaInfo = &models.EquivalenceResult{
							Type:  "TOTAL",
							Notes: fmt.Sprintf("Equivalencia: %s → %s", codigoOrig, materiaPlan.Code),
						}
						break
					}
				}
			}
		}

		// If we found the subject in origen (directly or through equivalence)
		if materiaOrigen != nil {
			// Check if it's already completed in double degree
			yaCompletadaEnDoble := false
			
			// Direct check
			if _, existe := materiasCursadasDoble[materiaPlan.Code]; existe {
				yaCompletadaEnDoble = true
			} else {
				// Check through equivalence
				for codigoOrig, codigoObj := range equivalenciaMap {
					if codigoObj == materiaPlan.Code {
						if _, existe := materiasCursadasDoble[codigoOrig]; existe {
							yaCompletadaEnDoble = true
							break
						}
					}
				}
			}

			if !yaCompletadaEnDoble {
				// Subject is in origen but not in double degree - generate warning
				warnings = append(warnings, models.Warning{
					Type:        "MATERIA_REQUIERE_HOMOLOGACION",
					Message:     fmt.Sprintf("La materia '%s' (%s) está en la historia origen pero no en la doble titulación, requiere homologación", materiaPlan.Name, materiaPlan.Code),
					SubjectCode: materiaPlan.Code,
					SubjectName: materiaPlan.Name,
				})
			} else {
				// Check if the academic component already has enough credits
				if materiaPlan.ComponenteAcademico != "" {
					creditosActuales := creditosPorComponente[materiaPlan.ComponenteAcademico]
					creditosRequeridos := getCreditosRequeridosPorComponente(planObjetivo, materiaPlan.ComponenteAcademico)
					
					if creditosActuales >= creditosRequeridos {
						// Component already has enough credits, don't homologate
						continue
					}
				}

				// This study plan subject can be homologated
				materiaHomologable := models.MateriaHomologable{
					CodigoObjetivo:    materiaPlan.Code,        // Always the study plan subject code
					NombreObjetivo:    materiaPlan.Name,        // Always the study plan subject name
					Creditos:          materiaPlan.Credits,     // Always the study plan subject credits
					TipologiaObjetivo: materiaPlan.Type,        // Always the study plan subject type
					CodigoOrigen:      codigoOrigen,            // The origen subject that fulfills it
					NombreOrigen:      nombreOrigen,            // The origen subject name
					TipologiaOrigen:   tipologiaOrigen,         // The origen subject type
					Equivalencia:      equivalenciaInfo,        // Equivalence info if applicable
				}

				materiasHomologables = append(materiasHomologables, materiaHomologable)
				totalCreditos += materiaPlan.Credits // Use study plan credits, not origen credits

				// Update component credits
				if materiaPlan.ComponenteAcademico != "" {
					creditosPorComponente[materiaPlan.ComponenteAcademico] += materiaPlan.Credits
				}
			}
		}
		// If subject is not found in origen (directly or through equivalence), it cannot be homologated
	}

	// 6. Calculate summary
	resumen := models.ResumenDobleTitulacion{
		MateriasCursadasOrigen: len(materiasOrigen),
		MateriasCursadasDoble:  len(materiasDoble),
		MateriasHomologables:   len(materiasHomologables),
		CreditosHomologables:   totalCreditos,
	}

	// Calculate homologation percentage
	if planObjetivo.TotalCredits > 0 {
		resumen.PorcentajeHomologacion = float64(totalCreditos) / float64(planObjetivo.TotalCredits) * 100
	}

	return &models.DobleTitulacionResult{
		MateriasHomologables: materiasHomologables,
		TotalMaterias:        len(materiasHomologables),
		TotalCreditos:        totalCreditos,
		Resumen:              resumen,
		Warnings:             warnings,
	}, nil
}

// Función auxiliar para obtener créditos requeridos por componente
func getCreditosRequeridosPorComponente(plan *models.StudyPlan, componente string) int {
	// Implementar según tu modelo de datos
	totalCreditos := 0
	for _, subject := range plan.Subjects {
		if subject.ComponenteAcademico == componente {
			totalCreditos += subject.Credits
		}
	}
	return totalCreditos
}

// CompareDobleTitulacionParsed compara dos listas de materias ya parseadas para doble titulación
func CompareDobleTitulacionParsed(db *gorm.DB, materiasOrigen, materiasDoble []models.SubjectInput, codigoCarreraObjetivo string) (*models.DobleTitulacionResult, error) {
	// 1. Obtener el plan de estudio activo de la carrera objetivo
	planObjetivo, err := GetStudyPlanByCareerCode(db, codigoCarreraObjetivo)
	if err != nil {
		return nil, errors.New("plan de estudio objetivo no encontrado para la carrera: " + codigoCarreraObjetivo)
	}

	// 2. Obtener equivalencias relevantes para el plan objetivo
	var equivalencias []models.Equivalence
	db.Preload("SourceSubject").Preload("TargetSubject").Where("career_id = ?", planObjetivo.CareerID).Find(&equivalencias)

	// Crear mapa de equivalencias para búsqueda rápida
	equivalenciaMap := make(map[string]string) // código origen -> código objetivo
	for _, equiv := range equivalencias {
		equivalenciaMap[equiv.SourceSubject.Code] = equiv.TargetSubject.Code
	}

	// 3. Crear mapas de materias cursadas para búsqueda rápida
	materiasCursadasOrigen := make(map[string]models.SubjectInput)
	for _, materia := range materiasOrigen {
		materiasCursadasOrigen[materia.Code] = materia
	}

	materiasCursadasDoble := make(map[string]models.SubjectInput)
	for _, materia := range materiasDoble {
		materiasCursadasDoble[materia.Code] = materia
	}

	// 4. Comparar materias del plan objetivo con la historia de origen
	var materiasHomologables []models.MateriaHomologable
	totalCreditos := 0

	for _, materiaPlan := range planObjetivo.Subjects {
		// Buscar si la materia está en la historia de origen (directa o por equivalencia)
		var materiaOrigen *models.SubjectInput
		var codigoOrigen string
		var nombreOrigen string
		var tipologiaOrigen string
		var equivalenciaInfo *models.EquivalenceResult

		// Verificar coincidencia directa
		if materia, existe := materiasCursadasOrigen[materiaPlan.Code]; existe {
			materiaOrigen = &materia
			codigoOrigen = materia.Code
			nombreOrigen = materia.Name
			tipologiaOrigen = string(materia.Type)
		} else {
			// Verificar por equivalencia
			for codigoOrig, codigoObj := range equivalenciaMap {
				if codigoObj == materiaPlan.Code {
					if materia, existe := materiasCursadasOrigen[codigoOrig]; existe {
						materiaOrigen = &materia
						codigoOrigen = materia.Code
						nombreOrigen = materia.Name
						tipologiaOrigen = string(materia.Type)
						equivalenciaInfo = &models.EquivalenceResult{
							Type:  "TOTAL",
							Notes: "Equivalencia: " + codigoOrig + " → " + materiaPlan.Code,
						}
						break
					}
				}
			}
		}

		// Si encontramos la materia en origen y NO está en la historia de doble titulación
		if materiaOrigen != nil {
			if _, yaCursadaEnDoble := materiasCursadasDoble[materiaPlan.Code]; !yaCursadaEnDoble {
				// Limpiar completamente el nombre origen de todos los prefijos problemáticos
				nombreOrigenLimpio := nombreOrigen
				// Múltiples patrones para limpiar nombres problemáticos más comprehensivamente
				nombreOrigenLimpio = regexp.MustCompile(`^\d+APROBAD?A?`).ReplaceAllString(nombreOrigenLimpio, "")
				nombreOrigenLimpio = regexp.MustCompile(`^APROBAD?A?`).ReplaceAllString(nombreOrigenLimpio, "")
				// Patrón adicional para manejar casos como "7APROBADAEstadística I"
				nombreOrigenLimpio = regexp.MustCompile(`^\d+[A-Z]+`).ReplaceAllStringFunc(nombreOrigenLimpio, func(match string) string {
					// Si el match contiene APROBADA o similar, eliminarlo completamente
					if strings.Contains(strings.ToUpper(match), "APROBAD") {
						return ""
					}
					return match
				})
				nombreOrigenLimpio = strings.TrimSpace(nombreOrigenLimpio)
				
				// Si después de limpiar el nombre queda vacío, usar el nombre del plan objetivo
				if nombreOrigenLimpio == "" {
					nombreOrigenLimpio = materiaPlan.Name
				}
				
				materiaHomologable := models.MateriaHomologable{
					CodigoObjetivo:    materiaPlan.Code,
					NombreObjetivo:    materiaPlan.Name,
					Creditos:          materiaPlan.Credits,
					TipologiaObjetivo: materiaPlan.Type,
					CodigoOrigen:      codigoOrigen,
					NombreOrigen:      nombreOrigenLimpio,
					TipologiaOrigen:   tipologiaOrigen,
					Equivalencia:      equivalenciaInfo,
				}

				materiasHomologables = append(materiasHomologables, materiaHomologable)
				totalCreditos += materiaPlan.Credits
			}
		}
	}

	// 5. Calcular resumen
	resumen := models.ResumenDobleTitulacion{
		MateriasCursadasOrigen: len(materiasOrigen),
		MateriasCursadasDoble:  len(materiasDoble),
		MateriasHomologables:   len(materiasHomologables),
		CreditosHomologables:   totalCreditos,
	}

	// Calcular porcentaje de homologación
	if planObjetivo.TotalCredits > 0 {
		resumen.PorcentajeHomologacion = float64(totalCreditos) / float64(planObjetivo.TotalCredits) * 100
	}

	return &models.DobleTitulacionResult{
		MateriasHomologables: materiasHomologables,
		TotalMaterias:        len(materiasHomologables),
		TotalCreditos:        totalCreditos,
		Resumen:              resumen,
	}, nil
}

// CompareDobleTitulacionCombinada compara dos historias académicas combinadas con un plan destino
// USA EXACTAMENTE LA MISMA LÓGICA EXITOSA QUE CAMBIO DE CARRERA
func CompareDobleTitulacionCombinada(db *gorm.DB, materiasOrigen, materiasDoble []models.SubjectInput, codigoCarreraObjetivo string) (*models.ComparisonResult, error) {
	fmt.Printf("[DEBUG DOBLE TITULACION] === INICIANDO LÓGICA SIMPLE ===\n")
	fmt.Printf("[DEBUG DOBLE TITULACION] Materias origen: %d\n", len(materiasOrigen))
	fmt.Printf("[DEBUG DOBLE TITULACION] Materias doble: %d\n", len(materiasDoble))
	
	// 1. COMBINAR AMBAS HISTORIAS ACADÉMICAS SIN DUPLICADOS
	materiasMap := make(map[string]models.SubjectInput)
	
	// Agregar materias de origen
	for _, materia := range materiasOrigen {
		key := strings.ToUpper(strings.TrimSpace(materia.Code))
		if key != "" {
			materiasMap[key] = materia
			fmt.Printf("[DEBUG DOBLE TITULACION] Agregada origen: %s - %s\n", materia.Code, materia.Name)
		}
	}
	
	// Agregar materias de doble (sin duplicar)
	for _, materia := range materiasDoble {
		key := strings.ToUpper(strings.TrimSpace(materia.Code))
		if key != "" {
			if _, exists := materiasMap[key]; !exists {
				materiasMap[key] = materia
				fmt.Printf("[DEBUG DOBLE TITULACION] Agregada doble: %s - %s\n", materia.Code, materia.Name)
			} else {
				fmt.Printf("[DEBUG DOBLE TITULACION] Duplicada ignorada: %s - %s\n", materia.Code, materia.Name)
			}
		}
	}
	
	// 2. CONVERTIR A SLICE COMBINADO
	var materiasCombinadas []models.SubjectInput
	for _, materia := range materiasMap {
		materiasCombinadas = append(materiasCombinadas, materia)
	}
	
	fmt.Printf("[DEBUG DOBLE TITULACION] Total materias combinadas: %d\n", len(materiasCombinadas))
	
	// 3. CREAR EL MISMO DTO QUE USA CAMBIO DE CARRERA
	historiaAcademicaCombinada := models.AcademicHistoryInput{
		CareerCode: codigoCarreraObjetivo,
		Subjects:   materiasCombinadas,
	}
	
	// 4. USAR EXACTAMENTE LA MISMA FUNCIÓN EXITOSA DE CAMBIO DE CARRERA
	fmt.Printf("[DEBUG DOBLE TITULACION] Llamando a CompareAcademicHistoryByCareerCode...\n")
	resultado, err := CompareAcademicHistoryByCareerCode(db, historiaAcademicaCombinada)
	if err != nil {
		fmt.Printf("[DEBUG DOBLE TITULACION] Error en comparación: %v\n", err)
		return nil, err
	}
	
	fmt.Printf("[DEBUG DOBLE TITULACION] === RESULTADO EXITOSO ===\n")
	fmt.Printf("[DEBUG DOBLE TITULACION] Materias equivalentes: %d\n", len(resultado.EquivalentSubjects))
	fmt.Printf("[DEBUG DOBLE TITULACION] Materias faltantes: %d\n", len(resultado.MissingSubjects))
	
	return resultado, nil
}

// procesarHistoriaAcademicaTexto procesa el texto de historia académica y retorna una lista de materias
func procesarHistoriaAcademicaTexto(texto string) []models.SubjectInput {
	var materias []models.SubjectInput
	
	// Dividir por líneas
	lineas := strings.Split(texto, "\n")
	
	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}
		
		// Procesar línea (asumiendo formato tabulado como en el proyecto de referencia)
		partes := strings.Split(linea, "\t")
		if len(partes) >= 5 {
			nombreCompleto := partes[0]
			
			// Extraer código del formato "Nombre (CÓDIGO)"
			codigo := ""
			nombre := nombreCompleto
			if match := regexp.MustCompile(`(.+)\s\((\d{6,}-?[A-Za-z]?)\)`).FindStringSubmatch(nombreCompleto); match != nil {
				nombre = strings.TrimSpace(match[1])
				codigo = strings.TrimSpace(match[2])
			}
			
			// Extraer créditos
			creditos := 0
			if creditosStr := strings.TrimSpace(partes[1]); creditosStr != "" {
				if c, err := strconv.Atoi(creditosStr); err == nil {
					creditos = c
				}
			}
			
			// Extraer tipo/tipología
			tipo := strings.TrimSpace(partes[2])
			
			// Extraer periodo
			periodo := strings.TrimSpace(partes[3])
			
			// Extraer calificación
			calificacion := 0.0
			if calStr := strings.TrimSpace(partes[4]); calStr != "" {
				if cal, err := strconv.ParseFloat(calStr, 64); err == nil {
					calificacion = cal
				}
			}
			
			// Mapear tipología
			tipologia := mapearTipologia(tipo)
			
			materia := models.SubjectInput{
				Code:     codigo,
				Name:     nombre,
				Credits:  creditos,
				Type:     tipologia,
				Grade:    calificacion,
				Status:   "APROBADA", // Asumimos que todas las materias en la historia están aprobadas
				Semester: periodo,
			}
			
			materias = append(materias, materia)
		}
	}
	
	return materias
}

// mapearTipologia convierte las tipologías del texto parseado a TipologiaAsignatura
func mapearTipologia(tipo string) models.TipologiaAsignatura {
	// Usar la función existente simple que funciona
	tipo = strings.TrimSpace(strings.ToUpper(tipo))
	
	switch {
	case strings.Contains(tipo, "FUND") && strings.Contains(tipo, "OBLIGATORIA"):
		return models.TipologiaFundamentalObligatoria
	case strings.Contains(tipo, "FUND") && strings.Contains(tipo, "OPTATIVA"):
		return models.TipologiaFundamentalOptativa
	case strings.Contains(tipo, "DISCIPLINAR") && strings.Contains(tipo, "OBLIGATORIA"):
		return models.TipologiaDisciplinarObligatoria
	case strings.Contains(tipo, "DISCIPLINAR") && strings.Contains(tipo, "OPTATIVA"):
		return models.TipologiaDisciplinarOptativa
	case strings.Contains(tipo, "LIBRE"):
		return models.TipologiaLibreEleccion
	case strings.Contains(tipo, "TRABAJO DE GRADO"):
		return models.TipologiaTrabajoGrado
	default:
		return models.TipologiaLibreEleccion
	}
}
