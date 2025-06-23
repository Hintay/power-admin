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

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

// AddCollectorCommand adds a new collector
var AddCollectorCommand = &cli.Command{
	Name:  "add",
	Usage: "Add a new collector",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "id",
			Aliases: []string{"i"},
			Usage:   "Collector ID (auto-generated UUID if not provided)",
		},
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Usage:    "Collector name",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"d"},
			Usage:   "Collector description",
		},
		&cli.StringFlag{
			Name:    "location",
			Aliases: []string{"l"},
			Usage:   "Collector location",
		},
		&cli.StringFlag{
			Name:    "user",
			Aliases: []string{"u"},
			Usage:   "Username of the collector owner",
		},
	},
	Action: AddCollector,
}

// ListCollectorsCommand lists all collectors
var ListCollectorsCommand = &cli.Command{
	Name:   "list",
	Usage:  "List all collectors",
	Action: ListCollectors,
}

// UpdateCollectorCommand updates an existing collector
var UpdateCollectorCommand = &cli.Command{
	Name:  "update",
	Usage: "Update an existing collector",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "id",
			Aliases:  []string{"i"},
			Usage:    "Collector ID",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "name",
			Aliases: []string{"n"},
			Usage:   "New collector name",
		},
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"d"},
			Usage:   "New collector description",
		},
		&cli.StringFlag{
			Name:    "location",
			Aliases: []string{"l"},
			Usage:   "New collector location",
		},
		&cli.BoolFlag{
			Name:    "active",
			Aliases: []string{"a"},
			Usage:   "Set collector as active",
		},
		&cli.BoolFlag{
			Name:  "inactive",
			Usage: "Set collector as inactive",
		},
	},
	Action: UpdateCollector,
}

// ConfigCollectorCommand configures collector parameters
var ConfigCollectorCommand = &cli.Command{
	Name:  "config",
	Usage: "Configure collector parameters",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "id",
			Aliases:  []string{"i"},
			Usage:    "Collector ID",
			Required: true,
		},
		&cli.IntFlag{
			Name:    "sample-interval",
			Aliases: []string{"si"},
			Usage:   "Sample interval in seconds",
		},
		&cli.IntFlag{
			Name:    "upload-interval",
			Aliases: []string{"ui"},
			Usage:   "Upload interval in seconds",
		},
		&cli.IntFlag{
			Name:    "max-cache-size",
			Aliases: []string{"mcs"},
			Usage:   "Maximum cache size (number of records)",
		},
		&cli.BoolFlag{
			Name:    "auto-upload",
			Aliases: []string{"au"},
			Usage:   "Enable auto upload",
		},
		&cli.BoolFlag{
			Name:    "no-auto-upload",
			Aliases: []string{"nau"},
			Usage:   "Disable auto upload",
		},
		&cli.IntFlag{
			Name:    "compression-level",
			Aliases: []string{"cl"},
			Usage:   "Compression level (0-9)",
		},
	},
	Action: ConfigCollector,
}

// StatusCollectorCommand shows collector status
var StatusCollectorCommand = &cli.Command{
	Name:  "status",
	Usage: "Show collector status",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "id",
			Aliases: []string{"i"},
			Usage:   "Collector ID (shows all if not specified)",
		},
	},
	Action: StatusCollector,
}

// DeleteCollectorCommand deletes a collector
var DeleteCollectorCommand = &cli.Command{
	Name:  "delete",
	Usage: "Delete a collector",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "id",
			Aliases:  []string{"i"},
			Usage:    "Collector ID",
			Required: true,
		},
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Force delete without confirmation",
		},
	},
	Action: DeleteCollector,
}

