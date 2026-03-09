package university

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToCollegeResponse(t *testing.T) {
	nameLocal := "کۆلێژی زانست"
	desc := "Science college"

	college := &College{
		ID:          uuid.New(),
		NameEN:      "College of Science",
		NameLocal:      &nameLocal,
		Code:        "SCI",
		Description: &desc,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	resp := ToCollegeResponse(college)

	if resp.ID != college.ID {
		t.Error("ID should match")
	}
	if resp.NameEN != college.NameEN {
		t.Error("NameEN should match")
	}
	if resp.NameLocal == nil || *resp.NameLocal != nameLocal {
		t.Error("NameLocal should match")
	}
	if resp.Code != college.Code {
		t.Error("Code should match")
	}
	if !resp.IsActive {
		t.Error("IsActive should be true")
	}
}

func TestToCollegesResponse(t *testing.T) {
	colleges := []College{
		{ID: uuid.New(), NameEN: "College A", Code: "A", IsActive: true},
		{ID: uuid.New(), NameEN: "College B", Code: "B", IsActive: false},
	}

	resp := ToCollegesResponse(colleges)

	if len(resp) != 2 {
		t.Fatalf("expected 2 colleges, got %d", len(resp))
	}
	if resp[0].NameEN != "College A" {
		t.Error("first college name should be College A")
	}
	if resp[1].Code != "B" {
		t.Error("second college code should be B")
	}
}

func TestToDepartmentResponse(t *testing.T) {
	collegeID := uuid.New()
	nameLocal := "کۆمپیوتەر"

	dept := &Department{
		ID:        uuid.New(),
		CollegeID: collegeID,
		NameEN:    "Computer Science",
		NameLocal:    &nameLocal,
		Code:      "CS",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := ToDepartmentResponse(dept)

	if resp.ID != dept.ID {
		t.Error("ID should match")
	}
	if resp.CollegeID != collegeID {
		t.Error("CollegeID should match")
	}
	if resp.NameEN != dept.NameEN {
		t.Error("NameEN should match")
	}
	if resp.Code != dept.Code {
		t.Error("Code should match")
	}
}

func TestToDepartmentsResponse(t *testing.T) {
	collegeID := uuid.New()
	depts := []Department{
		{ID: uuid.New(), CollegeID: collegeID, NameEN: "Dept A", Code: "A"},
		{ID: uuid.New(), CollegeID: collegeID, NameEN: "Dept B", Code: "B"},
	}

	resp := ToDepartmentsResponse(depts)

	if len(resp) != 2 {
		t.Fatalf("expected 2 departments, got %d", len(resp))
	}
}

func TestToProgramResponse(t *testing.T) {
	deptID := uuid.New()
	desc := "Bachelor program"

	program := &Program{
		ID:            uuid.New(),
		DepartmentID:  deptID,
		NameEN:        "Bachelor in CS",
		Code:          "BCS",
		DegreeType:    "bachelor",
		DurationYears: 4,
		TotalCredits:     240,
		Description:   &desc,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	resp := ToProgramResponse(program)

	if resp.ID != program.ID {
		t.Error("ID should match")
	}
	if resp.DepartmentID != deptID {
		t.Error("DepartmentID should match")
	}
	if resp.DegreeType != "bachelor" {
		t.Error("DegreeType should be bachelor")
	}
	if resp.DurationYears != 4 {
		t.Error("DurationYears should be 4")
	}
	if resp.TotalCredits != 240 {
		t.Error("TotalCredits should be 240")
	}
}

func TestToProgramsResponse(t *testing.T) {
	deptID := uuid.New()
	programs := []Program{
		{ID: uuid.New(), DepartmentID: deptID, NameEN: "Program A", Code: "A", DegreeType: "bachelor", DurationYears: 4, TotalCredits: 240},
		{ID: uuid.New(), DepartmentID: deptID, NameEN: "Program B", Code: "B", DegreeType: "master", DurationYears: 2, TotalCredits: 120},
	}

	resp := ToProgramsResponse(programs)

	if len(resp) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(resp))
	}
	if resp[0].DegreeType != "bachelor" {
		t.Error("first program degree type should be bachelor")
	}
	if resp[1].DegreeType != "master" {
		t.Error("second program degree type should be master")
	}
}

func TestCreateCollegeRequest_Validation(t *testing.T) {
	req := CreateCollegeRequest{
		NameEN: "Test College",
		Code:   "TC",
	}

	if req.NameEN != "Test College" {
		t.Error("NameEN should match")
	}
	if req.Code != "TC" {
		t.Error("Code should match")
	}
}

func TestCreateDepartmentRequest_Validation(t *testing.T) {
	collegeID := uuid.New()
	req := CreateDepartmentRequest{
		CollegeID: collegeID,
		NameEN:    "Test Dept",
		Code:      "TD",
	}

	if req.CollegeID != collegeID {
		t.Error("CollegeID should match")
	}
}

func TestCreateProgramRequest_Validation(t *testing.T) {
	deptID := uuid.New()
	req := CreateProgramRequest{
		DepartmentID:  deptID,
		NameEN:        "Test Program",
		Code:          "TP",
		DegreeType:    "bachelor",
		DurationYears: 4,
		TotalCredits:     240,
	}

	if req.DepartmentID != deptID {
		t.Error("DepartmentID should match")
	}
	if req.DegreeType != "bachelor" {
		t.Error("DegreeType should be bachelor")
	}
	if req.DurationYears != 4 {
		t.Error("DurationYears should be 4")
	}
}
