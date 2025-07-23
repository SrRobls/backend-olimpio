package functions

import (
    "errors"
    "fmt"
    "gorm.io/gorm"
    "olimpo-vicedecanatura/models"
    "strings"
    "regexp"
)

// Helper function for max
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

// Helper function to map subject type to component key
func mapSubjectTypeToComponentKey(subjectType models.TipologiaAsignatura) string {
    switch subjectType {
    case models.TipologiaFundamentalObligatoria:
        return "fund.obligatoria"
    case models.TipologiaFundamentalOptativa:
        return "fund.optativa"
    case models.TipologiaDisciplinarObligatoria:
        return "dis.obligatoria"
    case models.TipologiaDisciplinarOptativa:
        return "dis.optativa"
    case models.TipologiaLibreEleccion:
        return "libre"
    case models.TipologiaTrabajoGrado:
        return "libre"
    default:
        return "libre"
    }
}

func mapSubjectTypeToKey(subjectType models.TipologiaAsignatura) string {
    return mapSubjectTypeToComponentKey(subjectType) // Same logic
}

// GetStudyPlanByCareerCode obtiene el plan de estudio activo de una carrera por su código
func GetStudyPlanByCareerCode(db *gorm.DB, careerCode string) (*models.StudyPlan, error) {
    var studyPlan models.StudyPlan
    err := db.Preload("Subjects").Preload("Career").
        Joins("JOIN careers ON careers.id = study_plans.career_id").
        Where("careers.code = ? AND study_plans.is_active = ?", careerCode, true).
        First(&studyPlan).Error
        
    if err != nil {
        return nil, err
    }
    
    return &studyPlan, nil
}

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
    equivalenceMap := make(map[string][]string)
    for _, equiv := range equivalences {
        if _, exists := studyPlanSubjectsMap[equiv.SourceSubject.Code]; exists {
            equivalenceMap[equiv.SourceSubject.Code] = append(equivalenceMap[equiv.SourceSubject.Code], equiv.TargetSubject.Code)
        }
        if _, exists := studyPlanSubjectsMap[equiv.TargetSubject.Code]; exists {
            equivalenceMap[equiv.TargetSubject.Code] = append(equivalenceMap[equiv.TargetSubject.Code], equiv.SourceSubject.Code)
        }
    }

    // 4. Procesar la historia académica
    approvedSubjects := make(map[string]bool)
    for _, historySubject := range academicHistory.Subjects {
        cleanCode := strings.TrimSpace(historySubject.Code)
        approvedSubjects[cleanCode] = true
    }

    // 5. Define credit requirements for ALL types from study plan
    creditosRequeridos := map[string]int{
        "fund.obligatoria": studyPlan.FundObligatoriaCredits,
        "fund.optativa":    studyPlan.FundOptativaCredits,
        "dis.obligatoria":  studyPlan.DisObligatoriaCredits,
        "dis.optativa":     studyPlan.DisOptativaCredits,
        "libre":            studyPlan.LibreCredits,
    }

    // 6. Group subjects by component type
    subjectsByComponent := make(map[string][]models.Subject)
    for _, subject := range studyPlan.Subjects {
        componentKey := mapSubjectTypeToKey(subject.Type)
        subjectsByComponent[componentKey] = append(subjectsByComponent[componentKey], subject)
    }

    var equivalentSubjects []models.SubjectResult
    var missingSubjects []models.SubjectResult
    var warnings []models.Warning
    
    creditsByType := map[string]int{
        "fund.obligatoria": 0,
        "fund.optativa":    0,
        "dis.obligatoria":  0,
        "dis.optativa":     0,
        "libre":            0,
    }

    // 7. Process each component type with credit limits
    for componentKey, subjects := range subjectsByComponent {
        creditosRequeridosComponente := creditosRequeridos[componentKey]
        creditosCompletadosComponente := 0
        
        for _, planSubject := range subjects {
            // STOP if component requirement is already fulfilled
            if creditosCompletadosComponente >= creditosRequeridosComponente {
                break
            }
            
            isApproved := false
            var equivalenceInfo *models.EquivalenceResult
            
            // Check direct approval
            if approvedSubjects[planSubject.Code] {
                isApproved = true
            } else {
                // Check approval through equivalence
                if equivalentCodes, hasEquivalences := equivalenceMap[planSubject.Code]; hasEquivalences {
                    for _, equivCode := range equivalentCodes {
                        if approvedSubjects[equivCode] {
                            isApproved = true
                            equivalenceInfo = &models.EquivalenceResult{
                                Type:  "total",
                                Notes: "Aprobada por equivalencia con " + equivCode,
                            }
                            break
                        }
                    }
                }
            }

            // Create subject result
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
                
                // Only count credits up to the requirement
                creditsToAdd := planSubject.Credits
                if creditosCompletadosComponente + creditsToAdd > creditosRequeridosComponente {
                    creditsToAdd = creditosRequeridosComponente - creditosCompletadosComponente
                }
                
                creditsByType[componentKey] += creditsToAdd
                creditosCompletadosComponente += creditsToAdd
            } else {
                // Only add to missing if we haven't met the requirement yet
                if creditosCompletadosComponente < creditosRequeridosComponente {
                    subjectResult.Status = "PENDIENTE"
                    missingSubjects = append(missingSubjects, subjectResult)
                    
                    // Add warning for missing obligatory subjects
                    if strings.Contains(componentKey, "obligatoria") {
                        warnings = append(warnings, models.Warning{
                            Type:        "MATERIA_OBLIGATORIA_PENDIENTE",
                            Message:     fmt.Sprintf("La materia obligatoria '%s' (%s) está pendiente", planSubject.Name, planSubject.Code),
                            SubjectCode: planSubject.Code,
                            SubjectName: planSubject.Name,
                        })
                    }
                }
            }
        }
    }

    // 8. Calculate credits summary based on REQUIRED credits
    creditsSummary := models.CreditsSummary{
        FundObligatoria: models.CreditTypeInfo{
            Required:  studyPlan.FundObligatoriaCredits,
            Completed: creditsByType["fund.obligatoria"],
            Missing:   max(0, studyPlan.FundObligatoriaCredits - creditsByType["fund.obligatoria"]),
        },
        FundOptativa: models.CreditTypeInfo{
            Required:  studyPlan.FundOptativaCredits,
            Completed: creditsByType["fund.optativa"],
            Missing:   max(0, studyPlan.FundOptativaCredits - creditsByType["fund.optativa"]),
        },
        DisObligatoria: models.CreditTypeInfo{
            Required:  studyPlan.DisObligatoriaCredits,
            Completed: creditsByType["dis.obligatoria"],
            Missing:   max(0, studyPlan.DisObligatoriaCredits - creditsByType["dis.obligatoria"]),
        },
        DisOptativa: models.CreditTypeInfo{
            Required:  studyPlan.DisOptativaCredits,
            Completed: creditsByType["dis.optativa"],
            Missing:   max(0, studyPlan.DisOptativaCredits - creditsByType["dis.optativa"]),
        },
        Libre: models.CreditTypeInfo{
            Required:  studyPlan.LibreCredits,
            Completed: creditsByType["libre"],
            Missing:   max(0, studyPlan.LibreCredits - creditsByType["libre"]),
        },
    }

    // Calculate totals based on REQUIRED credits
    totalCompleted := creditsByType["fund.obligatoria"] + creditsByType["fund.optativa"] + 
                      creditsByType["dis.obligatoria"] + creditsByType["dis.optativa"] + creditsByType["libre"]
    
    totalRequired := studyPlan.FundObligatoriaCredits + studyPlan.FundOptativaCredits + 
                     studyPlan.DisObligatoriaCredits + studyPlan.DisOptativaCredits + studyPlan.LibreCredits

    creditsSummary.Total = models.CreditTypeInfo{
        Required:  totalRequired,
        Completed: totalCompleted,
        Missing:   max(0, totalRequired - totalCompleted),
    }

    return &models.ComparisonResult{
        EquivalentSubjects: equivalentSubjects,
        MissingSubjects:    missingSubjects,
        CreditsSummary:     creditsSummary,
        TotalCredits:       totalCompleted,
        MissingCredits:     max(0, totalRequired - totalCompleted),
        Warnings:           warnings,
    }, nil
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

    // 2. Crear mapas para facilitar las búsquedas
    studyPlanSubjectsMap := make(map[string]*models.Subject)
    for i := range studyPlan.Subjects {
        code := strings.ToUpper(strings.TrimSpace(studyPlan.Subjects[i].Code))
        studyPlanSubjectsMap[code] = &studyPlan.Subjects[i]
    }

    // 3. Procesar la historia académica
    approvedSubjects := make(map[string]models.SubjectInput)
    for _, historySubject := range academicHistory.Subjects {
        cleanCode := strings.ToUpper(strings.TrimSpace(historySubject.Code))
        if cleanCode != "" {
            approvedSubjects[cleanCode] = historySubject
        }
    }

    // 4. Obtener equivalencias relevantes
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
        
        if _, exists := studyPlanSubjectsMap[sourceCode]; exists {
            equivalenceMap[sourceCode] = append(equivalenceMap[sourceCode], targetCode)
        }
        if _, exists := studyPlanSubjectsMap[targetCode]; exists {
            equivalenceMap[targetCode] = append(equivalenceMap[targetCode], sourceCode)
        }
    }

    // 5. Define credit requirements for ALL types from study plan
    creditosRequeridos := map[string]int{
        "fund.obligatoria": studyPlan.FundObligatoriaCredits,
        "fund.optativa":    studyPlan.FundOptativaCredits,
        "dis.obligatoria":  studyPlan.DisObligatoriaCredits,
        "dis.optativa":     studyPlan.DisOptativaCredits,
        "libre":            studyPlan.LibreCredits,
    }

    // 6. Group subjects by component type
    subjectsByComponent := make(map[string][]models.Subject)
    for _, subject := range studyPlan.Subjects {
        componentKey := mapSubjectTypeToComponentKey(subject.Type)
        subjectsByComponent[componentKey] = append(subjectsByComponent[componentKey], subject)
    }

    var equivalentSubjects []models.SubjectResult
    var missingSubjects []models.SubjectResult
    
    creditsByType := map[string]int{
        "fund.obligatoria": 0,
        "fund.optativa":    0,
        "dis.obligatoria":  0,
        "dis.optativa":     0,
        "libre":            0,
    }

    // 7. Process each component type with credit limits
    for componentKey, subjects := range subjectsByComponent {
        creditosRequeridosComponente := creditosRequeridos[componentKey]
        creditosCompletadosComponente := 0
        
        fmt.Printf("[DEBUG CAMBIO CARRERA] Processing component: %s, required: %d credits\n", 
            componentKey, creditosRequeridosComponente)

        for _, planSubject := range subjects {
            // STOP if component requirement is already fulfilled
            if creditosCompletadosComponente >= creditosRequeridosComponente {
                fmt.Printf("[DEBUG CAMBIO CARRERA] Component %s requirement fulfilled, skipping remaining subjects\n", componentKey)
                break
            }

            planCode := strings.ToUpper(strings.TrimSpace(planSubject.Code))
            isApproved := false
            var equivalenceInfo *models.EquivalenceResult
            
            // Check direct approval
            if _, found := approvedSubjects[planCode]; found {
                isApproved = true
                fmt.Printf("[DEBUG CAMBIO CARRERA] ✅ ENCONTRADA directamente: %s (%s)\n", planSubject.Name, planCode)
            } else {
                // Check approval through equivalence
                if equivalentCodes, hasEquivalences := equivalenceMap[planCode]; hasEquivalences {
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
                
                // Only count credits up to the requirement
                creditsToAdd := planSubject.Credits
                if creditosCompletadosComponente + creditsToAdd > creditosRequeridosComponente {
                    creditsToAdd = creditosRequeridosComponente - creditosCompletadosComponente
                }
                
                creditsByType[componentKey] += creditsToAdd
                creditosCompletadosComponente += creditsToAdd
                
                fmt.Printf("[DEBUG CAMBIO CARRERA] Added %d credits to %s (total: %d/%d)\n", 
                    creditsToAdd, componentKey, creditosCompletadosComponente, creditosRequeridosComponente)
            } else {
                // Only add to missing if we haven't met the requirement yet
                if creditosCompletadosComponente < creditosRequeridosComponente {
                    subjectResult.Status = "PENDIENTE"
                    missingSubjects = append(missingSubjects, subjectResult)
                    fmt.Printf("[DEBUG CAMBIO CARRERA] ❌ Materia faltante: %s (%s)\n", planSubject.Name, planCode)
                }
            }
        }
    }

    // 8. Calculate credits summary based on REQUIRED credits
    creditsSummary := models.CreditsSummary{
        FundObligatoria: models.CreditTypeInfo{
            Required:  studyPlan.FundObligatoriaCredits,
            Completed: creditsByType["fund.obligatoria"],
            Missing:   max(0, studyPlan.FundObligatoriaCredits - creditsByType["fund.obligatoria"]),
        },
        FundOptativa: models.CreditTypeInfo{
            Required:  studyPlan.FundOptativaCredits,
            Completed: creditsByType["fund.optativa"],
            Missing:   max(0, studyPlan.FundOptativaCredits - creditsByType["fund.optativa"]),
        },
        DisObligatoria: models.CreditTypeInfo{
            Required:  studyPlan.DisObligatoriaCredits,
            Completed: creditsByType["dis.obligatoria"],
            Missing:   max(0, studyPlan.DisObligatoriaCredits - creditsByType["dis.obligatoria"]),
        },
        DisOptativa: models.CreditTypeInfo{
            Required:  studyPlan.DisOptativaCredits,
            Completed: creditsByType["dis.optativa"],
            Missing:   max(0, studyPlan.DisOptativaCredits - creditsByType["dis.optativa"]),
        },
        Libre: models.CreditTypeInfo{
            Required:  studyPlan.LibreCredits,
            Completed: creditsByType["libre"],
            Missing:   max(0, studyPlan.LibreCredits - creditsByType["libre"]),
        },
    }

    // Calculate totals based on REQUIRED credits
    totalCompleted := creditsByType["fund.obligatoria"] + creditsByType["fund.optativa"] + 
                      creditsByType["dis.obligatoria"] + creditsByType["dis.optativa"] + creditsByType["libre"]
    
    totalRequired := studyPlan.FundObligatoriaCredits + studyPlan.FundOptativaCredits + 
                     studyPlan.DisObligatoriaCredits + studyPlan.DisOptativaCredits + studyPlan.LibreCredits

    creditsSummary.Total = models.CreditTypeInfo{
        Required:  totalRequired,
        Completed: totalCompleted,
        Missing:   max(0, totalRequired - totalCompleted),
    }

    fmt.Printf("[DEBUG CAMBIO CARRERA] === RESULTADO FINAL ===\n")
    fmt.Printf("[DEBUG CAMBIO CARRERA] Materias equivalentes: %d\n", len(equivalentSubjects))
    fmt.Printf("[DEBUG CAMBIO CARRERA] Materias faltantes: %d\n", len(missingSubjects))
    fmt.Printf("[DEBUG CAMBIO CARRERA] Créditos completados: %d/%d\n", totalCompleted, totalRequired)

    return &models.ComparisonResult{
        EquivalentSubjects: equivalentSubjects,
        MissingSubjects:    missingSubjects,
        CreditsSummary:     creditsSummary,
        TotalCredits:       totalCompleted,
        MissingCredits:     max(0, totalRequired - totalCompleted),
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

	// Eliminar la equivalencia
	if err := db.Delete(&equivalence).Error; err != nil {
		return errors.New("failed to delete equivalence: " + err.Error())
	}

	return nil
}

// UpdateSubjectType actualiza la tipología de una asignatura basándose en su ID
func UpdateSubjectType(db *gorm.DB, subjectID uint, newType string) (*models.Subject, error) {
	// Validar que el tipo es válido
	if !models.ValidarTipologia(newType) {
		return nil, errors.New("invalid subject type. Must be one of: FUND. OBLIGATORIA, FUND. OPTATIVA, DISCIPLINAR OBLIGATORIA, DISCIPLINAR OPTATIVA, LIBRE ELECCIÓN, TRABAJO DE GRADO")
	}

	// Verificar que la asignatura existe
	var subject models.Subject
	if err := db.First(&subject, subjectID).Error; err != nil {
		return nil, errors.New("subject not found")
	}

	// Almacenar el tipo anterior para logging
	oldType := string(subject.Type)

	// Actualizar el tipo
	subject.Type = models.TipologiaAsignatura(newType)

	// Guardar los cambios
	if err := db.Save(&subject).Error; err != nil {
		return nil, errors.New("failed to update subject type: " + err.Error())
	}

	// Log del cambio
	fmt.Printf("[UPDATE SUBJECT TYPE] Asignatura %s (%s): %s → %s\n", 
		subject.Code, subject.Name, oldType, newType)

	return &subject, nil
}

// UpdateSubject actualiza una asignatura completa basándose en su ID
func UpdateSubject(db *gorm.DB, subjectID uint, updates struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Credits     int    `json:"credits"`
	Description string `json:"description"`
}) (*models.Subject, error) {
	// Verificar que la asignatura existe
	var subject models.Subject
	if err := db.First(&subject, subjectID).Error; err != nil {
		return nil, errors.New("subject not found")
	}

	// Preparar campos a actualizar
	updateFields := make(map[string]interface{})

	if updates.Name != "" {
		updateFields["name"] = updates.Name
	}
	
	if updates.Type != "" {
		if !models.ValidarTipologia(updates.Type) {
			return nil, errors.New("invalid subject type. Must be one of: FUND. OBLIGATORIA, FUND. OPTATIVA, DISCIPLINAR OBLIGATORIA, DISCIPLINAR OPTATIVA, LIBRE ELECCIÓN, TRABAJO DE GRADO")
		}
		updateFields["type"] = models.TipologiaAsignatura(updates.Type)
	}
	
	if updates.Credits > 0 {
		updateFields["credits"] = updates.Credits
	}
	
	if updates.Description != "" {
		updateFields["description"] = updates.Description
	}

	// Actualizar solo si hay campos para actualizar
	if len(updateFields) > 0 {
		if err := db.Model(&subject).Updates(updateFields).Error; err != nil {
			return nil, errors.New("failed to update subject: " + err.Error())
		}
	}

	// Recargar la asignatura actualizada
	if err := db.First(&subject, subjectID).Error; err != nil {
		return nil, errors.New("failed to reload updated subject")
	}

	return &subject, nil
}