// generateCollectorToken generates a random token for a collector.
func generateCollectorToken() (string, error) {
	bytes := make([]byte, 32) // Generates a 64-character hex string
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// AddCollector adds a new collector
func AddCollector(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	collectorID := command.String("id")
	name := command.String("name")
	description := command.String("description")
	location := command.String("location")
	username := command.String("user")

	// Generate UUID if ID not provided
	if collectorID == "" {
		collectorID = uuid.New().String()
		fmt.Printf("Generated Collector ID: %s\n", collectorID)
	}

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

	// Generate collector token
	collectorToken, err := generateCollectorToken()
	if err != nil {
		return fmt.Errorf("failed to generate collector token: %v", err)
	}

	// Create collector
	collector := model.Collector{
		CollectorID: collectorID,
		Name:        name,
		Description: description,
		Location:    location,
		IsActive:    true,
		UserID:      userID,
		Token:       collectorToken,
	}

	if err := db.Create(&collector).Error; err != nil {
		return fmt.Errorf("failed to create collector: %v", err)
	}

	// Create default configuration
	config := model.CollectorConfig{
		CollectorID:      collectorID,
		SampleInterval:   15,
		UploadInterval:   60,
		MaxCacheSize:     1000,
		AutoUpload:       true,
		CompressionLevel: 6,
	}

	if err := db.Create(&config).Error; err != nil {
		fmt.Printf("Warning: failed to create collector config: %v\n", err)
	}

	fmt.Printf("\n=== Collector Added Successfully ===\n")
	fmt.Printf("Collector ID: %s\n", collectorID)
	fmt.Printf("Database ID: %d\n", collector.ID)
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Token: %s\n", collector.Token)
	if description != "" {
		fmt.Printf("Description: %s\n", description)
	}
	if location != "" {
		fmt.Printf("Location: %s\n", location)
	}

	return nil
}

// ListCollectors lists all collectors
func ListCollectors(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	var collectors []model.Collector
	if err := db.Preload("User").Find(&collectors).Error; err != nil {
		return fmt.Errorf("failed to fetch collectors: %v", err)
	}

	if len(collectors) == 0 {
		fmt.Println("No collectors found")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCollector ID\tName\tLocation\tActive\tOnline\tOwner\tLast Seen\tCreated At")
	fmt.Fprintln(w, "----\t------------\t----\t--------\t------\t------\t-----\t---------\t----------")

	for _, collector := range collectors {
		activeStatus := "Yes"
		if !collector.IsActive {
			activeStatus = "No"
		}

		onlineStatus := "No"
		if collector.IsOnline() {
			onlineStatus = "Yes"
		}

		lastSeen := "Never"
		if !collector.LastSeenAt.IsZero() {
			lastSeen = collector.LastSeenAt.Format("2006-01-02 15:04:05")
		}

		owner := "N/A"
		if collector.User.Username != "" {
			owner = collector.User.Username
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			collector.ID, collector.CollectorID, collector.Name,
			collector.Location, activeStatus, onlineStatus, owner,
			lastSeen, collector.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	w.Flush()
	return nil
}

// UpdateCollector updates an existing collector
func UpdateCollector(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	collectorID := command.String("id")
	name := command.String("name")
	description := command.String("description")
	location := command.String("location")
	active := command.Bool("active")
	inactive := command.Bool("inactive")

	// Find collector
	var collector model.Collector
	if err := db.Where("collector_id = ?", collectorID).First(&collector).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("collector '%s' not found", collectorID)
		}
		return fmt.Errorf("failed to find collector: %v", err)
	}

	// Prepare updates
	updates := make(map[string]interface{})

	if name != "" {
		updates["name"] = name
	}
	if description != "" {
		updates["description"] = description
	}
	if location != "" {
		updates["location"] = location
	}
	if active && inactive {
		return fmt.Errorf("cannot set both active and inactive flags")
	}
	if active {
		updates["is_active"] = true
	}
	if inactive {
		updates["is_active"] = false
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates specified")
	}

	// Update collector
	if err := db.Model(&collector).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update collector: %v", err)
	}

	fmt.Printf("Collector '%s' updated successfully\n", collectorID)
	return nil
}

// ConfigCollector configures collector parameters
func ConfigCollector(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	collectorID := command.String("id")
	sampleInterval := command.Int("sample-interval")
	uploadInterval := command.Int("upload-interval")
	maxCacheSize := command.Int("max-cache-size")
	autoUpload := command.Bool("auto-upload")
	noAutoUpload := command.Bool("no-auto-upload")
	compressionLevel := command.Int("compression-level")

	// Find collector config
	var config model.CollectorConfig
	if err := db.Where("collector_id = ?", collectorID).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("collector config for '%s' not found", collectorID)
		}
		return fmt.Errorf("failed to find collector config: %v", err)
	}

	// Prepare updates
	updates := make(map[string]interface{})

	if sampleInterval > 0 {
		updates["sample_interval"] = sampleInterval
	}
	if uploadInterval > 0 {
		updates["upload_interval"] = uploadInterval
	}
	if maxCacheSize > 0 {
		updates["max_cache_size"] = maxCacheSize
	}
	if autoUpload && noAutoUpload {
		return fmt.Errorf("cannot set both auto-upload and no-auto-upload flags")
	}
	if autoUpload {
		updates["auto_upload"] = true
	}
	if noAutoUpload {
		updates["auto_upload"] = false
	}
	if compressionLevel >= 0 && compressionLevel <= 9 {
		updates["compression_level"] = compressionLevel
	} else if compressionLevel != 0 {
		return fmt.Errorf("compression level must be between 0 and 9")
	}

	if len(updates) == 0 {
		return fmt.Errorf("no configuration updates specified")
	}

	// Update config
	if err := db.Model(&config).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update collector config: %v", err)
	}

	fmt.Printf("Collector '%s' configuration updated successfully\n", collectorID)
	return nil
}

