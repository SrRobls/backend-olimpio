package database

import (
	"log"

	"gorm.io/gorm"
	"olimpo-vicedecanatura/models"
)

// RunMigrations ejecuta las migraciones de la base de datos
func RunMigrations(db *gorm.DB) {
	// Auto-migrar los modelos
	err := db.AutoMigrate(
		&models.Career{},
		&models.StudyPlan{},
		&models.Subject{},
		&models.Equivalence{},
	)
	if err != nil {
		log.Fatalf("Error ejecutando migraciones: %v", err)
	}

	// Migración manual para cambiar StudyPlanID por CareerID en equivalences
	// Solo ejecutar si la columna StudyPlanID existe
	var columnExists bool
	db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'equivalences' AND column_name = 'study_plan_id')").Scan(&columnExists)
	
	if columnExists {
		log.Println("Migrando tabla equivalences: cambiando StudyPlanID por CareerID...")
		
		// Agregar la nueva columna CareerID
		if err := db.Exec("ALTER TABLE equivalences ADD COLUMN IF NOT EXISTS career_id BIGINT;").Error; err != nil {
			log.Printf("Error agregando columna career_id: %v", err)
		}
		
		// Copiar datos de StudyPlanID a CareerID (asumiendo que cada plan pertenece a una carrera)
		if err := db.Exec(`
			UPDATE equivalences 
			SET career_id = (
				SELECT career_id 
				FROM study_plans 
				WHERE study_plans.id = equivalences.study_plan_id
			)
			WHERE career_id IS NULL;
		`).Error; err != nil {
			log.Printf("Error copiando datos de StudyPlanID a CareerID: %v", err)
		}
		
		// Eliminar la columna StudyPlanID
		if err := db.Exec("ALTER TABLE equivalences DROP COLUMN IF EXISTS study_plan_id;").Error; err != nil {
			log.Printf("Error eliminando columna study_plan_id: %v", err)
		}
		
		log.Println("Migración de equivalences completada")
	}

	// Crear índices adicionales si son necesarios
	// Por ejemplo, para búsquedas frecuentes por código de materia
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_subjects_code ON subjects(code);").Error; err != nil {
		log.Printf("Error creando índice: %v", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_careers_code ON careers(code);").Error; err != nil {
		log.Printf("Error creando índice: %v", err)
	}
	
	// Índices para equivalences
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_equivalences_career_id ON equivalences(career_id);").Error; err != nil {
		log.Printf("Error creando índice: %v", err)
	}
	
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_equivalences_source_subject_id ON equivalences(source_subject_id);").Error; err != nil {
		log.Printf("Error creando índice: %v", err)
	}
	
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_equivalences_target_subject_id ON equivalences(target_subject_id);").Error; err != nil {
		log.Printf("Error creando índice: %v", err)
	}
}

// SeedInitialData inserta datos iniciales en la base de datos
func SeedInitialData(db *gorm.DB) {
	// Verificar si ya existen datos
	var count int64
	db.Model(&models.Career{}).Count(&count)
	if count > 0 {
		log.Println("La base de datos ya contiene datos iniciales")
		return
	}

	// Crear algunas carreras de ejemplo
	careers := []models.Career{
		{
			Name:        "Ingeniería de Sistemas",
			Code:        "ISIS",
			Description: "Carrera de Ingeniería de Sistemas",
		},
		{
			Name:        "Ingeniería Administrativa",
			Code:        "IADM",
			Description: "Carrera de Ingeniería Administrativa",
		},
		// Agregar más carreras según sea necesario
	}

	// Insertar carreras
	for _, career := range careers {
		if err := db.Create(&career).Error; err != nil {
			log.Printf("Error creando carrera %s: %v", career.Name, err)
		}
	}

	// Nota: Los planes de estudio y materias se pueden cargar desde archivos JSON
	// o mediante una interfaz administrativa
} 