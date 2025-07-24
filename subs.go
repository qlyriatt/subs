package subs

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

var db DB
var logger *log.Logger = log.Default()

var ErrNotFound = errors.New("not found in db")

type DB interface {
	Create(sub Sub) (string, error)
	Read(id string) (Sub, error)
	Update(id string, sub Sub) error
	Delete(id string) error
	List() ([]Sub, error)
	Sum(filter Sub) (int, error)
}

type Sub struct {
	ID      string  `json:"sub_id"`
	Service string  `json:"service_name"`
	Price   int     `json:"price"`
	User_ID string  `json:"user_id"`
	Start   string  `json:"start_date"`
	End     *string `json:"end_date"`
}

func (s Sub) String() string {

	str := "{"
	if s.ID != "" {
		str += "ID: " + s.ID + " "
	}

	str += fmt.Sprintf("Service: %v, Price: %v, User_ID: %v, Start: %v", s.Service, s.Price, s.User_ID, s.Start)

	if s.End != nil {
		str += ", End: " + *s.End
	}

	str += "}"

	return str
}

func validateDate(date string) error {

	if date == "" || len(date) != 7 {
		return errors.New("invalid date string")
	}

	ss := strings.Split(date, "-")
	if len(ss) != 2 {
		return errors.New("invalid date string")
	}

	m, err := strconv.Atoi(ss[0])
	if m < 1 || m > 12 || err != nil {
		return errors.New("invalid date string")
	}

	y, err := strconv.Atoi(ss[1])
	if y < 1970 || y > 9999 || err != nil {
		return errors.New("invalid date string")
	}

	return nil
}

func validateSub(sub Sub) error {

	if sub.Service == "" {
		return errors.New("empty service name")
	}

	if sub.Price < 0 {
		return errors.New("invalid price")
	}

	if err := uuid.Validate(sub.User_ID); err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	if err := validateDate(sub.Start); err != nil {
		return errors.New("invalid start period")
	}

	if sub.End != nil {
		if err := validateDate(*sub.End); err != nil {
			return errors.New("invalid end period")
		}
	}

	return nil
}

func validateFilter(filter Sub) error {

	if filter.User_ID != "" {
		if err := uuid.Validate(filter.User_ID); err != nil {
			return fmt.Errorf("invalid user id: %w", err)
		}
	}

	if err := validateDate(filter.Start); err != nil {
		return errors.New("invalid start period")
	}

	if filter.End == nil {
		return errors.New("invalid end period")
	}

	if err := validateDate(*filter.End); err != nil {
		return errors.New("invalid end period")
	}

	return nil
}

func createHandler(w http.ResponseWriter, r *http.Request) {

	var sub Sub
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		logger.Printf("create: resp 400: %v", err)
		http.Error(w, "invalid request: json error", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := validateSub(sub); err != nil {
		logger.Printf("create: resp 400: %v; req %v", err, sub)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, err := db.Create(sub)
	if err != nil {
		logger.Printf("create: resp 500: %v; valid req %v", err, sub)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"sub_id": id})
	logger.Printf("create: resp 201 with %v; req %v", id, sub)
}

func readHandler(w http.ResponseWriter, r *http.Request) {

	id := mux.Vars(r)["id"]
	if err := uuid.Validate(id); err != nil {
		logger.Printf("read: resp 400: %v; req %v", err, id)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	sub, err := db.Read(id)
	if err != nil {
		if err == ErrNotFound {
			logger.Printf("read: resp 404; valid req %v", id)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		logger.Printf("read: resp 500: %v; valid req %v", err, id)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
	logger.Printf("read: resp 200 with %v; req %v", sub, id)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

	id := mux.Vars(r)["id"]
	if err := uuid.Validate(id); err != nil {
		logger.Printf("update: resp 400: %v; req %v", err, id)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var sub Sub
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		logger.Printf("update: resp 400: %v;", err)
		http.Error(w, "invalid request: json error", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := validateSub(sub); err != nil {
		logger.Printf("update: resp 400: %v; req %v, %v", err, id, sub)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.Update(id, sub); err != nil {
		if err == ErrNotFound {
			logger.Printf("read: resp 404; valid req %v", id)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		logger.Printf("update: resp 500: %v; valid req %v, %v", err, id, sub)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	logger.Printf("update: resp 200; req %v, %v", id, sub)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {

	id := mux.Vars(r)["id"]
	if err := uuid.Validate(id); err != nil {
		logger.Printf("delete: resp 400: %v; req %v", err, id)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.Delete(id); err != nil {
		if err == ErrNotFound {
			logger.Printf("read: resp 404; valid req %v", id)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		logger.Printf("delete: resp 500: %v; valid req %v", err, id)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	logger.Printf("delete: resp 204; req %v", id)
}

func listHandler(w http.ResponseWriter, r *http.Request) {

	subs, err := db.List()
	if err != nil {
		logger.Printf("list: resp 500: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
	logger.Printf("list: resp 200 with %v entries", len(subs))
}

func sumHandler(w http.ResponseWriter, r *http.Request) {

	start_date := r.URL.Query().Get("start_date")
	end_date := r.URL.Query().Get("end_date")
	user_id := r.URL.Query().Get("user_id")
	service_name := r.URL.Query().Get("service_name")

	filter := Sub{
		Start:   start_date,
		End:     &end_date,
		User_ID: user_id,
		Service: service_name,
	}
	if err := validateFilter(filter); err != nil {
		logger.Printf("sum: resp 400: %v; req %v", err, filter)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	sum, err := db.Sum(filter)
	if err != nil {
		logger.Printf("sum: resp 500: %v; valid req %v", err, filter)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"sum": sum})
	logger.Printf("sum: resp 200 with %v; req %v", sum, filter)
}

func newRouter() http.Handler {

	r := mux.NewRouter()
	r.HandleFunc("/subs/sum", sumHandler).Methods("GET")
	r.HandleFunc("/subs", createHandler).Methods("POST")
	r.HandleFunc("/subs/{id}", readHandler).Methods("GET")
	r.HandleFunc("/subs/{id}", updateHandler).Methods("PUT")
	r.HandleFunc("/subs/{id}", deleteHandler).Methods("DELETE")
	r.HandleFunc("/subs", listHandler).Methods("GET")

	return r
}

func SetLogger(l *log.Logger) {
	logger = l
}

func Start(database DB) {

	db = database

	r := newRouter()
	r.(*mux.Router).HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./swagger.yaml")
	}).Methods("GET")
	r.(*mux.Router).PathPrefix("/swagger/").Handler(httpSwagger.Handler(httpSwagger.URL("/swagger.yaml")))

	logger.Println("Subs started on port 8080")
	logger.Fatal(http.ListenAndServe(":8080", r))
}
