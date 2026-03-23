package settings

import "time"

type UpdateInstitutionRequest struct {
	NameEN  string `json:"name_en" binding:"required"`
	NameKU  string `json:"name_ku"`
	NameAR  string `json:"name_ar"`
	Type    string `json:"type" binding:"required,oneof=public private"`
	Country string `json:"country" binding:"required"`
	Region  string `json:"region"`
}

type UpdateDegreeLabelRequest struct {
	EN    string `json:"en" binding:"required"`
	Local string `json:"local"`
}

type UpdateGradingRequest struct {
	Display string         `json:"display" binding:"required,oneof=numeric letter both"`
	Scale   map[string]int `json:"scale" binding:"required"`
}

type UpdateFeaturesRequest struct {
	CreditsTracking bool `json:"credits_tracking"`
	AllowRetake     bool `json:"allow_retake"`
	AllowPretake    bool `json:"allow_pretake"`
	FullYearRepeat  bool `json:"full_year_repeat"`
	GradeVisibility bool `json:"grade_visibility"`
}

type UpdateAcademicRequest struct {
	SemestersPerYear  int    `json:"semesters_per_year" binding:"required,min=1,max=3"`
	MaxFailureRepeats int    `json:"max_failure_repeats" binding:"min=0"`
	DefaultLanguage   string `json:"default_language" binding:"required,oneof=en ku ar"`
}

type UpdateSettingsRequest struct {
	Institution  *UpdateInstitutionRequest           `json:"institution"`
	DegreeLabels map[string]UpdateDegreeLabelRequest `json:"degree_labels"`
	Grading      *UpdateGradingRequest               `json:"grading"`
	Features     *UpdateFeaturesRequest              `json:"features"`
	Academic     *UpdateAcademicRequest              `json:"academic"`
}

type UpdatePreferencesRequest struct {
	Language           *string `json:"language" binding:"omitempty,oneof=en ku ar"`
	Timezone           *string `json:"timezone"`
	EmailNotifications *bool   `json:"email_notifications"`
	PushNotifications  *bool   `json:"push_notifications"`
}

type SettingsResponse struct {
	Institution  Institution            `json:"institution"`
	DegreeLabels map[string]DegreeLabel `json:"degree_labels"`
	Grading      GradingConfig          `json:"grading"`
	Features     Features               `json:"features"`
	Academic     AcademicConfig         `json:"academic"`
}

type PreferencesResponse struct {
	Language           string    `json:"language"`
	Timezone           string    `json:"timezone"`
	EmailNotifications bool      `json:"email_notifications"`
	PushNotifications  bool      `json:"push_notifications"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type FeaturesResponse struct {
	CreditsTracking bool `json:"credits_tracking"`
	AllowRetake     bool `json:"allow_retake"`
	AllowPretake    bool `json:"allow_pretake"`
	FullYearRepeat  bool `json:"full_year_repeat"`
	GradeVisibility bool `json:"grade_visibility"`
}

func ToSettingsResponse(s *UniversitySettings) SettingsResponse {
	return SettingsResponse{
		Institution:  s.Institution,
		DegreeLabels: s.DegreeLabels,
		Grading:      s.Grading,
		Features:     s.Features,
		Academic:     s.Academic,
	}
}

func ToPreferencesResponse(p *UserPreferences) PreferencesResponse {
	return PreferencesResponse{
		Language:           p.Language,
		Timezone:           p.Timezone,
		EmailNotifications: p.EmailNotifications,
		PushNotifications:  p.PushNotifications,
		UpdatedAt:          p.UpdatedAt,
	}
}

func ToFeaturesResponse(f Features) FeaturesResponse {
	return FeaturesResponse(f)
}

func ToSettingsUpdates(req UpdateSettingsRequest) SettingsUpdates {
	var updates SettingsUpdates

	if req.Institution != nil {
		updates.Institution = &Institution{
			NameEN:  req.Institution.NameEN,
			NameKU:  req.Institution.NameKU,
			NameAR:  req.Institution.NameAR,
			Type:    req.Institution.Type,
			Country: req.Institution.Country,
			Region:  req.Institution.Region,
		}
	}

	if req.DegreeLabels != nil {
		updates.DegreeLabels = make(map[string]DegreeLabel)
		for k, v := range req.DegreeLabels {
			updates.DegreeLabels[k] = DegreeLabel(v)
		}
	}

	if req.Grading != nil {
		updates.Grading = &GradingConfig{
			Display: req.Grading.Display,
			Scale:   req.Grading.Scale,
		}
	}

	if req.Features != nil {
		updates.Features = &Features{
			CreditsTracking: req.Features.CreditsTracking,
			AllowRetake:     req.Features.AllowRetake,
			AllowPretake:    req.Features.AllowPretake,
			FullYearRepeat:  req.Features.FullYearRepeat,
			GradeVisibility: req.Features.GradeVisibility,
		}
	}

	if req.Academic != nil {
		updates.Academic = &AcademicConfig{
			SemestersPerYear:  req.Academic.SemestersPerYear,
			MaxFailureRepeats: req.Academic.MaxFailureRepeats,
			DefaultLanguage:   req.Academic.DefaultLanguage,
		}
	}

	return updates
}

func ToPreferencesUpdates(req UpdatePreferencesRequest) PreferencesUpdates {
	return PreferencesUpdates(req)
}
