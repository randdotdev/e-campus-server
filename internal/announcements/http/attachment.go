package http

import (
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/announcements"
)

// AttachmentResponse is the transport shape of a file attached to a post or
// an activity — no inode id on the wire; bytes are reached through the
// download redirect. Both nouns produce it, so it belongs to neither noun
// file.
type AttachmentResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	FileType    string    `json:"file_type"`
	OrderIndex  int       `json:"order_index"`
}

// AddAttachmentRequest references a file in the actor's own drive; the
// stored size is the size of record, so the request no longer declares it.
type AddAttachmentRequest struct {
	UploadID    uuid.UUID `json:"upload_id" binding:"required"`
	DisplayName string    `json:"display_name" binding:"omitempty,max=255"`
	FileType    string    `json:"file_type" binding:"required,oneof=image document voice video"`
	OrderIndex  int       `json:"order_index"`
}

func postAttachmentResponse(a *announcements.PostAttachment) AttachmentResponse {
	return AttachmentResponse{ID: a.ID, DisplayName: a.DisplayName, FileType: a.FileType, OrderIndex: a.OrderIndex}
}

func postAttachmentResponses(attachments []announcements.PostAttachment) []AttachmentResponse {
	if len(attachments) == 0 {
		return nil
	}
	result := make([]AttachmentResponse, len(attachments))
	for i := range attachments {
		result[i] = postAttachmentResponse(&attachments[i])
	}
	return result
}

func activityAttachmentResponses(attachments []announcements.ActivityAttachment) []AttachmentResponse {
	if len(attachments) == 0 {
		return nil
	}
	result := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		result[i] = AttachmentResponse{ID: a.ID, DisplayName: a.DisplayName, FileType: a.FileType, OrderIndex: a.OrderIndex}
	}
	return result
}
