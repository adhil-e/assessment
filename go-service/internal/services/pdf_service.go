package services

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"go-service/internal/models"
)

type PDFService struct{}

func NewPDFService() *PDFService {
	return &PDFService{}
}

// GenerateStudentReport generates a PDF report for the given student
func (s *PDFService) GenerateStudentReport(student *models.Student) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// Header
	pdf.SetFont("Arial", "B", 20)
	pdf.SetTextColor(33, 37, 41)
	pdf.CellFormat(0, 12, "Student Report", "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Generation timestamp
	pdf.SetFont("Arial", "I", 10)
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated on: %s", time.Now().Format("January 2, 2006 at 3:04 PM")), "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Student Basic Info Section
	s.addSectionHeader(pdf, "Personal Information")
	s.addField(pdf, "Student ID", fmt.Sprintf("%d", student.ID))
	s.addField(pdf, "Full Name", student.Name)
	s.addField(pdf, "Email", student.Email)
	s.addField(pdf, "Phone", student.Phone)
	s.addField(pdf, "Gender", student.Gender)
	s.addField(pdf, "Date of Birth", formatDate(student.DOB))
	pdf.Ln(5)

	// Academic Info Section
	s.addSectionHeader(pdf, "Academic Information")
	s.addField(pdf, "Class", student.Class)
	s.addField(pdf, "Section", student.Section)
	s.addField(pdf, "Roll Number", fmt.Sprintf("%d", student.Roll))
	s.addField(pdf, "Admission Date", formatDate(student.AdmissionDate))
	s.addField(pdf, "Class Teacher", student.ReporterName)
	systemAccessStatus := "Inactive"
	if student.SystemAccess {
		systemAccessStatus = "Active"
	}
	s.addField(pdf, "System Access", systemAccessStatus)
	pdf.Ln(5)

	// Parent/Guardian Info Section
	s.addSectionHeader(pdf, "Parent/Guardian Information")
	if student.FatherName != "" {
		s.addField(pdf, "Father's Name", student.FatherName)
		if student.FatherPhone != "" {
			s.addField(pdf, "Father's Phone", student.FatherPhone)
		}
	}
	if student.MotherName != "" {
		s.addField(pdf, "Mother's Name", student.MotherName)
		if student.MotherPhone != "" {
			s.addField(pdf, "Mother's Phone", student.MotherPhone)
		}
	}
	if student.GuardianName != "" {
		s.addField(pdf, "Guardian's Name", student.GuardianName)
		s.addField(pdf, "Guardian's Phone", student.GuardianPhone)
		s.addField(pdf, "Relation", student.RelationOfGuardian)
	}
	pdf.Ln(5)

	// Address Section
	s.addSectionHeader(pdf, "Address Information")
	s.addField(pdf, "Current Address", student.CurrentAddress)
	s.addField(pdf, "Permanent Address", student.PermanentAddress)

	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(0, 10, "This is a system-generated report from the Student Management System", "", 0, "C", false, 0, "")

	// Output to bytes
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *PDFService) addSectionHeader(pdf *gofpdf.Fpdf, title string) {
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(0, 123, 255)
	pdf.CellFormat(0, 10, title, "", 1, "L", false, 0, "")
	pdf.SetDrawColor(0, 123, 255)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(3)
}

func (s *PDFService) addField(pdf *gofpdf.Fpdf, label, value string) {
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(33, 37, 41)
	pdf.CellFormat(60, 7, label+":", "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(73, 80, 87)
	if value == "" {
		value = "N/A"
	}
	pdf.CellFormat(0, 7, value, "", 1, "L", false, 0, "")
}

func formatDate(dateStr string) string {
	if dateStr == "" {
		return "N/A"
	}

	// Try parsing various date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("January 2, 2006")
		}
	}

	return dateStr
}
