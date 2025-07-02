package models

import (
	"time"
)

// TipologiaAsignatura representa los tipos permitidos de asignaturas
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

// Career representa una carrera en la universidad
type Career struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"size:100;not null"`
	Code        string    `gorm:"size:20;unique;not null"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	StudyPlans  []StudyPlan `gorm:"foreignKey:CareerID"`
}

// StudyPlan representa un plan de estudio de una carrera
type StudyPlan struct {
	ID          uint      `gorm:"primaryKey"`
	CareerID    uint      `gorm:"not null"`
	Version     string    `gorm:"size:20;not null"` // Ejemplo: "2023-1"
	IsActive    bool      `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TotalCredits int `gorm:"not null"`
	FoundationalCredits int `gorm:"not null"`
	DisciplinaryCredits int `gorm:"not null"`
	ElectiveCreditsPercentage int `gorm:"not null"`
	Subjects    []Subject `gorm:"many2many:study_plan_subjects;"`
	Career      Career    `gorm:"foreignKey:CareerID"`
	// Nuevos campos para créditos por tipología
	FundObligatoriaCredits int `gorm:"not null"`
	FundOptativaCredits    int `gorm:"not null"`
	DisObligatoriaCredits  int `gorm:"not null"`
	DisOptativaCredits     int `gorm:"not null"`
	LibreCredits           int `gorm:"not null"`
}

// Subject representa una materia del plan de estudio
type Subject struct {
	ID          uint              `gorm:"primaryKey"`
	Code        string            `gorm:"size:20;unique;not null"` // Código de la materia
	Name        string            `gorm:"size:100;not null"`
	Credits     int               `gorm:"not null"`
	Type        TipologiaAsignatura `gorm:"size:50;not null"` // Tipo de materia (fundamental, disciplinar, etc)
	Description string            `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Relaciones
	Equivalences  []Equivalence `gorm:"foreignKey:SourceSubjectID"`
	StudyPlans    []StudyPlan   `gorm:"many2many:study_plan_subjects;"`
}

// Equivalence representa una equivalencia entre materias de diferentes planes
type Equivalence struct {
	ID              uint      `gorm:"primaryKey"`
	SourceSubjectID uint      `gorm:"not null"` // Materia origen (nueva materia que se crea)
	TargetSubjectID uint      `gorm:"not null"` // Materia destino (materia existente del plan)
	Type            string    `gorm:"size:20;not null"` // Tipo de equivalencia (total, parcial, etc)
	Notes           string    `gorm:"type:text"`
	CareerID        uint      `gorm:"not null"` // Carrera a la que aplica la equivalencia
	CreatedAt       time.Time
	UpdatedAt       time.Time
	// Relaciones
	SourceSubject Subject `gorm:"foreignKey:SourceSubjectID"`
	TargetSubject Subject `gorm:"foreignKey:TargetSubjectID"`
	Career        Career  `gorm:"foreignKey:CareerID"`
}

// AcademicHistoryInput representa la entrada de historia académica para procesar
// Este es un DTO (Data Transfer Object) y no se almacena en la base de datos
type AcademicHistoryInput struct {
	CareerCode    string   `json:"career_code" binding:"required"`
	Subjects      []SubjectInput `json:"subjects" binding:"required"`
}

// SubjectInput representa una materia en la historia académica de entrada
type SubjectInput struct {
	Code        string            `json:"code" binding:"required"`
	Name        string            `json:"name" binding:"required"`
	Credits     int               `json:"credits" binding:"required"`
	Type        TipologiaAsignatura `json:"type" binding:"required"`
	Grade       float64           `json:"grade" binding:"required"`
	Status      string            `json:"status" binding:"required"` // Aprobada, Reprobada, En curso, etc.
	Semester    string            `json:"semester" binding:"required"` // Semestre en que se cursó
}

// ComparisonResult representa el resultado de la comparación de planes
// Este es un DTO y no se almacena en la base de datos
type ComparisonResult struct {
	EquivalentSubjects []SubjectResult `json:"equivalent_subjects"`
	MissingSubjects    []SubjectResult `json:"missing_subjects"`
	TotalCredits       int             `json:"total_credits"`
	MissingCredits     int             `json:"missing_credits"`
	CreditsSummary     CreditsSummary  `json:"credits_summary"`
}

