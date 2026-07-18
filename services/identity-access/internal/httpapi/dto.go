package httpapi

import "github.com/cricketdrs/services/identity-access/internal/domain"

type organizationResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toOrganizationResponse(org domain.Organization) organizationResponse {
	return organizationResponse{ID: string(org.ID), Name: org.Name}
}

type userResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
}

func toUserResponse(u domain.User) userResponse {
	return userResponse{
		ID:             string(u.ID),
		OrganizationID: string(u.OrganizationID),
		Email:          u.Email,
		Role:           string(u.Role),
	}
}

type createOrganizationRequest struct {
	Name          string `json:"name"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

type createOrganizationResponse struct {
	Organization organizationResponse `json:"organization"`
	Admin        userResponse         `json:"admin"`
}

type inviteUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type assignRoleRequest struct {
	Role string `json:"role"`
}

type loginRequest struct {
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Password       string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type consentRequest struct {
	ConsentType    string  `json:"consent_type"`
	Grant          bool    `json:"grant"`
	IsMinor        bool    `json:"is_minor,omitempty"`
	GuardianUserID *string `json:"guardian_user_id,omitempty"`
}

type consentResponse struct {
	UserID         string          `json:"user_id"`
	OrganizationID string          `json:"organization_id"`
	Grants         map[string]bool `json:"grants"`
	IsMinor        bool            `json:"is_minor"`
	GuardianUserID *string         `json:"guardian_user_id,omitempty"`
}

func toConsentResponse(rec domain.ConsentRecord) consentResponse {
	grants := make(map[string]bool, len(rec.Grants))
	for k, v := range rec.Grants {
		grants[string(k)] = v
	}
	var guardian *string
	if rec.GuardianUserID != nil {
		s := string(*rec.GuardianUserID)
		guardian = &s
	}
	return consentResponse{
		UserID:         string(rec.UserID),
		OrganizationID: string(rec.OrganizationID),
		Grants:         grants,
		IsMinor:        rec.IsMinor,
		GuardianUserID: guardian,
	}
}
