package settings

import "time"

type UpdateInstitutionRequest struct {
	Name          map[string]string `json:"name" binding:"required"`
	Type          string            `json:"type" binding:"required,oneof=public private"`
	Country       string            `json:"country" binding:"required"`
	Region        string            `json:"region"`
	Accreditation string            `json:"accreditation"`
	Founded       int               `json:"founded"`
	About         map[string]string `json:"about"`
	Address       string            `json:"address"`
	Phone         string            `json:"phone"`
	Email         string            `json:"email"`
	Website       string            `json:"website"`
	LogoURL       string            `json:"logo_url"`
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
	CreditsTracking       bool `json:"credits_tracking"`
	AllowRetake           bool `json:"allow_retake"`
	AllowPretake          bool `json:"allow_pretake"`
	FullYearRepeat        bool `json:"full_year_repeat"`
	GradeVisibility       bool `json:"grade_visibility"`
	ShowUniversityMembers bool `json:"show_university_members"`
	ShowCourseMembers     bool `json:"show_course_members"`
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

// SettingsResponse is for admin endpoints (all languages)
type SettingsResponse struct {
	Institution  Institution            `json:"institution"`
	DegreeLabels map[string]DegreeLabel `json:"degree_labels"`
	Grading      GradingConfig          `json:"grading"`
	Features     Features               `json:"features"`
	Academic     AcademicConfig         `json:"academic"`
}

// InstitutionPublicResponse is for public endpoints (single language)
type InstitutionPublicResponse struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Country       string `json:"country"`
	Region        string `json:"region,omitempty"`
	Accreditation string `json:"accreditation,omitempty"`
	Founded       int    `json:"founded,omitempty"`
	About         string `json:"about,omitempty"`
	Address       string `json:"address,omitempty"`
	Phone         string `json:"phone,omitempty"`
	Email         string `json:"email,omitempty"`
	Website       string `json:"website,omitempty"`
	LogoURL       string `json:"logo_url,omitempty"`
}

type PreferencesResponse struct {
	Language           string    `json:"language"`
	Timezone           string    `json:"timezone"`
	EmailNotifications bool      `json:"email_notifications"`
	PushNotifications  bool      `json:"push_notifications"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type FeaturesResponse struct {
	CreditsTracking       bool `json:"credits_tracking"`
	AllowRetake           bool `json:"allow_retake"`
	AllowPretake          bool `json:"allow_pretake"`
	FullYearRepeat        bool `json:"full_year_repeat"`
	GradeVisibility       bool `json:"grade_visibility"`
	ShowUniversityMembers bool `json:"show_university_members"`
	ShowCourseMembers     bool `json:"show_course_members"`
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

func ToInstitutionPublicResponse(i Institution, lang string) InstitutionPublicResponse {
	return InstitutionPublicResponse{
		Name:          i.GetName(lang),
		Type:          i.Type,
		Country:       i.Country,
		Region:        i.Region,
		Accreditation: i.Accreditation,
		Founded:       i.Founded,
		About:         i.GetAbout(lang),
		Address:       i.Address,
		Phone:         i.Phone,
		Email:         i.Email,
		Website:       i.Website,
		LogoURL:       i.LogoURL,
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
			Name:          req.Institution.Name,
			Type:          req.Institution.Type,
			Country:       req.Institution.Country,
			Region:        req.Institution.Region,
			Accreditation: req.Institution.Accreditation,
			Founded:       req.Institution.Founded,
			About:         req.Institution.About,
			Address:       req.Institution.Address,
			Phone:         req.Institution.Phone,
			Email:         req.Institution.Email,
			Website:       req.Institution.Website,
			LogoURL:       req.Institution.LogoURL,
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
			CreditsTracking:       req.Features.CreditsTracking,
			AllowRetake:           req.Features.AllowRetake,
			AllowPretake:          req.Features.AllowPretake,
			FullYearRepeat:        req.Features.FullYearRepeat,
			GradeVisibility:       req.Features.GradeVisibility,
			ShowUniversityMembers: req.Features.ShowUniversityMembers,
			ShowCourseMembers:     req.Features.ShowCourseMembers,
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