// SubjectResult representa una materia en el resultado de la comparación
type SubjectResult struct {
	Code        string            `json:"code"`
	Name        string            `json:"name"`
	Credits     int               `json:"credits"`
	Type        TipologiaAsignatura `json:"type"`
	Status      string            `json:"status"` // Equivalente, Falta, etc.
	Equivalence *EquivalenceResult `json:"equivalence,omitempty"`
}

// EquivalenceResult representa una equivalencia en el resultado
type EquivalenceResult struct {
	Type  string `json:"type"`
	Notes string `json:"notes"`
}

// CreditTypeInfo representa el resumen de créditos por tipo
type CreditTypeInfo struct {
	Required  int `json:"required"`
	Completed int `json:"completed"`
	Missing   int `json:"missing"`
}

// CreditsSummary representa el resumen de créditos por cada tipología y el total
type CreditsSummary struct {
	FundObligatoria   CreditTypeInfo `json:"fund_obligatoria"`
	FundOptativa      CreditTypeInfo `json:"fund_optativa"`
	DisObligatoria    CreditTypeInfo `json:"dis_obligatoria"`
	DisOptativa       CreditTypeInfo `json:"dis_optativa"`
	Libre             CreditTypeInfo `json:"libre"`
	Total             CreditTypeInfo `json:"total"`
}

// DobleTitulacionInput representa la entrada para comparación de doble titulación
// Ahora recibe el código de la carrera objetivo
type DobleTitulacionInput struct {
	HistoriaOrigen     string `json:"historia_origen" binding:"required"`     // Historia académica del primer plan
	HistoriaDoble      string `json:"historia_doble" binding:"required"`      // Historia académica del segundo plan (doble titulación)
	CodigoCarreraObjetivo string `json:"codigo_carrera_objetivo" binding:"required"` // Código de la carrera objetivo
}

// DobleTitulacionResult representa el resultado de la comparación de doble titulación
type DobleTitulacionResult struct {
	MateriasHomologables []MateriaHomologable `json:"materias_homologables"`
	TotalMaterias        int                  `json:"total_materias"`
	TotalCreditos        int                  `json:"total_creditos"`
	Resumen              ResumenDobleTitulacion `json:"resumen"`
}

// MateriaHomologable representa una materia que se puede homologar en doble titulación
type MateriaHomologable struct {
	CodigoObjetivo     string            `json:"codigo_objetivo"`     // Código en el plan objetivo
	NombreObjetivo     string            `json:"nombre_objetivo"`     // Nombre en el plan objetivo
	Creditos           int               `json:"creditos"`
	TipologiaObjetivo  TipologiaAsignatura `json:"tipologia_objetivo"` // Tipología en el plan objetivo
	CodigoOrigen       string            `json:"codigo_origen"`       // Código original (del primer plan)
	NombreOrigen       string            `json:"nombre_origen"`       // Nombre original (del primer plan)
	TipologiaOrigen    string            `json:"tipologia_origen"`    // Tipología original
	Periodo            string            `json:"periodo"`             // Periodo en que se cursó
	Calificacion       float64           `json:"calificacion"`        // Calificación obtenida
	Equivalencia       *EquivalenceResult `json:"equivalencia,omitempty"` // Info de equivalencia si aplica
}

// ResumenDobleTitulacion representa el resumen de la comparación
type ResumenDobleTitulacion struct {
	MateriasCursadasOrigen    int `json:"materias_cursadas_origen"`    // Total materias cursadas en plan origen
	MateriasCursadasDoble     int `json:"materias_cursadas_doble"`     // Total materias cursadas en plan doble
	MateriasHomologables      int `json:"materias_homologables"`       // Materias que se pueden homologar
	CreditosHomologables      int `json:"creditos_homologables"`       // Créditos que se pueden homologar
	PorcentajeHomologacion    float64 `json:"porcentaje_homologacion"` // Porcentaje de homologación
}