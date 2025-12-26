package application

import "time"

// ShipmentDTO represents a shipment in responses
type ShipmentDTO struct {
	ShipmentID        string             `json:"shipmentId"`
	OrderID           string             `json:"orderId"`
	PackageID         string             `json:"packageId"`
	WaveID            string             `json:"waveId,omitempty"`
	Status            string             `json:"status"`
	Carrier           CarrierDTO         `json:"carrier"`
	Label             *ShippingLabelDTO  `json:"label,omitempty"`
	Manifest          *ManifestDTO       `json:"manifest,omitempty"`
	Package           PackageInfoDTO     `json:"package"`
	Recipient         AddressDTO         `json:"recipient"`
	Shipper           AddressDTO         `json:"shipper"`
	ServiceType       string             `json:"serviceType"`
	EstimatedDelivery *time.Time         `json:"estimatedDelivery,omitempty"`
	ActualDelivery    *time.Time         `json:"actualDelivery,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
	LabeledAt         *time.Time         `json:"labeledAt,omitempty"`
	ManifestedAt      *time.Time         `json:"manifestedAt,omitempty"`
	ShippedAt         *time.Time         `json:"shippedAt,omitempty"`
}

// CarrierDTO represents a shipping carrier
type CarrierDTO struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	AccountID   string `json:"accountId"`
	ServiceType string `json:"serviceType"`
}

// ShippingLabelDTO represents a shipping label
type ShippingLabelDTO struct {
	TrackingNumber string    `json:"trackingNumber"`
	LabelFormat    string    `json:"labelFormat"`
	LabelData      string    `json:"labelData"`
	LabelURL       string    `json:"labelUrl,omitempty"`
	GeneratedAt    time.Time `json:"generatedAt"`
}

// ManifestDTO represents a shipping manifest
type ManifestDTO struct {
	ManifestID    string    `json:"manifestId"`
	CarrierCode   string    `json:"carrierCode"`
	ShipmentCount int       `json:"shipmentCount"`
	GeneratedAt   time.Time `json:"generatedAt"`
}

// PackageInfoDTO represents package information
type PackageInfoDTO struct {
	PackageID   string        `json:"packageId"`
	Weight      float64       `json:"weight"`
	Dimensions  DimensionsDTO `json:"dimensions"`
	PackageType string        `json:"packageType"`
}

// DimensionsDTO represents package dimensions
type DimensionsDTO struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// AddressDTO represents a shipping address
type AddressDTO struct {
	Name       string `json:"name"`
	Company    string `json:"company,omitempty"`
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
}
