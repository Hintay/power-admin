package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"Power-Monitor/model"

	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// GenerateRegCodeCommand generates a new registration code
var GenerateRegCodeCommand = &cli.Command{
	Name:  "generate",
	Usage: "Generate a new registration code",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"d"},
			Usage:   "Description for the registration code",
		},
		&cli.StringFlag{
			Name:     "user",
			Aliases:  []string{"u"},
			Usage:    "Username of the code creator",
			Required: true,
		},
		&cli.IntFlag{
			Name:    "expires",
			Aliases: []string{"e"},
			Usage:   "Code expiration time in hours",
			Value:   24,
		},
		&cli.StringFlag{
			Name:    "code",
			Aliases: []string{"c"},
			Usage:   "Custom code (auto-generated if not provided)",
		},
	},
	Action: GenerateRegCode,
}

// ListRegCodesCommand lists all registration codes
var ListRegCodesCommand = &cli.Command{
	Name:   "list",
	Usage:  "List all registration codes",
	Action: ListRegCodes,
}

// RevokeRegCodeCommand revokes a registration code
var RevokeRegCodeCommand = &cli.Command{
	Name:  "revoke",
	Usage: "Revoke a registration code",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "code",
			Aliases:  []string{"c"},
			Usage:    "Registration code to revoke",
			Required: true,
		},
	},
	Action: RevokeRegCode,
}

// generateRandomCode generates a random registration code
func generateRandomCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateRegCode generates a new registration code
func GenerateRegCode(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	description := command.String("description")
	username := command.String("user")
	expiresHours := command.Int("expires")
	customCode := command.String("code")

	var userID uint
	if username != "" {
		var user model.User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user '%s' not found", username)
			}
			return fmt.Errorf("failed to find user: %v", err)
		}
		userID = user.ID
	}

	// Generate or use custom code
	var code string
	if customCode != "" {
		code = customCode
		// Check if code already exists
		var existingCode model.RegistrationCode
		if err := db.Where("code = ?", code).First(&existingCode).Error; err == nil {
			return fmt.Errorf("registration code '%s' already exists", code)
		}
	} else {
		code, err = generateRandomCode()
		if err != nil {
			return fmt.Errorf("failed to generate code: %v", err)
		}
	}

	// Create registration code
	regCode := model.RegistrationCode{
		Code:        code,
		Description: description,
		IsUsed:      false,
		ExpiresAt:   time.Now().Add(time.Duration(expiresHours) * time.Hour),
		UserID:      userID,
	}

	if err := db.Create(&regCode).Error; err != nil {
		return fmt.Errorf("failed to create registration code: %v", err)
	}

	fmt.Printf("Registration code generated successfully\n")
	fmt.Printf("Code: %s\n", code)
	fmt.Printf("Description: %s\n", description)
	fmt.Printf("Expires: %s\n", regCode.ExpiresAt.Format("2006-01-02 15:04:05"))
	return nil
}

// ListRegCodes lists all registration codes
func ListRegCodes(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	var regCodes []model.RegistrationCode
	if err := db.Preload("User").Find(&regCodes).Error; err != nil {
		return fmt.Errorf("failed to fetch registration codes: %v", err)
	}

	if len(regCodes) == 0 {
		fmt.Println("No registration codes found")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCode\tDescription\tUsed\tUsed By\tCreator\tExpires\tCreated At")
	fmt.Fprintln(w, "----\t----\t-----------\t----\t-------\t-------\t-------\t----------")

	for _, regCode := range regCodes {
		usedStatus := "No"
		if regCode.IsUsed {
			usedStatus = "Yes"
		}

		usedBy := "N/A"
		if regCode.UsedBy != "" {
			usedBy = regCode.UsedBy
		}

		creator := "N/A"
		if regCode.User.Username != "" {
			creator = regCode.User.Username
		}

		// Check if expired
		expired := ""
		if time.Now().After(regCode.ExpiresAt) {
			expired = " (EXPIRED)"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s%s\t%s\n",
			regCode.ID, regCode.Code, regCode.Description,
			usedStatus, usedBy, creator,
			regCode.ExpiresAt.Format("2006-01-02 15:04:05"), expired,
			regCode.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	w.Flush()
	return nil
}

// RevokeRegCode revokes a registration code
func RevokeRegCode(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	code := command.String("code")

	// Find registration code
	var regCode model.RegistrationCode
	if err := db.Where("code = ?", code).First(&regCode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("registration code '%s' not found", code)
		}
		return fmt.Errorf("failed to find registration code: %v", err)
	}

	if regCode.IsUsed {
		return fmt.Errorf("registration code '%s' has already been used by '%s'", code, regCode.UsedBy)
	}

	// Mark as used (effectively revoking it)
	regCode.IsUsed = true
	regCode.UsedBy = "REVOKED"

	if err := db.Save(&regCode).Error; err != nil {
		return fmt.Errorf("failed to revoke registration code: %v", err)
	}

	fmt.Printf("Registration code '%s' revoked successfully\n", code)
	return nil
}
