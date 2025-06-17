package functions 

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"olimpo-vicedecanatura/models"
	"strings"
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
		approvedSubjects[strings.TrimSpace(historySubject.Code)] = true
	}
	fmt.Printf("[DEBUG] Materias aprobadas en historia académica: %+v\n", approvedSubjects)
	fmt.Printf("[DEBUG] Materias del plan: ")
	for _, planSubject := range studyPlan.Subjects {
		fmt.Printf("%s, ", planSubject.Code)
	}
	fmt.Println()
	fmt.Printf("[DEBUG] Equivalencias cargadas: %+v\n", equivalenceMap)

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

		// Verificar si está aprobada directamente
		if approvedSubjects[planSubject.Code] {
			isApproved = true
		} else {
			// Verificar si está aprobada por equivalencia
			if equivalentCodes, hasEquivalences := equivalenceMap[planSubject.Code]; hasEquivalences {
				for _, equivCode := range equivalentCodes {
					if approvedSubjects[equivCode] {
						isApproved = true
						equivalenceInfo = &models.EquivalenceResult{
							Type:  "total", // Asumimos equivalencia total por simplicidad
							Notes: "Aprobada por equivalencia con " + equivCode,
						}
						break
					}
				}
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
			creditsByType[string(planSubject.Type)] += planSubject.Credits
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
