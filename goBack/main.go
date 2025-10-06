package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EmployeeDetails returned by aggregation
type EmployeeDetails struct {
	EmpID      interface{} `bson:"emp_id" json:"emp_id"`
	EmpName    interface{} `bson:"emp_name" json:"emp_name"`
	Department interface{} `bson:"department" json:"department"`
	Language   interface{} `bson:"language" json:"language"`
}

var (
	client    *mongo.Client
	dbName    string
	idCounter = 1
	idMu      sync.Mutex
)

// helper to get collection
func coll(name string) *mongo.Collection {
	return client.Database(dbName).Collection(name)
}

// set minimal CORS headers (colleague-style)
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// nextID returns thread-safe sequential id
func nextID() int {
	idMu.Lock()
	defer idMu.Unlock()
	v := idCounter
	idCounter++
	return v
}

// initIDCounter reads the highest emp_id and sets idCounter = max+1
func initIDCounter(ctx context.Context) {
	collection := coll("Employee")
	opts := options.FindOne().SetSort(bson.D{{Key: "emp_id", Value: -1}})
	var last bson.M
	err := collection.FindOne(ctx, bson.D{}, opts).Decode(&last)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			idCounter = 1
			log.Println("No employees found. Starting IDs from 1.")
			return
		}
		log.Printf("initIDCounter: error reading last id: %v\n", err)
		return
	}
	if v, ok := last["emp_id"]; ok {
		switch t := v.(type) {
		case int32:
			idCounter = int(t) + 1
		case int64:
			idCounter = int(t) + 1
		case float64:
			idCounter = int(t) + 1
		case int:
			idCounter = t + 1
		default:
			idCounter = 1
		}
	}
	log.Printf("Initialized ID counter. Starting from %d\n", idCounter)
}

// ---------------- Handlers ----------------

// employeesHandler handles GET (aggregate) and POST (create) on /api/employees
func employeesHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	switch r.Method {
	case http.MethodGet:
		getEmployees(w, r)
	case http.MethodPost:
		createEmployee(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getEmployees runs aggregation joining Department and Developers and projects fields
func getEmployees(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "aggregate: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	var results []EmployeeDetails
	if err := cur.All(ctx, &results); err != nil {
		http.Error(w, "cursor all: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

// lastIDHandler returns the highest emp_id
func lastIDHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := coll("Employee")
	opts := options.FindOne().SetSort(bson.D{{Key: "emp_id", Value: -1}})
	var last bson.M
	err := collection.FindOne(ctx, bson.D{}, opts).Decode(&last)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(bson.M{"last_emp_id": 0})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		case int:
			lastId = t
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bson.M{"last_emp_id": lastId})
}

// createEmployee handles POST to /api/employees or /api/employees/create
func createEmployee(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		EmpId      int    `json:"emp_id"`
		EmpName    string `json:"emp_name"`
		Department string `json:"department"`
		Language   string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}
	// assign id if not provided
	if input.EmpId == 0 {
		input.EmpId = nextID()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := client.Database(dbName)
	if _, err := db.Collection("Employee").InsertOne(ctx, bson.M{"emp_id": input.EmpId, "emp_name": input.EmpName}); err != nil {
		http.Error(w, "insert employee: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := db.Collection("Department").InsertOne(ctx, bson.M{"emp_id": input.EmpId, "department_name": input.Department}); err != nil {
		http.Error(w, "insert department: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := db.Collection("Developers").InsertOne(ctx, bson.M{"emp_id": input.EmpId, "language": input.Language}); err != nil {
		http.Error(w, "insert developers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(bson.M{"message": "Employee created successfully", "emp_id": input.EmpId})
}

// empByIDHandler handles PUT and DELETE for /api/employees/{id}
func empByIDHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	// path: /api/employees/{id}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/employees/")
	if idStr == "" {
		http.Error(w, "id required in path", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		updateEmployee(w, r, id)
	case http.MethodDelete:
		deleteEmployee(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// updateEmployee updates Employee / Department / Developers (upsert where reasonable)
func updateEmployee(w http.ResponseWriter, r *http.Request, empId int) {
	var input struct {
		EmpName    *string `json:"emp_name"`
		Department *string `json:"department"`
		Language   *string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := client.Database(dbName)

	if input.EmpName != nil {
		if _, err := db.Collection("Employee").UpdateOne(ctx, bson.M{"emp_id": empId}, bson.M{"$set": bson.M{"emp_name": *input.EmpName}}); err != nil {
			http.Error(w, "update employee: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if input.Department != nil {
		if _, err := db.Collection("Department").UpdateOne(ctx, bson.M{"emp_id": empId}, bson.M{"$set": bson.M{"department_name": *input.Department}}, options.Update().SetUpsert(true)); err != nil {
			http.Error(w, "update department: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if input.Language != nil {
		if _, err := db.Collection("Developers").UpdateOne(ctx, bson.M{"emp_id": empId}, bson.M{"$set": bson.M{"language": *input.Language}}, options.Update().SetUpsert(true)); err != nil {
			http.Error(w, "update developers: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bson.M{"message": "Employee updated successfully"})
}

// deleteEmployee deletes Employee and related records
func deleteEmployee(w http.ResponseWriter, r *http.Request, empId int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := client.Database(dbName)

	res, err := db.Collection("Employee").DeleteOne(ctx, bson.M{"emp_id": empId})
	if err != nil {
		http.Error(w, "delete employee: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := db.Collection("Department").DeleteMany(ctx, bson.M{"emp_id": empId}); err != nil {
		http.Error(w, "delete department: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := db.Collection("Developers").DeleteMany(ctx, bson.M{"emp_id": empId}); err != nil {
		http.Error(w, "delete developers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bson.M{"message": "Employee deleted successfully", "deleted_count": res.DeletedCount})
}

func main() {
	// read from env or fall back to defaults
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		// change this default to your cluster if needed
		mongoURI = "mongodb+srv://Karthikeyan:Hema%401199@mycluster.5oolqvy.mongodb.net/"
	}
	dbName = os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "my_db"
	}

	// connect to mongo
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongo connect error: %v", err)
	}
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatalf("mongo ping error: %v", err)
	}
	log.Printf("Connected to MongoDB: %s\n", dbName)

	// initialize id counter (colleague-style)
	initIDCounter(ctx)

	// routes (plain net/http)
	http.HandleFunc("/api/employees", employeesHandler)      // GET / POST
	http.HandleFunc("/api/employees/create", createEmployee) // POST alias
	http.HandleFunc("/api/employees/last-id", lastIDHandler) // GET
	http.HandleFunc("/api/employees/", empByIDHandler)       // PUT / DELETE by id

	// static SPA serving (like colleague)
	fs := http.FileServer(http.Dir("./frontend/dist"))
	http.Handle("/assets/", fs)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// if file exists in dist serve it; else serve index.html
		if _, err := os.Stat("./frontend/dist" + r.URL.Path); err == nil {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, "./frontend/dist/index.html")
	})

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