// GetSubjectByID obtiene una asignatura por su ID
func GetSubjectByID(db *gorm.DB, subjectID uint) (*models.Subject, error) {
	var subject models.Subject
	if err := db.First(&subject, subjectID).Error; err != nil {
		return nil, errors.New("subject not found")
	}
	return &subject, nil
}

// GetSubjectByCode obtiene una asignatura por su código
func GetSubjectByCode(db *gorm.DB, code string) (*models.Subject, error) {
	var subject models.Subject
	if err := db.Where("code = ?", code).First(&subject).Error; err != nil {
		return nil, errors.New("subject not found")
	}
	return &subject, nil
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

// Helper function to process individual subject for homologation
func processSubjectForHomologation(
    materiaPlan models.Subject,
    materiasCursadasOrigen map[string]models.SubjectInput,
    materiasCursadasDoble map[string]models.SubjectInput,
    equivalenciaMap map[string]string,
    isObligatory bool,
) (*models.MateriaHomologable, *models.Warning) {
    
    var materiaOrigen *models.SubjectInput
    var codigoOrigen string
    var nombreOrigen string
    var tipologiaOrigen string
    var equivalenciaInfo *models.EquivalenceResult

    // Check direct match in origen
    if materia, existe := materiasCursadasOrigen[materiaPlan.Code]; existe {
        materiaOrigen = &materia
        codigoOrigen = materia.Code
        nombreOrigen = materia.Name
        tipologiaOrigen = string(materia.Type)
    } else {
        // Check for equivalence
        for codigoOrig, codigoObj := range equivalenciaMap {
            if codigoObj == materiaPlan.Code {
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

    // If subject not found in origen, return nil (cannot be homologated)
    if materiaOrigen == nil {
        // Only generate warning for obligatory subjects
        if isObligatory {
            warning := &models.Warning{
                Type:        "MATERIA_OBLIGATORIA_FALTANTE",
                Message:     fmt.Sprintf("La materia obligatoria '%s' (%s) no está en la historia origen", materiaPlan.Name, materiaPlan.Code),
                SubjectCode: materiaPlan.Code,
                SubjectName: materiaPlan.Name,
            }
            return nil, warning
        }
        return nil, nil
    }

    // Check if already completed in double degree
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
        warning := &models.Warning{
            Type:        "MATERIA_REQUIERE_HOMOLOGACION",
            Message:     fmt.Sprintf("La materia '%s' (%s) está en la historia origen pero no en la doble titulación, requiere homologación", materiaPlan.Name, materiaPlan.Code),
            SubjectCode: materiaPlan.Code,
            SubjectName: materiaPlan.Name,
        }
        return nil, warning
    }

    // Subject can be homologated
    homologable := &models.MateriaHomologable{
        CodigoObjetivo:    materiaPlan.Code,
        NombreObjetivo:    materiaPlan.Name,
        Creditos:          materiaPlan.Credits,
        TipologiaObjetivo: materiaPlan.Type,
        CodigoOrigen:      codigoOrigen,
        NombreOrigen:      nombreOrigen,
        TipologiaOrigen:   tipologiaOrigen,
        Equivalencia:      equivalenciaInfo,
    }

    return homologable, nil
}

// CompareDobleTitulacion compara dos historias académicas para determinar materias homologables
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

    // 5. Define credit requirements for ALL types from study plan
    creditosRequeridos := map[string]int{
        "fund.obligatoria": planObjetivo.FundObligatoriaCredits,
        "fund.optativa":    planObjetivo.FundOptativaCredits,
        "dis.obligatoria":  planObjetivo.DisObligatoriaCredits,
        "dis.optativa":     planObjetivo.DisOptativaCredits,
        "libre":            planObjetivo.LibreCredits,
    }

    // 6. Group ALL subjects by component type
    subjectsByComponent := make(map[string][]models.Subject)
    for _, subject := range planObjetivo.Subjects {
        componentKey := mapSubjectTypeToComponentKey(subject.Type)
        subjectsByComponent[componentKey] = append(subjectsByComponent[componentKey], subject)
    }

    var materiasHomologables []models.MateriaHomologable
    var warnings []models.Warning
    totalCreditos := 0
    creditosPorComponente := make(map[string]int)

    // 7. Process EACH component type with its specific credit limit
    for componentKey, subjects := range subjectsByComponent {
        creditosRequeridosComponente := creditosRequeridos[componentKey]
        creditosCompletadosComponente := 0
        
        fmt.Printf("[DEBUG] Processing component: %s, required credits: %d, available subjects: %d\n", 
            componentKey, creditosRequeridosComponente, len(subjects))

        // Process subjects until credit requirement is met
        for _, materiaPlan := range subjects {
            // STOP processing this component if requirement is already fulfilled
            if creditosCompletadosComponente >= creditosRequeridosComponente {
                fmt.Printf("[DEBUG] Component %s requirement fulfilled (%d/%d credits), skipping remaining subjects\n", 
                    componentKey, creditosCompletadosComponente, creditosRequeridosComponente)
                break
            }
            
            // Check if this subject can be homologated
            homologable, warning := processSubjectForHomologation(
                materiaPlan, materiasCursadasOrigen, materiasCursadasDoble, 
                equivalenciaMap, strings.Contains(componentKey, "obligatoria"),
            )
            
            if warning != nil {
                warnings = append(warnings, *warning)
            }
            
            if homologable != nil {
                materiasHomologables = append(materiasHomologables, *homologable)
                creditosCompletadosComponente += materiaPlan.Credits
                totalCreditos += materiaPlan.Credits
                
                fmt.Printf("[DEBUG] Added subject %s (%d credits) to component %s. Component total: %d/%d\n", 
                    materiaPlan.Code, materiaPlan.Credits, componentKey, 
                    creditosCompletadosComponente, creditosRequeridosComponente)
            }
        }
        
        creditosPorComponente[componentKey] = creditosCompletadosComponente
        fmt.Printf("[DEBUG] Final component %s: %d/%d credits completed\n", 
            componentKey, creditosCompletadosComponente, creditosRequeridosComponente)
    }

    // 8. Calculate summary based on REQUIRED credits for ALL components
    resumen := models.ResumenDobleTitulacion{
        MateriasCursadasOrigen: len(materiasOrigen),
        MateriasCursadasDoble:  len(materiasDoble),
        MateriasHomologables:   len(materiasHomologables),
        CreditosHomologables:   totalCreditos,
    }

    // Calculate total REQUIRED credits (not total possible credits)
    totalCreditosRequeridos := planObjetivo.FundObligatoriaCredits +
                               planObjetivo.FundOptativaCredits +
                               planObjetivo.DisObligatoriaCredits +
                               planObjetivo.DisOptativaCredits +
                               planObjetivo.LibreCredits
    
    if totalCreditosRequeridos > 0 {
        resumen.PorcentajeHomologacion = float64(totalCreditos) / float64(totalCreditosRequeridos) * 100
        if resumen.PorcentajeHomologacion > 100 {
            resumen.PorcentajeHomologacion = 100
        }
    }

    fmt.Printf("[DEBUG] FINAL SUMMARY:\n")
    fmt.Printf("  - Fund Obligatoria: %d/%d credits\n", creditosPorComponente["fund.obligatoria"], planObjetivo.FundObligatoriaCredits)
    fmt.Printf("  - Fund Optativa: %d/%d credits\n", creditosPorComponente["fund.optativa"], planObjetivo.FundOptativaCredits)
    fmt.Printf("  - Dis Obligatoria: %d/%d credits\n", creditosPorComponente["dis.obligatoria"], planObjetivo.DisObligatoriaCredits)
    fmt.Printf("  - Dis Optativa: %d/%d credits\n", creditosPorComponente["dis.optativa"], planObjetivo.DisOptativaCredits)
    fmt.Printf("  - Libre: %d/%d credits\n", creditosPorComponente["libre"], planObjetivo.LibreCredits)
    fmt.Printf("  - TOTAL: %d/%d credits (%.1f%%)\n", totalCreditos, totalCreditosRequeridos, resumen.PorcentajeHomologacion)

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
        // Note: Removed ComponenteAcademico reference as it doesn't exist in your model
        componentKey := mapSubjectTypeToComponentKey(subject.Type)
        if componentKey == componente {
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
    lineas := strings.Split(texto, "\n")
    
    for _, linea := range lineas {
        if linea == "" {
            continue
        }
        
        partes := strings.Split(linea, ":")
        if len(partes) < 5 {
            continue
        }
        
        codigo := strings.TrimSpace(partes[0])
        nombre := strings.TrimSpace(partes[1])
        
        var creditos int
        fmt.Sscanf(partes[2], "%d", &creditos)
        
        tipo := models.TipologiaAsignatura(strings.TrimSpace(partes[3]))
        status := strings.TrimSpace(partes[4])
        
        materias = append(materias, models.SubjectInput{
            Code:     codigo,
            Name:     nombre,
            Credits:  creditos,
            Type:     tipo,
            Status:   status,
            Grade:    0.0,
            Semester: "",
        })
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
