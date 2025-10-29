package domain

import "time"

type UserRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Phone    string `json:"phone"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type DriverRegisterRequest struct {
	ID            string            `db:"id" json:"id"`
	CreatedAt     time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time         `db:"updated_at" json:"updated_at"`
	LicenseNumber string            `db:"license_number" json:"license_number"`
	VehicleType   string            `db:"vehicle_type" json:"vehicle_type"`
	VehicleAttrs  VehicleAttributes `db:"vehicle_attrs" json:"vehicle_attrs"`
	Status        string            `db:"status" json:"status"`
	IsVerified    bool              `db:"is_verified" json:"is_verified"`
}

type VehicleAttributes struct {
	Color string `json:"color"`
	Model string `json:"model"`
	Year  int    `json:"year"`
}
