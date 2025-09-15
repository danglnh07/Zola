package db

import (
	"database/sql"

	"gorm.io/gorm"
)

type Role string

type OauthProvider string

type ContentType string

type MessageStatus string

const (
	User  Role = "user"
	Admin Role = "admin"

	Google OauthProvider = "google"

	Text  ContentType = "text"
	Image ContentType = "image"
	Video ContentType = "video"
	File  ContentType = "file"

	Sent     MessageStatus = "sent"
	Received MessageStatus = "received"
	Read     MessageStatus = "read"
)

type Account struct {
	gorm.Model
	Username         string         `json:"username" gorm:"unique"`
	Email            string         `json:"email" gorm:"unique"`
	Password         sql.NullString `json:"password"`
	Role             Role           `json:"role"`
	OauthProvider    sql.NullString `json:"oauth_provider"`
	OauthProviderID  sql.NullString `json:"oauth_provider_id"`
	TokenVersion     uint           `json:"token_version"`
	Friends          []Account      `json:"friends" gorm:"many2many:friends;"`
	MessagesSent     []Message      `json:"messages_sent" gorm:"foreignKey:SenderID"`
	MessagesReceived []Message      `json:"messages_received" gorm:"foreignKey:ReceiverID"`
}

type Message struct {
	gorm.Model
	SenderID   uint          `json:"sender_id"`
	ReceiverID uint          `json:"receiver_id"`
	Content    string        `json:"content"` // If the content is image, video or file, content would be the path
	Type       ContentType   `json:"type"`
	Status     MessageStatus `json:"status"`
}
