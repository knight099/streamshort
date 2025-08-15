package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"time"

	"streamshort/models"
)

type SimpleMigrationData struct {
	Version     string
	Description string
	Created     string
	Tables      []SimpleTableData
}

type SimpleTableData struct {
	Name   string
	Fields []SimpleFieldData
}

type SimpleFieldData struct {
	Name        string
	Type        string
	Constraints string
}

func generateSimpleMigration(version, description string) {
	// Create migrations directory if it doesn't exist
	migrationsDir := "migrations"
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		log.Fatal("Failed to create migrations directory:", err)
	}

	// Generate simple SQL migration file
	filename := filepath.Join(migrationsDir, fmt.Sprintf("%s_simple.sql", version))

	// Create template with custom functions
	tmpl := template.New("simple").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	})
	tmpl = template.Must(tmpl.Parse(simpleTemplate))

	// Create migration data
	data := SimpleMigrationData{
		Version:     version,
		Description: description,
		Created:     time.Now().Format("2025-08-16"),
		Tables:      extractSimpleTables(),
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("Failed to create SQL migration file:", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		log.Fatal("Failed to execute SQL template:", err)
	}

	fmt.Printf("📄 Generated simple SQL migration: %s\n", filename)
}

func extractSimpleTables() []SimpleTableData {
	var tables []SimpleTableData

	// Extract table info from each model
	models := []interface{}{
		&models.User{},
		&models.OTPTransaction{},
		&models.RefreshToken{},
		&models.CreatorProfile{},
		&models.PayoutDetails{},
		&models.CreatorAnalytics{},
	}

	for _, model := range models {
		table := extractSimpleTableFromModel(model)
		if table.Name != "" {
			tables = append(tables, table)
		}
	}

	return tables
}

func extractSimpleTableFromModel(model interface{}) SimpleTableData {
	t := reflect.TypeOf(model).Elem()

	// Get table name
	tableName := strings.ToLower(t.Name()) + "s"

	var fields []SimpleFieldData

	// Extract fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		gormTag := field.Tag.Get("gorm")
		if strings.Contains(gormTag, "ignore") {
			continue
		}

		fieldName := getSimpleFieldName(field)
		fieldType := getSimpleFieldType(field, gormTag)
		constraints := getSimpleConstraints(gormTag)

		fields = append(fields, SimpleFieldData{
			Name:        fieldName,
			Type:        fieldType,
			Constraints: constraints,
		})
	}

	return SimpleTableData{
		Name:   tableName,
		Fields: fields,
	}
}

func getSimpleFieldName(field reflect.StructField) string {
	gormTag := field.Tag.Get("gorm")

	// Check for column name in GORM tag
	if strings.Contains(gormTag, "column:") {
		parts := strings.Split(gormTag, "column:")
		if len(parts) > 1 {
			columnPart := strings.Split(parts[1], " ")[0]
			return strings.TrimSpace(columnPart)
		}
	}

	return strings.ToLower(field.Name)
}

func getSimpleFieldType(field reflect.StructField, gormTag string) string {
	// Check for explicit type in GORM tag
	if strings.Contains(gormTag, "type:") {
		parts := strings.Split(gormTag, "type:")
		if len(parts) > 1 {
			typePart := strings.Split(parts[1], " ")[0]
			// Clean up the type
			typePart = strings.TrimSpace(typePart)
			if strings.Contains(typePart, ";") {
				typePart = strings.Split(typePart, ";")[0]
			}
			return typePart
		}
	}

	// Map Go types to SQL types
	switch field.Type.Kind() {
	case reflect.String:
		if strings.Contains(gormTag, "primaryKey") {
			return "UUID"
		}
		return "VARCHAR(255)"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "INTEGER"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INTEGER"
	case reflect.Uint64:
		return "BIGINT"
	case reflect.Float32, reflect.Float64:
		return "DECIMAL(10,2)"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Struct:
		if field.Type == reflect.TypeOf(time.Time{}) {
			return "TIMESTAMP WITH TIME ZONE"
		}
		return "TEXT"
	default:
		return "TEXT"
	}
}

func getSimpleConstraints(gormTag string) string {
	var constraints []string

	if strings.Contains(gormTag, "primaryKey") {
		constraints = append(constraints, "PRIMARY KEY")
	}

	if strings.Contains(gormTag, "not null") {
		constraints = append(constraints, "NOT NULL")
	}

	if strings.Contains(gormTag, "unique") {
		constraints = append(constraints, "UNIQUE")
	}

	if strings.Contains(gormTag, "default:") {
		parts := strings.Split(gormTag, "default:")
		if len(parts) > 1 {
			defaultPart := strings.Split(parts[1], " ")[0]
			// Clean up the default value
			defaultPart = strings.TrimSpace(defaultPart)
			if strings.Contains(defaultPart, ";") {
				defaultPart = strings.Split(defaultPart, ";")[0]
			}
			constraints = append(constraints, fmt.Sprintf("DEFAULT %s", defaultPart))
		}
	}

	return strings.Join(constraints, " ")
}

const simpleTemplate = `-- Migration: {{.Version}}_simple.sql
-- Description: {{.Description}}
-- Created: {{.Created}}
-- Auto-generated from Go models

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

{{range .Tables}}
-- Create {{.Name}} table
CREATE TABLE IF NOT EXISTS {{.Name}} (
{{range .Fields}}    {{.Name}} {{.Type}}{{if .Constraints}} {{.Constraints}}{{end}},
{{end}}
);

{{end}}

-- Migration completed successfully
`
