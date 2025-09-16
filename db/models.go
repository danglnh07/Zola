package db

import (
	"database/sql"

	"gorm.io/gorm"
)

type Role string

type OauthProvider string

type ContentType string

type MessageStatus string

type FriendshipStatus string

type NotificationStatus string

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
	Seen     MessageStatus = "read"

	Pending  FriendshipStatus = "pending"
	Accepted FriendshipStatus = "accepted"
	Rejected FriendshipStatus = "rejected"

	Read   NotificationStatus = "read"
	Unread NotificationStatus = "unread"
)

type Account struct {
	gorm.Model
	Username               string         `json:"username" gorm:"unique;not null"`
	Email                  string         `json:"email" gorm:"unique;not null"`
	Password               sql.NullString `json:"password"`
	Role                   Role           `json:"role" gorm:"not null"`
	OauthProvider          sql.NullString `json:"oauth_provider"`
	OauthProviderID        sql.NullString `json:"oauth_provider_id"`
	TokenVersion           uint           `json:"token_version" gorm:"not null"`
	FriendRequestsSent     []Friendship   `gorm:"foreignKey:RequesterID"`
	FriendRequestsReceived []Friendship   `gorm:"foreignKey:AddresseeID"`
	MessagesSent           []Message      `json:"messages_sent" gorm:"foreignKey:SenderID"`
	MessagesReceived       []Message      `json:"messages_received" gorm:"foreignKey:ReceiverID"`
	NotificationSent       []Notification `json:"notification_sent" gorm:"foreignKey:SourceID"`
	NotificationReceived   []Notification `json:"notification_received" gorm:"foreignKey:DestID"`
}

type Friendship struct {
	gorm.Model
	RequesterID uint             `json:"requester_id" gorm:"not null"`
	Requester   Account          `json:"requester" gorm:"foreignKey:RequesterID;not null"`
	AddresseeID uint             `json:"addressee_id" gorm:"not null"`
	Addressee   Account          `json:"addressee" gorm:"foreignKey:addressee_id;not null"`
	Status      FriendshipStatus `json:"status"`
}

type Message struct {
	gorm.Model
	SenderID   uint          `json:"sender_id"`
	ReceiverID uint          `json:"receiver_id"`
	Content    string        `json:"content"` // If the content is image, video or file, content would be the path
	Type       ContentType   `json:"type"`
	Status     MessageStatus `json:"status"`
}

type Notification struct {
	gorm.Model
	SourceID uint               `json:"source_id"` // The source of where the notification come from
	DestID   uint               `json:"dest_id"`   // The destination of the notification, where it suppose to be sent to
	Content  string             `json:"content"`
	Status   NotificationStatus `json:"status"`
}
