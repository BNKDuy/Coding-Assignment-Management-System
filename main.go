package main

import (
	"errors"
	"log"
	"net/http"
	"server/Types"
	"sync"

	"github.com/go-chi/chi/v5"
)

var userData = struct {
	sync.RWMutex
	// Key: username
	// Value: User
	m map[string]Types.User
}{m: make(map[string]Types.User)}

var assignmentData = struct {
	sync.RWMutex
	// Key: ProblemID
	// Value: Path to that Problem
	m map[string]Types.Assignment
}{m: make(map[string]Types.Assignment)}

// Authenticate user with http basic auth
func GetAuthorizedUser(w http.ResponseWriter, r *http.Request) (*Types.User, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, errors.New("unauthorized")
	}

	_ = username
	_ = password

	userData.RLock()
	p, ok := userData.m[username]
	userData.RUnlock()

	if !ok || p.GetPassword() != password {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, errors.New("unauthorized")
	}

	log.Println(username, " authenticated")

	return &p, nil
}

func main() {
	userData.Lock()
	// https://1000randomnames.com
	userData.m["AshlynLambert"] = *Types.NewUser("AshlynLambert", "123", Types.RoleStudent)
	userData.m["GiselleAguilar"] = *Types.NewUser("GiselleAguilar", "123", Types.RoleStudent)
	userData.m["NolaPark"] = *Types.NewUser("NolaPark", "123", Types.RoleStudent)
	userData.m["JonahWagner"] = *Types.NewUser("JonahWagner", "123", Types.RoleStudent)
	userData.m["AlbertGrimes"] = *Types.NewUser("AlbertGrimes", "123", Types.RoleStudent)
	userData.m["SuttonMosley"] = *Types.NewUser("SuttonMosley", "123", Types.RoleStudent)
	userData.m["RobinWhitaker"] = *Types.NewUser("RobinWhitaker", "123", Types.RoleStudent)
	userData.m["AdriannaNelson"] = *Types.NewUser("AdriannaNelson", "123", Types.RoleStudent)
	userData.m["TheaMaynard"] = *Types.NewUser("TheaMaynard", "123", Types.RoleStudent)
	userData.m["KeithTran"] = *Types.NewUser("KeithTran", "123", Types.RoleStudent)
	userData.m["LandryMcKay"] = *Types.NewUser("LandryMcKay", "123", Types.RoleStudent)
	userData.m["FridaMaxwell"] = *Types.NewUser("FridaMaxwell", "123", Types.RoleStudent)
	userData.Unlock()

	// Initialize a new router
	r := chi.NewRouter()

	// Deprecated (this was the old APIs that simply do batch grading)
	// This is kept for my testing of new languages exec only
	r.Post("/api/py", handlePython)
	r.Post("/api/c", handleC)
	r.Post("/api/cpp", handleCPP)
	r.Post("/api/java", handleJava)
	r.Post("/api/go", handleGo)

	// Instructor APIs
	// File: endpointsInstructor.go
	r.Post("/instructors", instructorSignUp)
	r.Post("/assignments", createAssignment)
	r.Get("/assignments/{id}/gradebook", getGradebook)

	// Student APIs
	// File: endpointsStudent.go
	r.Post("/students", studentSignUp)
	r.Post("/assignments/{id}", submitAssignment)

	// Both endpoint
	// File: endpointsShared.go
	r.Get("/assignments", listAssignments)
	r.Get("/assignments/{id}", getAssignment)
	r.Get("/assignments/{id}/submissions", getSubmissions)

	// Listen and Server on port 3000
	log.Println("Listening on port 3000 with TLS")
	err := http.ListenAndServeTLS(":3000", "cert.pem", "key.pem", r)

	// If something is wrong, fall back to listen without TLS
	if err != nil {
		log.Println("Failed to set up TLS")
		log.Println("Listening on port 3000 without TLS")
		http.ListenAndServe(":3000", r)
	}
}
