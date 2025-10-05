package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EmployeeDetails maps the aggregation result.
// Use lowercase bson tags to match typical DB fields (ensure they match your DB).
type EmployeeDetails struct {
	EmpID      interface{} `bson:"emp_id" json:"emp_id"`
	EmpName    interface{} `bson:"emp_name" json:"emp_name"`
	Department interface{} `bson:"department" json:"department"`
	Language   interface{} `bson:"language" json:"language"`
}

var mongoClient *mongo.Client
var dbName string

func main() {

	fmt.Println("Starting Employee Management Backend...")
	// load .env (optional — won't fail the run)
	_ = godotenv.Load() // ignore error; prefer system env in some CI

	// ✅ Hardcode your MongoDB URI and DB name here
	mongoURI := "mongodb+srv://Karthikeyan:Hema%401199@mycluster.5oolqvy.mongodb.net/"
	dbName = "my_db"

	fmt.Println("Mongo URI:", mongoURI)
	fmt.Println("Database Name:", dbName)
	// allowed origin for CORS (still configurable via env if needed)
	allowedOrigin := os.Getenv("ALLOW_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}

	// PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// connect to mongo
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var err error
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongo connect error: %v", err)
	}
	if err = mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("mongo ping error: %v", err)
	}
	log.Printf("Connected to MongoDB: %s\n", dbName)

	// ensure disconnect on exit
	defer func() {
		_ = mongoClient.Disconnect(context.Background())
	}()

	// TODO: start your server here (router, handlers, etc.)
	fmt.Printf("Server running on port %s with allowed origin %s\n", port, allowedOrigin)

	// ----- Gin setup -----
	gin.SetMode(gin.ReleaseMode) // set Release for less noisy logs in prod; use env to control
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	// Set trusted proxies (tighten for production; here we set localhost)
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		log.Printf("warning: set trusted proxies: %v", err)
	}

	// CORS (allow typical headers + Authorization)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// small helper to avoid favicon 404 noise
	r.GET("/favicon.ico", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	// routes
	r.GET("/api/employees", getEmployees)
	r.GET("/api/employees/last-id", getLastEmpID)
	r.POST("/api/employees/create", createEmployee)
	r.PUT("/api/employees/:id", updateEmployee)
	r.DELETE("/api/employees/:id", deleteEmployee)

	// graceful shutdown
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// start server in goroutine
	go func() {
		log.Printf("Server listening on :%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server listen error: %v", err)
		}
	}()

	// wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("Server exiting gracefully")
}

// helper to get a collection
func coll(name string) *mongo.Collection {
	return mongoClient.Database(dbName).Collection(name)
}

// getEmployees: aggregation endpoint (fixed pipeline with proper bson.D)
func getEmployees(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := coll("Employee")

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "Department"},
			{Key: "localField", Value: "emp_id"},
			{Key: "foreignField", Value: "emp_id"},
			{Key: "as", Value: "departments"},
		}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "Developers"},
			{Key: "localField", Value: "emp_id"},
			{Key: "foreignField", Value: "emp_id"},
			{Key: "as", Value: "languages"},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "emp_id", Value: 1},
			{Key: "emp_name", Value: 1},
			{Key: "department", Value: bson.D{
				{Key: "$arrayElemAt", Value: bson.A{"$departments.department_name", 0}},
			}},
			{Key: "language", Value: bson.D{
				{Key: "$arrayElemAt", Value: bson.A{"$languages.language", 0}},
			}},
		}}},
	}

	cur, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "aggregate: " + err.Error()})
		return
	}
	defer cur.Close(ctx)

	var results []EmployeeDetails
	if err := cur.All(ctx, &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cursor all: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}

func getLastEmpID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := coll("Employee")
	opts := options.FindOne().SetSort(bson.D{{Key: "emp_id", Value: -1}})
	var last bson.M
	err := collection.FindOne(ctx, bson.D{}, opts).Decode(&last)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusOK, gin.H{"last_emp_id": 0})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	lastId := 0
	if v, ok := last["emp_id"]; ok {
		switch t := v.(type) {
		case int32:
			lastId = int(t)
		case int64:
			lastId = int(t)
		case float64:
			lastId = int(t)
		}
	}
	c.JSON(http.StatusOK, gin.H{"last_emp_id": lastId})
}

func createEmployee(c *gin.Context) {
	var input struct {
		EmpId      int    `json:"emp_id" binding:"required"`
		EmpName    string `json:"emp_name" binding:"required"`
		Department string `json:"department"`
		Language   string `json:"language"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := mongoClient.Database(dbName)

	// Consider using a transaction if atomicity is required across collections.
	if _, err := db.Collection("Employee").InsertOne(ctx, bson.M{"emp_id": input.EmpId, "emp_name": input.EmpName}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := db.Collection("Department").InsertOne(ctx, bson.M{"emp_id": input.EmpId, "department_name": input.Department}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := db.Collection("Developers").InsertOne(ctx, bson.M{"emp_id": input.EmpId, "language": input.Language}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Employee created successfully"})
}

func updateEmployee(c *gin.Context) {
	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	empId, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id; must be numeric"})
		return
	}

	var input struct {
		EmpName    *string `json:"emp_name"`
		Department *string `json:"department"`
		Language   *string `json:"language"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := mongoClient.Database(dbName)

	if input.EmpName != nil {
		if _, err := db.Collection("Employee").UpdateOne(ctx, bson.M{"emp_id": empId}, bson.M{"$set": bson.M{"emp_name": *input.EmpName}}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update employee: " + err.Error()})
			return
		}
	}
	if input.Department != nil {
		if _, err := db.Collection("Department").UpdateOne(ctx, bson.M{"emp_id": empId}, bson.M{"$set": bson.M{"department_name": *input.Department}}, options.Update().SetUpsert(true)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update department: " + err.Error()})
			return
		}
	}
	if input.Language != nil {
		if _, err := db.Collection("Developers").UpdateOne(ctx, bson.M{"emp_id": empId}, bson.M{"$set": bson.M{"language": *input.Language}}, options.Update().SetUpsert(true)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update developers: " + err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Employee updated successfully"})
}

func deleteEmployee(c *gin.Context) {
	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	empId, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id; must be numeric"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := mongoClient.Database(dbName)

	res, err := db.Collection("Employee").DeleteOne(ctx, bson.M{"emp_id": empId})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete employee: " + err.Error()})
		return
	}
	if _, err := db.Collection("Department").DeleteMany(ctx, bson.M{"emp_id": empId}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete department: " + err.Error()})
		return
	}
	if _, err := db.Collection("Developers").DeleteMany(ctx, bson.M{"emp_id": empId}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete developers: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted successfully", "deleted_count": res.DeletedCount})
}
