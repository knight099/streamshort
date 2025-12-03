package main

import (
	"fmt"
	"log"
	"os"
	"streamshort/config"
	"streamshort/models"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(".env.local"); err == nil {
		log.Println("Loaded environment from .env.local")
	}
	_ = godotenv.Load()

	// Initialize database
	svcs := config.InitServices()
	db := svcs.DB

	// Create test OTP transaction
	phone := "+919650690943"
	otp := "069009"

	// Delete any existing OTP for this phone
	db.Where("phone = ?", phone).Delete(&models.OTPTransaction{})

	// Create new OTP transaction
	otpTx := models.OTPTransaction{
		TxnID:     fmt.Sprintf("test_%d", time.Now().Unix()),
		Phone:     phone,
		OTP:       otp,
		Used:      false,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := db.Create(&otpTx).Error; err != nil {
		log.Fatalf("Failed to create OTP transaction: %v", err)
	}

	fmt.Printf("✅ Created OTP transaction for phone: %s\n", phone)
	fmt.Printf("   OTP: %s\n", otp)
	fmt.Printf("   Expires at: %s\n", otpTx.ExpiresAt.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("You can now use this OTP to authenticate:")
	fmt.Printf("curl -X POST http://localhost:8080/auth/otp/verify \\\n")
	fmt.Printf("  -H 'Content-Type: application/json' \\\n")
	fmt.Printf("  -d '{\"phone\": \"%s\", \"otp\": \"%s\"}'\n", phone, otp)

	os.Exit(0)
}
