package models

// User represents an application user.
type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string `gorm:"size:128;not null" json:"-"`

	// Relationships (not included in JSON serialization to match Flask Marshmallow behavior)
	Stores []Store `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// Store represents a store owned by a user.
type Store struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"size:64;not null" json:"name"`
	UserID uint   `gorm:"index;not null" json:"user_id"`

	// Relationships (not included in JSON serialization to match Flask Marshmallow behavior)
	Tags  []Tag  `gorm:"foreignKey:StoreID;constraint:OnDelete:CASCADE" json:"-"`
	Items []Item `gorm:"foreignKey:StoreID;constraint:OnDelete:CASCADE" json:"-"`
}

// Tag represents a tag associated with a store.
type Tag struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"size:64;not null" json:"name"`
	StoreID uint   `gorm:"index;not null" json:"store_id"`
}

// Item represents an item in a store.
type Item struct {
	ID      uint    `gorm:"primaryKey" json:"id"`
	Name    string  `gorm:"size:64;not null" json:"name"`
	Price   float64 `gorm:"not null" json:"price"`
	StoreID uint    `gorm:"index;not null" json:"store_id"`
}
