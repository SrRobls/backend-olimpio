package main



import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	fmt.Println("🚀 Iniciando proceso completo de población de datos...")

	// Obtener el directorio actual
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error obteniendo directorio actual:", err)
	}

	// Cambiar al directorio de scripts
	scriptsDir := filepath.Join(currentDir, "scripts")
	if err := os.Chdir(scriptsDir); err != nil {
		log.Fatal("Error cambiando al directorio de scripts:", err)
	}

	// 1. Crear la carrera
	fmt.Println("\n📚 Paso 1: Creando carrera de Ingeniería de Sistemas...")
	cmd1 := exec.Command("go", "run", "create_career.go")
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	
	if err := cmd1.Run(); err != nil {
		log.Fatal("Error ejecutando create_career.go:", err)
	}

	// 2. Poblar el pensum
	fmt.Println("\n📖 Paso 2: Poblando pensum de Ingeniería de Sistemas...")
	cmd2 := exec.Command("go", "run", "seed_ing_sistemas.go")
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	
	if err := cmd2.Run(); err != nil {
		log.Fatal("Error ejecutando seed_ing_sistemas.go:", err)
	}

	fmt.Println("\n🎉 ¡Proceso completado exitosamente!")
	fmt.Println("✅ La base de datos ha sido poblada con:")
	fmt.Println("   - Carrera de Ingeniería de Sistemas")
	fmt.Println("   - Plan de estudio 2023-1")
	fmt.Println("   - Todas las materias del pensum")
	fmt.Println("   - Equivalencias entre materias")
} 