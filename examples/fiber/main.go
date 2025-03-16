package main

import (
	"context"
	"log"
	"os"

	"github.com/anaknegeri/gokit"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model for demonstration
type User struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Age       int    `json:"age" validate:"gte=18,lte=120"`
}

func main() {
	// Setup environment variables for filesystem
	os.Setenv("STORAGE_TYPE", "local")
	os.Setenv("UPLOAD_STORAGE_PATH", "./uploads")
	os.Setenv("UPLOAD_MAX_SIZE", "20") // 20MB
	os.Setenv("ALLOWED_FILE_TYPES", ".jpg,.jpeg,.png,.gif,.pdf,.doc,.docx,.xls,.xlsx")

	// Initialize logger
	customLogger := gokit.InitLogger()

	// Initialize context
	ctx := context.Background()

	// Initialize filesystem
	fs, err := gokit.NewFilesystem(ctx)
	if err != nil {
		customLogger.Fatalf("Failed to initialize filesystem: %v", err)
	}

	// Initialize database for pagination example
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		customLogger.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&User{})
	if err != nil {
		customLogger.Fatalf("Failed to migrate database: %v", err)
	}

	// Create sample users
	createSampleUsers(db)

	// Initialize validator
	validate := gokit.NewValidator()

	// Initialize paginator
	paginator := gokit.NewPaginator(db)

	// Create fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: 30 * 1024 * 1024, // 30MB
	})

	// Add middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// API routes
	api := app.Group("/api")

	// File routes
	fileAPI := api.Group("/files")
	fileAPI.Post("/upload", fs.GetUploadHandler()("files").(fiber.Handler))
	fileAPI.Get("/info/*", fs.GetFileInfoHandler()("files").(fiber.Handler))
	fileAPI.Get("/*", fs.GetFileHandler()("files").(fiber.Handler))
	fileAPI.Delete("/*", fs.GetDeleteFileHandler()("files").(fiber.Handler))
	fileAPI.Get("/", fs.GetListFilesHandler()("files").(fiber.Handler))

	// User routes
	userAPI := api.Group("/users")

	// List users with pagination
	userAPI.Get("/", func(c *fiber.Ctx) error {
		// Get pagination params from query
		params := gokit.PaginationParams{
			Page:     c.QueryInt("page", 1),
			PageSize: c.QueryInt("pageSize", 10),
		}

		// Get users with pagination
		var users []User
		result, err := paginator.Paginate(params, &users)
		if err != nil {
			customLogger.Errorf("Failed to get users: %v", err)
			return gokit.ErrorResponseWithErr(c, gokit.WrapError(
				err,
				fiber.StatusInternalServerError,
				"Failed to get users",
			))
		}

		return gokit.SuccessResponse(c, "Users retrieved successfully", result)
	})

	// Get user by ID
	userAPI.Get("/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return gokit.BadRequestResponse(c, "Invalid user ID", nil)
		}

		var user User
		if err := db.First(&user, id).Error; err != nil {
			customLogger.Warnf("User not found: %v", err)
			return gokit.NotFoundResponse(c, "User not found")
		}

		return gokit.SuccessResponse(c, "User retrieved successfully", user)
	})

	// Create user with validation
	userAPI.Post("/", func(c *fiber.Ctx) error {
		var user User

		// Parse request body
		if err := c.BodyParser(&user); err != nil {
			return gokit.BadRequestResponse(c, "Invalid request body", nil)
		}

		// Validate user
		if err := validate.Struct(user); err != nil {
			return gokit.ErrorResponseWithErr(c, gokit.ValidatorError(err))
		}

		// Save user
		if err := db.Create(&user).Error; err != nil {
			customLogger.Errorf("Failed to create user: %v", err)
			return gokit.ErrorResponseWithErr(c, gokit.WrapError(
				err,
				fiber.StatusInternalServerError,
				"Failed to create user",
			))
		}

		return gokit.CreatedResponse(c, "User created successfully", user)
	})

	// Start server
	customLogger.Info("Starting server on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}

// Create sample users for pagination example
func createSampleUsers(db *gorm.DB) {
	// Check if we already have users
	var count int64
	db.Model(&User{}).Count(&count)
	if count > 0 {
		return
	}

	// Create sample users
	users := []User{
		{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Age: 30},
		{FirstName: "Jane", LastName: "Doe", Email: "jane.doe@example.com", Age: 25},
		{FirstName: "Bob", LastName: "Smith", Email: "bob.smith@example.com", Age: 40},
		{FirstName: "Alice", LastName: "Johnson", Email: "alice.johnson@example.com", Age: 35},
		{FirstName: "Charlie", LastName: "Brown", Email: "charlie.brown@example.com", Age: 50},
	}

	for _, user := range users {
		db.Create(&user)
	}
}
