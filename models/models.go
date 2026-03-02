package models

// User represents an application user.
type User struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	Username     string  `gorm:"uniqueIndex;not null;size:64" json:"username"`
	PasswordHash string  `gorm:"not null;size:128" json:"-"`
	Stores       []Store `gorm:"constraint:OnDelete:CASCADE" json:"stores,omitempty"`
}

// Store represents a store owned by a user.
type Store struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"not null;size:64" json:"name"`
	UserID uint   `gorm:"index;not null" json:"user_id"`
	Tags   []Tag  `gorm:"constraint:OnDelete:CASCADE" json:"tags,omitempty"`
	Items  []Item `gorm:"constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// Tag represents a tag associated with a store.
type Tag struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"not null;size:64" json:"name"`
	StoreID uint   `gorm:"index;not null" json:"store_id"`
}

// Item represents an item in a store.
type Item struct {
	ID      uint    `gorm:"primaryKey" json:"id"`
	Name    string  `gorm:"not null;size:64" json:"name"`
	Price   float64 `gorm:"not null" json:"price"`
	StoreID uint    `gorm:"index;not null" json:"store_id"`
}