// StatusCollector shows collector status
func StatusCollector(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	collectorID := command.String("id")

	var collectors []model.Collector
	query := db.Preload("User")
	if collectorID != "" {
		query = query.Where("collector_id = ?", collectorID)
	}

	if err := query.Find(&collectors).Error; err != nil {
		return fmt.Errorf("failed to fetch collectors: %v", err)
	}

	if len(collectors) == 0 {
		if collectorID != "" {
			fmt.Printf("Collector '%s' not found\n", collectorID)
		} else {
			fmt.Println("No collectors found")
		}
		return nil
	}

	for _, collector := range collectors {
		fmt.Printf("\n=== Collector: %s ===\n", collector.CollectorID)
		fmt.Printf("ID: %d\n", collector.ID)
		fmt.Printf("Name: %s\n", collector.Name)
		fmt.Printf("Description: %s\n", collector.Description)
		fmt.Printf("Location: %s\n", collector.Location)
		fmt.Printf("Active: %t\n", collector.IsActive)
		fmt.Printf("Online: %t\n", collector.IsOnline())
		fmt.Printf("Version: %s\n", collector.Version)
		fmt.Printf("IP Address: %s\n", collector.IPAddress)

		if collector.User.Username != "" {
			fmt.Printf("Owner: %s (%s)\n", collector.User.Username, collector.User.FullName)
		} else {
			fmt.Printf("Owner: N/A\n")
		}

		if !collector.LastSeenAt.IsZero() {
			fmt.Printf("Last Seen: %s (%s ago)\n",
				collector.LastSeenAt.Format("2006-01-02 15:04:05"),
				time.Since(collector.LastSeenAt).Round(time.Second))
		} else {
			fmt.Printf("Last Seen: Never\n")
		}

		fmt.Printf("Created: %s\n", collector.CreatedAt.Format("2006-01-02 15:04:05"))

		// Get configuration
		var config model.CollectorConfig
		if err := db.Where("collector_id = ?", collector.CollectorID).First(&config).Error; err == nil {
			fmt.Printf("\n--- Configuration ---\n")
			fmt.Printf("Sample Interval: %d seconds\n", config.SampleInterval)
			fmt.Printf("Upload Interval: %d seconds\n", config.UploadInterval)
			fmt.Printf("Max Cache Size: %d records\n", config.MaxCacheSize)
			fmt.Printf("Auto Upload: %t\n", config.AutoUpload)
			fmt.Printf("Compression Level: %d\n", config.CompressionLevel)
		}

		// Get data count
		var dataCount int64
		db.Model(&model.PowerData{}).Where("collector_id = ?", collector.CollectorID).Count(&dataCount)
		fmt.Printf("Total Data Records: %d\n", dataCount)
	}

	return nil
}

// DeleteCollector deletes a collector
func DeleteCollector(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	collectorID := command.String("id")
	force := command.Bool("force")

	// Find collector
	var collector model.Collector
	if err := db.Where("collector_id = ?", collectorID).First(&collector).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("collector '%s' not found", collectorID)
		}
		return fmt.Errorf("failed to find collector: %v", err)
	}

	// Check data count
	var dataCount int64
	db.Model(&model.PowerData{}).Where("collector_id = ?", collectorID).Count(&dataCount)

	// Confirmation prompt unless force flag is set
	if !force {
		fmt.Printf("Are you sure you want to delete collector '%s'?\n", collectorID)
		if dataCount > 0 {
			fmt.Printf("Warning: This will also delete %d power data records!\n", dataCount)
		}
		fmt.Print("Type 'yes' to confirm: ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete in transaction
	tx := db.Begin()

	// Delete power data
	if err := tx.Where("collector_id = ?", collectorID).Delete(&model.PowerData{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete power data: %v", err)
	}

	// Delete collector config
	if err := tx.Where("collector_id = ?", collectorID).Delete(&model.CollectorConfig{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete collector config: %v", err)
	}

	// Delete collector
	if err := tx.Delete(&collector).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete collector: %v", err)
	}

	tx.Commit()

	fmt.Printf("Collector '%s' and %d data records deleted successfully\n", collectorID, dataCount)
	return nil
}
