package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	SourceSubjectID uint      `gorm:"not null"` // Materia origen
	TargetSubjectID uint      `gorm:"not null"` // Materia destino
	Type            string    `gorm:"size:20;not null"` // Tipo de equivalencia (total, parcial, etc)
	Notes           string    `gorm:"type:text"`
	StudyPlanID     uint      `gorm:"not null"` // Plan de estudio al que aplica la equivalencia
	CareerID        uint      `gorm:"not null"` // Carrera a la que aplica la equivalencia
	CreatedAt       time.Time
	UpdatedAt       time.Time
	// Relaciones
	SourceSubject Subject   `gorm:"foreignKey:SourceSubjectID"`
	TargetSubject Subject   `gorm:"foreignKey:TargetSubjectID"`
	StudyPlan     StudyPlan `gorm:"foreignKey:StudyPlanID"`
}

// Datos del pensum de Ingeniería de Sistemas
var planIngSistemas = []map[string]interface{}{
	{"codigo": "1000004-M", "nombre": "Cálculo Diferencial", "creditos": 4, "tipologia": "B - Fundamentación Obligatoria"},
	{"codigo": "1000008-M", "nombre": "Geometría Vectorial y Analítica", "creditos": 4, "tipologia": "B - Fundamentación Obligatoria"},
	{"codigo": "1000005-M", "nombre": "Cálculo Integral", "creditos": 4, "tipologia": "B - Fundamentación Obligatoria"},
	{"codigo": "1000003-M", "nombre": "Álgebra Lineal", "creditos": 4, "tipologia": "B - Fundamentación Obligatoria"},
	{"codigo": "3006906", "nombre": "Matemáticas Discretas", "creditos": 4, "tipologia": "B - Fundamentación Obligatoria"},
	{"codigo": "3010651", "nombre": "Estadística I", "creditos": 3, "tipologia": "B - Fundamentación Obligatoria"},
	{"codigo": "1000019-M", "nombre": "Física Mecánica", "creditos": 4, "tipologia": "B - Fundamentación Obligatoria"},
	{"codigo": "1000007-M", "nombre": "Ecuaciones Diferenciales", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "1000006-M", "nombre": "Cálculo en Varias Variables", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3006905", "nombre": "Matemáticas Especiales", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3006907", "nombre": "Métodos Numéricos", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3010334", "nombre": "Fundamentos de Matemáticas", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3010391", "nombre": "Geometría Aplicada", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3006915", "nombre": "Estadística II", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3006927", "nombre": "Estadística Descriptiva y Exploratoria", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3009137", "nombre": "Estadística III", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "1000017-M", "nombre": "Física de Electricidad y Magnetismo", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "1000020-M", "nombre": "Física de Oscilaciones, Ondas y Óptica", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3006829", "nombre": "Química General", "creditos": 3, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3006825", "nombre": "Laboratorio de Química General", "creditos": 2, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "1000009-M", "nombre": "Biología General", "creditos": 3, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3006931", "nombre": "Introducción al Manejo de Datos Estadísticos", "creditos": 4, "tipologia": "O - Fundamentación Optativa"},
	{"codigo": "3010435", "nombre": "Fundamentos de Programación", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007744", "nombre": "Programación Orientada a Objetos", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007741", "nombre": "Estructura de Datos", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007847", "nombre": "Bases de Datos I", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3010426", "nombre": "Teoría de Lenguajes de Programación", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3010476", "nombre": "Introducción a la Inteligencia Artificial", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007867", "nombre": "Sistemas Operativos", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007865", "nombre": "Redes y Telecomunicaciones I", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3011020", "nombre": "Fundamentos de Analítica", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007853", "nombre": "Ingeniería de Software", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007852", "nombre": "Ingeniería de Requisitos", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3010440", "nombre": "Calidad de Software", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3010438", "nombre": "Introducción a la Ingeniería de Sistemas e Informática", "creditos": 2, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007324", "nombre": "Investigación de Operaciones I", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3010415", "nombre": "Introducción al Análisis de Decisiones", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007331", "nombre": "Simulación de Sistemas", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3010408", "nombre": "Fundamentos de Proyectos en Ingeniería", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3010407", "nombre": "Estructuración y Evaluación de Proyectos de Ingeniería", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3010439", "nombre": "Proyecto Integrado de Ingeniería", "creditos": 4, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3011166", "nombre": "Fundamentos de Sistemas de Información e Inteligencia de Negocios", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007854", "nombre": "Técnicas de Aprendizaje Estadístico", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007860", "nombre": "Sistema de Recuperación de Información de web", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007862", "nombre": "Visión Artificial", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3009150", "nombre": "Redes Neuronales Artificiales y Algoritmos Bioinspirados", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3009151", "nombre": "Introducción a la Robótica", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007850", "nombre": "Diseño y Construcción de Productos de Software", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007848", "nombre": "Bases de Datos II", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3009430", "nombre": "Análisis y Diseño de Algoritmos", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007871", "nombre": "Programación Matemática", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007872", "nombre": "Sistemas Complejos", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007325", "nombre": "Investigación de Operaciones II", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007311", "nombre": "Dinámica de Sistemas", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007873", "nombre": "Teoría de la Organización Industrial", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007851", "nombre": "Gestión de Proyectos de Software", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3010836", "nombre": "Cátedra de Sistemas: una Visión Histórico-Cultural de la Computación", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3010757", "nombre": "Ciencias de la Computación y Aplicaciones Móviles", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3010585", "nombre": "Creación de Videojuegos", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3009936", "nombre": "Seguridad Web", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3011172", "nombre": "Desarrollo Web II", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3011093", "nombre": "Computación Paralela", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3011018", "nombre": "Creación Multimedia", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
}

// Equivalencias para Ingeniería de Sistemas
var equivalenciasIngSistemas = map[string][]string{
	"3006914": {"3010651"},
	"3007742": {"3010435"},
	"3007743": {"3010426"},
	"3007855": {"3010476"},
	"3007849": {"3010440"},
	"3007322": {"3010415"},
	"3007746": {"3010114"},
	"3009550": {"3007862"},
	"3007844": {"3010408"},
	"3007845": {"3010407"},
	"3007846": {"3010439"},
	"3008883": {"3011166"},
}

// Materias origen para equivalencias (códigos antiguos)
var materiasOrigenEquivalencias = []map[string]interface{}{
	{"codigo": "3006914", "nombre": "Estadística I (Antigua)", "creditos": 3, "tipologia": "B - Fundamentación Obligatoria"},
	{"codigo": "3007742", "nombre": "Fundamentos de Programación (Antigua)", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007743", "nombre": "Teoría de Lenguajes de Programación (Antigua)", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007855", "nombre": "Introducción a la Inteligencia Artificial (Antigua)", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007849", "nombre": "Calidad de Software (Antigua)", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007322", "nombre": "Introducción al Análisis de Decisiones (Antigua)", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007746", "nombre": "Materia Antigua 3007746", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3009550", "nombre": "Visión Artificial (Antigua)", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
	{"codigo": "3007844", "nombre": "Fundamentos de Proyectos en Ingeniería (Antigua)", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007845", "nombre": "Estructuración y Evaluación de Proyectos de Ingeniería (Antigua)", "creditos": 3, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3007846", "nombre": "Proyecto Integrado de Ingeniería (Antigua)", "creditos": 4, "tipologia": "C - Disciplinar Obligatoria"},
	{"codigo": "3008883", "nombre": "Fundamentos de Sistemas de Información e Inteligencia de Negocios (Antigua)", "creditos": 3, "tipologia": "T - Disciplinar Optativa"},
}

// Función para mapear tipologías del prototipo a las del modelo
func mapTipologia(tipologia string) TipologiaAsignatura {
	switch tipologia {
	case "B - Fundamentación Obligatoria":
		return TipologiaFundamentalObligatoria
	case "O - Fundamentación Optativa":
		return TipologiaFundamentalOptativa
	case "C - Disciplinar Obligatoria":
		return TipologiaDisciplinarObligatoria
	case "T - Disciplinar Optativa":
		return TipologiaDisciplinarOptativa
	default:
		return TipologiaLibreEleccion
	}
}

func main() {
	// Configuración de la base de datos
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=olimpo_vicedecanatura port=5432 sslmode=disable"
	}

	// Conectar a la base de datos
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Error conectando a la base de datos:", err)
	}

	// Obtener el ID de la carrera
	careerID := uint(1) // Por defecto, usar ID 1 (ISIS)
	
	// Intentar leer desde el archivo career_info.txt
	if careerInfoBytes, err := os.ReadFile("career_info.txt"); err == nil {
		careerInfo := string(careerInfoBytes)
		fmt.Sscanf(careerInfo, "CAREER_ID=%d", &careerID)
		fmt.Printf("📋 Usando ID de carrera desde archivo: %d\n", careerID)
	} else {
		fmt.Printf("📋 Usando ID de carrera por defecto: %d (ISIS)\n", careerID)
	}

	fmt.Println("🚀 Iniciando población de datos para Ingeniería de Sistemas...")

	// 1. Crear el plan de estudio
	studyPlan := StudyPlan{
		CareerID:    careerID,
		Version:     "2023-1",
		IsActive:    true,
		TotalCredits: 0, // Se calculará después
		FoundationalCredits: 0,
		DisciplinaryCredits: 0,
		ElectiveCreditsPercentage: 0,
		FundObligatoriaCredits: 0,
		FundOptativaCredits: 0,
		DisObligatoriaCredits: 0,
		DisOptativaCredits: 0,
		LibreCredits: 0,
	}

	if err := db.Create(&studyPlan).Error; err != nil {
		log.Fatal("Error creando el plan de estudio:", err)
	}
	fmt.Printf("✅ Plan de estudio creado con ID: %d\n", studyPlan.ID)

	// 2. Crear las materias
	var subjects []Subject
	var totalCredits, fundObligatoriaCredits, fundOptativaCredits, disObligatoriaCredits, disOptativaCredits int

	for _, materia := range planIngSistemas {
		subject := Subject{
			Code:        materia["codigo"].(string),
			Name:        materia["nombre"].(string),
			Credits:     materia["creditos"].(int),
			Type:        mapTipologia(materia["tipologia"].(string)),
			Description: fmt.Sprintf("Materia del plan de Ingeniería de Sistemas - %s", materia["tipologia"].(string)),
		}

		// Crear o actualizar la materia
		var existingSubject Subject
		if err := db.Where("code = ?", subject.Code).First(&existingSubject).Error; err != nil {
			if err := db.Create(&subject).Error; err != nil {
				log.Printf("Error creando materia %s: %v", subject.Code, err)
				continue
			}
			fmt.Printf("✅ Materia creada: %s - %s\n", subject.Code, subject.Name)
		} else {
			subject = existingSubject
			fmt.Printf("ℹ️  Materia ya existe: %s - %s\n", subject.Code, subject.Name)
		}

		subjects = append(subjects, subject)
		totalCredits += subject.Credits

		// Contar créditos por tipología
		switch subject.Type {
		case TipologiaFundamentalObligatoria:
			fundObligatoriaCredits += subject.Credits
		case TipologiaFundamentalOptativa:
			fundOptativaCredits += subject.Credits
		case TipologiaDisciplinarObligatoria:
			disObligatoriaCredits += subject.Credits
		case TipologiaDisciplinarOptativa:
			disOptativaCredits += subject.Credits
		}
	}

	// 3. Asociar materias al plan de estudio
	for _, subject := range subjects {
		if err := db.Model(&studyPlan).Association("Subjects").Append(&subject); err != nil {
			log.Printf("Error asociando materia %s al plan: %v", subject.Code, err)
		}
	}

	// 4. Actualizar estadísticas del plan de estudio
	studyPlan.TotalCredits = totalCredits
	studyPlan.FoundationalCredits = fundObligatoriaCredits + fundOptativaCredits
	studyPlan.DisciplinaryCredits = disObligatoriaCredits + disOptativaCredits
	studyPlan.FundObligatoriaCredits = fundObligatoriaCredits
	studyPlan.FundOptativaCredits = fundOptativaCredits
	studyPlan.DisObligatoriaCredits = disObligatoriaCredits
	studyPlan.DisOptativaCredits = disOptativaCredits

	if err := db.Save(&studyPlan).Error; err != nil {
		log.Fatal("Error actualizando estadísticas del plan:", err)
	}

	// 5. Crear materias origen para equivalencias
	fmt.Println("\n📚 Creando materias origen para equivalencias...")
	for _, materiaOrigen := range materiasOrigenEquivalencias {
		subject := Subject{
			Code:        materiaOrigen["codigo"].(string),
			Name:        materiaOrigen["nombre"].(string),
			Credits:     materiaOrigen["creditos"].(int),
			Type:        mapTipologia(materiaOrigen["tipologia"].(string)),
			Description: fmt.Sprintf("Materia origen para equivalencias - %s", materiaOrigen["tipologia"].(string)),
		}

		// Crear o actualizar la materia origen
		var existingSubject Subject
		if err := db.Where("code = ?", subject.Code).First(&existingSubject).Error; err != nil {
			if err := db.Create(&subject).Error; err != nil {
				log.Printf("Error creando materia origen %s: %v", subject.Code, err)
				continue
			}
			fmt.Printf("✅ Materia origen creada: %s - %s\n", subject.Code, subject.Name)
		} else {
			subject = existingSubject
			fmt.Printf("ℹ️  Materia origen ya existe: %s - %s\n", subject.Code, subject.Name)
		}
	}

	// 6. Crear equivalencias
	fmt.Println("\n🔗 Creando equivalencias...")
	for codigoAntiguo, codigosNuevos := range equivalenciasIngSistemas {
		// Buscar materia origen
		var sourceSubject Subject
		if err := db.Where("code = ?", codigoAntiguo).First(&sourceSubject).Error; err != nil {
			fmt.Printf("⚠️  Materia origen no encontrada: %s\n", codigoAntiguo)
			continue
		}

		for _, codigoNuevo := range codigosNuevos {
			// Buscar materia destino
			var targetSubject Subject
			if err := db.Where("code = ?", codigoNuevo).First(&targetSubject).Error; err != nil {
				fmt.Printf("⚠️  Materia destino no encontrada: %s\n", codigoNuevo)
				continue
			}

			// Crear equivalencia
			equivalence := Equivalence{
				SourceSubjectID: sourceSubject.ID,
				TargetSubjectID: targetSubject.ID,
				Type:            "TOTAL",
				Notes:           fmt.Sprintf("Equivalencia automática: %s → %s", codigoAntiguo, codigoNuevo),
				StudyPlanID:     studyPlan.ID,
				CareerID:        careerID,
			}

			if err := db.Create(&equivalence).Error; err != nil {
				log.Printf("Error creando equivalencia %s → %s: %v", codigoAntiguo, codigoNuevo, err)
			} else {
				fmt.Printf("✅ Equivalencia creada: %s → %s\n", codigoAntiguo, codigoNuevo)
			}
		}
	}

	fmt.Printf("\n🎉 ¡Población completada!\n")
	fmt.Printf("📊 Resumen:\n")
	fmt.Printf("   - Plan de estudio: %s (ID: %d)\n", studyPlan.Version, studyPlan.ID)
	fmt.Printf("   - Total materias: %d\n", len(subjects))
	fmt.Printf("   - Total créditos: %d\n", totalCredits)
	fmt.Printf("   - Créditos Fundamentación Obligatoria: %d\n", fundObligatoriaCredits)
	fmt.Printf("   - Créditos Fundamentación Optativa: %d\n", fundOptativaCredits)
	fmt.Printf("   - Créditos Disciplinar Obligatoria: %d\n", disObligatoriaCredits)
	fmt.Printf("   - Créditos Disciplinar Optativa: %d\n", disOptativaCredits)
	fmt.Printf("   - Total equivalencias: %d\n", len(equivalenciasIngSistemas))
} 