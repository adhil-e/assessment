package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"go-service/internal/services"
)

// ReportHandler handles HTTP requests for PDF report generation.
type ReportHandler struct {
	studentService *services.StudentService
	pdfService     *services.PDFService
}

// NewReportHandler creates a new ReportHandler with the given services.
func NewReportHandler(studentService *services.StudentService, pdfService *services.PDFService) *ReportHandler {
	return &ReportHandler{
		studentService: studentService,
		pdfService:     pdfService,
	}
}

// AuthInfo holds authentication data to forward to Node.js API.
type AuthInfo struct {
	Cookies   string // Cookie header value
	CSRFToken string // X-CSRF-Token header value
}

// GetStudentReport handles GET /api/v1/students/:id/report
// It fetches student data from Node.js API and generates a PDF report.
func (h *ReportHandler) GetStudentReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentID := vars["id"]

	if studentID == "" {
		http.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}

	// Extract auth info from incoming request to forward to Node.js API.
	// The Node.js backend uses cookie-based auth with CSRF protection.
	authInfo := &services.AuthInfo{
		Cookies:   r.Header.Get("Cookie"),
		CSRFToken: r.Header.Get("X-CSRF-Token"),
	}

	// Fetch student data from Node.js API
	student, err := h.studentService.FetchStudent(studentID, authInfo)
	if err != nil {
		log.Printf("Error fetching student %s: %v", studentID, err)

		switch err.Error() {
		case "student not found":
			http.Error(w, "Student not found", http.StatusNotFound)
		case "unauthorized: please provide valid authentication":
			http.Error(w, "Unauthorized: please provide valid authentication", http.StatusUnauthorized)
		default:
			http.Error(w, "Failed to fetch student data", http.StatusInternalServerError)
		}
		return
	}

	// Generate PDF report
	pdfBytes, err := h.pdfService.GenerateStudentReport(student)
	if err != nil {
		log.Printf("Error generating PDF for student %s: %v", studentID, err)
		http.Error(w, "Failed to generate PDF report", http.StatusInternalServerError)
		return
	}

	// Set response headers for PDF download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=student_%s_report.pdf", studentID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	// Write PDF to response
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(pdfBytes); err != nil {
		log.Printf("Error writing PDF response: %v", err)
	}
}

// HealthCheck handles GET /health
func (h *ReportHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"go-pdf-service"}`))
}
