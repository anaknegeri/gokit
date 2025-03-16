package main

import (
	"log"

	"github.com/anaknegeri/gokit"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Role represents a user role
type Role struct {
	ID          string       `json:"ID" gorm:"primaryKey;type:uuid"`
	Name        string       `json:"Name" gorm:"uniqueIndex;not null"`
	Description string       `json:"Description"`
	CreatedAt   string       `json:"CreatedAt"`
	UpdatedAt   string       `json:"UpdatedAt"`
	Users       []User       `json:"Users,omitempty" gorm:"many2many:user_roles;"`
	Permissions []Permission `json:"Permissions,omitempty" gorm:"many2many:role_permissions;"`
}

// User represents a user in the system
type User struct {
	ID    string `json:"ID" gorm:"primaryKey;type:uuid"`
	Name  string `json:"Name"`
	Email string `json:"Email" gorm:"uniqueIndex;not null"`
	Roles []Role `json:"Roles,omitempty" gorm:"many2many:user_roles;"`
}

// Permission represents a system permission
type Permission struct {
	ID          string `json:"ID" gorm:"primaryKey;type:uuid"`
	Name        string `json:"Name" gorm:"uniqueIndex;not null"`
	Description string `json:"Description"`
	Roles       []Role `json:"Roles,omitempty" gorm:"many2many:role_permissions;"`
}

func main() {
	// Initialize database with SQLite
	db, err := gorm.Open(sqlite.Open("roles.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate the schema
	db.AutoMigrate(&Role{}, &User{}, &Permission{})

	// Create sample roles if none exist
	var count int64
	db.Model(&Role{}).Count(&count)
	if count == 0 {
		createSampleRoles(db)
	}

	// Create paginator
	paginator := gokit.NewPaginator(db)

	// Initialize Fiber app
	app := fiber.New()

	// Routes
	app.Get("/api/roles", func(c *fiber.Ctx) error {
		// Get pagination params from query
		params := gokit.PaginationParams{
			Page:     c.QueryInt("page", 1),
			PageSize: c.QueryInt("pageSize", 10),
		}

		// Get roles with pagination
		var roles []Role
		result, err := paginator.Paginate(params, &roles)
		if err != nil {
			return gokit.ErrorResponseWithErr(c, err)
		}

		// Return paginated response
		return gokit.SuccessWithPagination(c, "Roles retrieved successfully", result)
	})

	// Start server
	log.Println("Server started on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}

// Create sample roles
func createSampleRoles(db *gorm.DB) {
	roles := []Role{
		{
			ID:          uuid.New().String(),
			Name:        "Administrator",
			Description: "Full system access with all permissions",
			CreatedAt:   "2025-03-08T21:44:34Z",
			UpdatedAt:   "2025-03-08T21:44:34Z",
		},
		{
			ID:          uuid.New().String(),
			Name:        "Event Organizer",
			Description: "Can create and manage events, tickets, and view reports",
			CreatedAt:   "2025-03-08T21:46:58Z",
			UpdatedAt:   "2025-03-08T21:46:58Z",
		},
		{
			ID:          uuid.New().String(),
			Name:        "Customer",
			Description: "Regular user account for purchasing tickets",
			CreatedAt:   "2025-03-08T21:48:12Z",
			UpdatedAt:   "2025-03-08T21:48:12Z",
		},
		{
			ID:          uuid.New().String(),
			Name:        "Content Manager",
			Description: "Can create and edit content on the platform",
			CreatedAt:   "2025-03-08T21:49:30Z",
			UpdatedAt:   "2025-03-08T21:49:30Z",
		},
		{
			ID:          uuid.New().String(),
			Name:        "Analyst",
			Description: "Read-only access to reports and analytics",
			CreatedAt:   "2025-03-08T21:50:45Z",
			UpdatedAt:   "2025-03-08T21:50:45Z",
		},
	}

	for _, role := range roles {
		db.Create(&role)
	}
}
