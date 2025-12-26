package application

import "github.com/wms-platform/shipping-service/internal/domain"

// ToShipmentDTO converts a domain Shipment to ShipmentDTO
func ToShipmentDTO(shipment *domain.Shipment) *ShipmentDTO {
	if shipment == nil {
		return nil
	}

	dto := &ShipmentDTO{
		ShipmentID:        shipment.ShipmentID,
		OrderID:           shipment.OrderID,
		PackageID:         shipment.PackageID,
		WaveID:            shipment.WaveID,
		Status:            string(shipment.Status),
		Carrier:           ToCarrierDTO(shipment.Carrier),
		Package:           ToPackageInfoDTO(shipment.Package),
		Recipient:         ToAddressDTO(shipment.Recipient),
		Shipper:           ToAddressDTO(shipment.Shipper),
		ServiceType:       shipment.ServiceType,
		EstimatedDelivery: shipment.EstimatedDelivery,
		ActualDelivery:    shipment.ActualDelivery,
		CreatedAt:         shipment.CreatedAt,
		UpdatedAt:         shipment.UpdatedAt,
		LabeledAt:         shipment.LabeledAt,
		ManifestedAt:      shipment.ManifestedAt,
		ShippedAt:         shipment.ShippedAt,
	}

	if shipment.Label != nil {
		dto.Label = ToShippingLabelDTO(shipment.Label)
	}

	if shipment.Manifest != nil {
		dto.Manifest = ToManifestDTO(shipment.Manifest)
	}

	return dto
}

// ToCarrierDTO converts a domain Carrier to CarrierDTO
func ToCarrierDTO(carrier domain.Carrier) CarrierDTO {
	return CarrierDTO{
		Code:        carrier.Code,
		Name:        carrier.Name,
		AccountID:   carrier.AccountID,
		ServiceType: carrier.ServiceType,
	}
}

// ToShippingLabelDTO converts a domain ShippingLabel to ShippingLabelDTO
func ToShippingLabelDTO(label *domain.ShippingLabel) *ShippingLabelDTO {
	if label == nil {
		return nil
	}

	return &ShippingLabelDTO{
		TrackingNumber: label.TrackingNumber,
		LabelFormat:    label.LabelFormat,
		LabelData:      label.LabelData,
		LabelURL:       label.LabelURL,
		GeneratedAt:    label.GeneratedAt,
	}
}

// ToManifestDTO converts a domain Manifest to ManifestDTO
func ToManifestDTO(manifest *domain.Manifest) *ManifestDTO {
	if manifest == nil {
		return nil
	}

	return &ManifestDTO{
		ManifestID:    manifest.ManifestID,
		CarrierCode:   manifest.CarrierCode,
		ShipmentCount: manifest.ShipmentCount,
		GeneratedAt:   manifest.GeneratedAt,
	}
}

// ToPackageInfoDTO converts a domain PackageInfo to PackageInfoDTO
func ToPackageInfoDTO(pkg domain.PackageInfo) PackageInfoDTO {
	return PackageInfoDTO{
		PackageID:   pkg.PackageID,
		Weight:      pkg.Weight,
		Dimensions:  ToDimensionsDTO(pkg.Dimensions),
		PackageType: pkg.PackageType,
	}
}

// ToDimensionsDTO converts a domain Dimensions to DimensionsDTO
func ToDimensionsDTO(dimensions domain.Dimensions) DimensionsDTO {
	return DimensionsDTO{
		Length: dimensions.Length,
		Width:  dimensions.Width,
		Height: dimensions.Height,
	}
}

// ToAddressDTO converts a domain Address to AddressDTO
func ToAddressDTO(address domain.Address) AddressDTO {
	return AddressDTO{
		Name:       address.Name,
		Company:    address.Company,
		Street1:    address.Street1,
		Street2:    address.Street2,
		City:       address.City,
		State:      address.State,
		PostalCode: address.PostalCode,
		Country:    address.Country,
		Phone:      address.Phone,
		Email:      address.Email,
	}
}

// ToShipmentDTOs converts a slice of domain Shipments to ShipmentDTOs
func ToShipmentDTOs(shipments []*domain.Shipment) []ShipmentDTO {
	dtos := make([]ShipmentDTO, 0, len(shipments))
	for _, shipment := range shipments {
		if dto := ToShipmentDTO(shipment); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}
