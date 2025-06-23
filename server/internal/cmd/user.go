package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"text/tabwriter"

	"Power-Monitor/model"

	"github.com/urfave/cli/v3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/gorm"
)

// CreateUserCommand creates a new user
var CreateUserCommand = &cli.Command{
	Name:  "create",
	Usage: "Create a new user",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "username",
			Aliases:  []string{"u"},
			Usage:    "Username for the new user",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "email",
			Aliases:  []string{"e"},
			Usage:    "Email address for the new user",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "password",
			Aliases: []string{"p"},
			Usage:   "Password for the new user (will prompt if not provided)",
		},
		&cli.StringFlag{
			Name:    "fullname",
			Aliases: []string{"f"},
			Usage:   "Full name for the new user",
		},
		&cli.StringFlag{
			Name:    "role",
			Aliases: []string{"r"},
			Usage:   "Role for the new user (admin or user)",
			Value:   "user",
		},
	},
	Action: CreateUser,
}

// ListUsersCommand lists all users
var ListUsersCommand = &cli.Command{
	Name:   "list",
	Usage:  "List all users",
	Action: ListUsers,
}

// UpdateUserCommand updates an existing user
var UpdateUserCommand = &cli.Command{
	Name:  "update",
	Usage: "Update an existing user",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "username",
			Aliases:  []string{"u"},
			Usage:    "Username of the user to update",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "email",
			Aliases: []string{"e"},
			Usage:   "New email address",
		},
		&cli.StringFlag{
			Name:    "fullname",
			Aliases: []string{"f"},
			Usage:   "New full name",
		},
		&cli.StringFlag{
			Name:    "role",
			Aliases: []string{"r"},
			Usage:   "New role (admin or user)",
		},
		&cli.BoolFlag{
			Name:    "active",
			Aliases: []string{"a"},
			Usage:   "Set user as active",
		},
		&cli.BoolFlag{
			Name:    "inactive",
			Aliases: []string{"i"},
			Usage:   "Set user as inactive",
		},
	},
	Action: UpdateUser,
}

// ResetPasswordCommand resets a user's password
var ResetPasswordCommand = &cli.Command{
	Name:  "reset-password",
	Usage: "Reset a user's password",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "username",
			Aliases:  []string{"u"},
			Usage:    "Username of the user",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "password",
			Aliases: []string{"p"},
			Usage:   "New password (will prompt if not provided)",
		},
	},
	Action: ResetPassword,
}

// DeleteUserCommand deletes a user
var DeleteUserCommand = &cli.Command{
	Name:  "delete",
	Usage: "Delete a user",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "username",
			Aliases:  []string{"u"},
			Usage:    "Username of the user to delete",
			Required: true,
		},
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Force delete without confirmation",
		},
	},
	Action: DeleteUser,
}

// CreateUser creates a new user
func CreateUser(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	username := command.String("username")
	email := command.String("email")
	password := command.String("password")
	fullname := command.String("fullname")
	role := command.String("role")

	// Validate role
	if role != "admin" && role != "user" {
		return fmt.Errorf("invalid role: %s (must be 'admin' or 'user')", role)
	}

	// Prompt for password if not provided
	if password == "" {
		fmt.Print("Enter password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %v", err)
		}
		password = string(bytePassword)
		fmt.Println()
	}

	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters long")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	// Create user
	user := model.User{
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
		FullName: fullname,
		Role:     role,
		Active:   true,
	}

	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	fmt.Printf("User '%s' created successfully with ID: %d\n", username, user.ID)
	return nil
}

// ListUsers lists all users
func ListUsers(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %v", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUsername\tEmail\tFull Name\tRole\tActive\tLast Login\tCreated At")
	fmt.Fprintln(w, "----\t--------\t-----\t---------\t----\t------\t----------\t----------")

	for _, user := range users {
		lastLogin := "Never"
		if !user.LastLoginAt.IsZero() {
			lastLogin = user.LastLoginAt.Format("2006-01-02 15:04:05")
		}

		activeStatus := "Yes"
		if !user.Active {
			activeStatus = "No"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			user.ID, user.Username, user.Email, user.FullName,
			user.Role, activeStatus, lastLogin,
			user.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	w.Flush()
	return nil
}

// UpdateUser updates an existing user
func UpdateUser(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	username := command.String("username")
	email := command.String("email")
	fullname := command.String("fullname")
	role := command.String("role")
	active := command.Bool("active")
	inactive := command.Bool("inactive")

	// Find user
	var user model.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user '%s' not found", username)
		}
		return fmt.Errorf("failed to find user: %v", err)
	}

	// Prepare updates
	updates := make(map[string]interface{})

	if email != "" {
		updates["email"] = email
	}
	if fullname != "" {
		updates["full_name"] = fullname
	}
	if role != "" {
		if role != "admin" && role != "user" {
			return fmt.Errorf("invalid role: %s (must be 'admin' or 'user')", role)
		}
		updates["role"] = role
	}
	if active && inactive {
		return fmt.Errorf("cannot set both active and inactive flags")
	}
	if active {
		updates["active"] = true
	}
	if inactive {
		updates["active"] = false
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates specified")
	}

	// Update user
	if err := db.Model(&user).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	fmt.Printf("User '%s' updated successfully\n", username)
	return nil
}

// ResetPassword resets a user's password
func ResetPassword(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	username := command.String("username")
	password := command.String("password")

	// Find user
	var user model.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user '%s' not found", username)
		}
		return fmt.Errorf("failed to find user: %v", err)
	}

	// Prompt for password if not provided
	if password == "" {
		fmt.Print("Enter new password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %v", err)
		}
		password = string(bytePassword)
		fmt.Println()
	}

	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters long")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	// Update password
	if err := db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return fmt.Errorf("failed to update password: %v", err)
	}

	fmt.Printf("Password for user '%s' reset successfully\n", username)
	return nil
}

// DeleteUser deletes a user
func DeleteUser(ctx context.Context, command *cli.Command) error {
	confPath := command.Root().String("config")
	db, err := initDatabase(confPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	username := command.String("username")
	force := command.Bool("force")

	// Find user
	var user model.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user '%s' not found", username)
		}
		return fmt.Errorf("failed to find user: %v", err)
	}

	// Confirmation prompt unless force flag is set
	if !force {
		fmt.Printf("Are you sure you want to delete user '%s'? (y/N): ", username)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete user
	if err := db.Delete(&user).Error; err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	fmt.Printf("User '%s' deleted successfully\n", username)
	return nil
}
