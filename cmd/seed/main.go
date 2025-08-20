package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"streamshort/config"
	"streamshort/models"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	// Load env files if present
	_ = godotenv.Load(".env.local", ".env")

	db := config.InitDB()

	if err := seed(db); err != nil {
		log.Fatalf("seeding failed: %v", err)
	}

	log.Println("Seeding completed.")
}

func seed(db *gorm.DB) error {
	// If we already have series, assume seeded
	var existingSeriesCount int64
	if err := db.Model(&models.Series{}).Count(&existingSeriesCount).Error; err != nil {
		return err
	}
	if existingSeriesCount > 0 {
		log.Printf("Series already present (%d). Skipping seed.", existingSeriesCount)
		return nil
	}

	// 1) Ensure a user exists
	const seedPhone = "+910000000001"
	var user models.User
	if err := db.Where("phone = ?", seedPhone).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			user = models.User{Phone: seedPhone}
			if err := db.Create(&user).Error; err != nil {
				return fmt.Errorf("create user: %w", err)
			}
			log.Printf("Created user %s", user.ID)
		} else {
			return err
		}
	} else {
		log.Printf("Found user %s", user.ID)
	}

	// 2) Ensure a creator profile exists
	var creator models.CreatorProfile
	if err := db.Where("user_id = ?", user.ID).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			creator = models.CreatorProfile{
				UserID:          user.ID,
				DisplayName:     "Demo Studio",
				Bio:             "We make short-form series for demos",
				KYCDocumentPath: "s3://uploads/kyc/demo_kyc_doc.jpg",
				KYCStatus:       "verified",
			}
			if err := db.Create(&creator).Error; err != nil {
				return fmt.Errorf("create creator profile: %w", err)
			}
			log.Printf("Created creator profile %s", creator.ID)
		} else {
			return err
		}
	} else {
		log.Printf("Found creator profile %s", creator.ID)
	}

	// 3) Create a couple of series
	seriesList := []models.Series{
		{
			CreatorID:    creator.ID,
			Title:        "Getting Started with StreamShort",
			Synopsis:     "A walkthrough series covering platform basics",
			Language:     "en",
			PriceType:    "free",
			ThumbnailURL: strPtr("https://example.com/thumbs/series1.jpg"),
			Status:       "published",
		},
		{
			CreatorID:    creator.ID,
			Title:        "Culinary Shorts: 5-Min Recipes",
			Synopsis:     "Quick recipes for busy folks",
			Language:     "en",
			PriceType:    "subscription",
			PriceAmount:  float64Ptr(2.99),
			ThumbnailURL: strPtr("https://example.com/thumbs/series2.jpg"),
			Status:       "draft",
		},
	}

	for i := range seriesList {
		if err := db.Create(&seriesList[i]).Error; err != nil {
			return fmt.Errorf("create series %d: %w", i+1, err)
		}
		log.Printf("Created series %s - %s", seriesList[i].ID, seriesList[i].Title)
	}

	// Manually set category_tags via array literal to avoid driver array encoding issues
	if err := setTextArray(db, "series", "category_tags", seriesList[0].ID, []string{"education", "howto"}); err != nil {
		return fmt.Errorf("set category_tags for series1: %w", err)
	}
	if err := setTextArray(db, "series", "category_tags", seriesList[1].ID, []string{"cooking", "lifestyle"}); err != nil {
		return fmt.Errorf("set category_tags for series2: %w", err)
	}

	// 4) Create episodes for each series
	for _, s := range seriesList {
		episodes := []models.Episode{
			{
				SeriesID:        s.ID,
				Title:           fmt.Sprintf("%s - Episode 1", s.Title),
				EpisodeNumber:   1,
				DurationSeconds: 300,
				HLSManifestURL:  strPtr("https://cdn.example.com/hls/ep1.m3u8"),
				ThumbURL:        strPtr("https://example.com/thumbs/ep1.jpg"),
				Status:          "published",
				PublishedAt:     timePtr(time.Now().Add(-48 * time.Hour)),
			},
			{
				SeriesID:        s.ID,
				Title:           fmt.Sprintf("%s - Episode 2", s.Title),
				EpisodeNumber:   2,
				DurationSeconds: 360,
				HLSManifestURL:  strPtr("https://cdn.example.com/hls/ep2.m3u8"),
				ThumbURL:        strPtr("https://example.com/thumbs/ep2.jpg"),
				Status:          "ready",
			},
		}
		for j := range episodes {
			if err := db.Create(&episodes[j]).Error; err != nil {
				return fmt.Errorf("create episode %d for series %s: %w", j+1, s.ID, err)
			}
			log.Printf("  Created episode %s - %s", episodes[j].ID, episodes[j].Title)
		}
	}

	return nil
}

func strPtr(s string) *string        { return &s }
func float64Ptr(f float64) *float64  { return &f }
func timePtr(t time.Time) *time.Time { return &t }

// setTextArray updates a text[] column using a Postgres array literal
func setTextArray(db *gorm.DB, table string, column string, id string, values []string) error {
	processed := make([]string, 0, len(values))
	for _, v := range values {
		if strings.ContainsAny(v, ",{}\"\\ ") {
			escaped := strings.ReplaceAll(v, "\\", "\\\\")
			escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
			processed = append(processed, "\""+escaped+"\"")
		} else {
			processed = append(processed, v)
		}
	}
	literal := "{" + strings.Join(processed, ",") + "}"
	return db.Exec(
		fmt.Sprintf("UPDATE %s SET %s = ?::text[] WHERE id = ?", table, column),
		literal, id,
	).Error
}
