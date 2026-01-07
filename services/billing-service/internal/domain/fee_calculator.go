package domain

// FeeSchedule represents the fee structure for a seller
// This is passed from seller-service when calculating fees
type FeeSchedule struct {
	StorageFeePerCubicFtPerDay float64
	PickFeePerUnit             float64
	PackFeePerOrder            float64
	ReceivingFeePerUnit        float64
	ShippingMarkupPercent      float64
	ReturnProcessingFee        float64
	GiftWrapFee                float64
	HazmatHandlingFee          float64
	OversizedItemFee           float64
	ColdChainFeePerUnit        float64
	FragileHandlingFee         float64
	VolumeDiscounts            []VolumeDiscount
}

// VolumeDiscount represents a volume-based discount tier
type VolumeDiscount struct {
	MinUnits        int
	MaxUnits        int
	DiscountPercent float64
}

// FeeCalculator calculates fees based on a fee schedule
type FeeCalculator struct {
	schedule *FeeSchedule
}

// NewFeeCalculator creates a new fee calculator
func NewFeeCalculator(schedule *FeeSchedule) *FeeCalculator {
	return &FeeCalculator{schedule: schedule}
}

// CalculateStorageFee calculates daily storage fee
func (c *FeeCalculator) CalculateStorageFee(cubicFeet float64) float64 {
	return cubicFeet * c.schedule.StorageFeePerCubicFtPerDay
}

// CalculatePickFee calculates picking fee for units
func (c *FeeCalculator) CalculatePickFee(units int) float64 {
	fee := float64(units) * c.schedule.PickFeePerUnit
	return c.applyVolumeDiscount(fee, units)
}

// CalculatePackFee calculates packing fee for orders
func (c *FeeCalculator) CalculatePackFee(orders int) float64 {
	return float64(orders) * c.schedule.PackFeePerOrder
}

// CalculateReceivingFee calculates receiving fee for units
func (c *FeeCalculator) CalculateReceivingFee(units int) float64 {
	return float64(units) * c.schedule.ReceivingFeePerUnit
}

// CalculateShippingFee calculates shipping fee with markup
func (c *FeeCalculator) CalculateShippingFee(baseCost float64) float64 {
	markup := baseCost * (c.schedule.ShippingMarkupPercent / 100)
	return baseCost + markup
}

// CalculateReturnProcessingFee calculates return processing fee
func (c *FeeCalculator) CalculateReturnProcessingFee(returns int) float64 {
	return float64(returns) * c.schedule.ReturnProcessingFee
}

// CalculateGiftWrapFee calculates gift wrap fee
func (c *FeeCalculator) CalculateGiftWrapFee(items int) float64 {
	return float64(items) * c.schedule.GiftWrapFee
}

// CalculateHazmatFee calculates hazmat handling fee
func (c *FeeCalculator) CalculateHazmatFee(units int) float64 {
	return float64(units) * c.schedule.HazmatHandlingFee
}

// CalculateOversizedFee calculates oversized item fee
func (c *FeeCalculator) CalculateOversizedFee(items int) float64 {
	return float64(items) * c.schedule.OversizedItemFee
}

// CalculateColdChainFee calculates cold chain handling fee
func (c *FeeCalculator) CalculateColdChainFee(units int) float64 {
	return float64(units) * c.schedule.ColdChainFeePerUnit
}

// CalculateFragileFee calculates fragile item handling fee
func (c *FeeCalculator) CalculateFragileFee(items int) float64 {
	return float64(items) * c.schedule.FragileHandlingFee
}

// applyVolumeDiscount applies volume discount if applicable
func (c *FeeCalculator) applyVolumeDiscount(fee float64, units int) float64 {
	for _, discount := range c.schedule.VolumeDiscounts {
		if units >= discount.MinUnits && (discount.MaxUnits == 0 || units <= discount.MaxUnits) {
			return fee * (1 - discount.DiscountPercent/100)
		}
	}
	return fee
}

// FeeCalculationRequest represents a request to calculate fees
type FeeCalculationRequest struct {
	TenantID   string
	SellerID   string
	FacilityID string

	// Activity metrics
	StorageCubicFeet  float64
	UnitsPicked       int
	OrdersPacked      int
	UnitsReceived     int
	ShippingBaseCost  float64
	ReturnsProcessed  int
	GiftWrapItems     int
	HazmatUnits       int
	OversizedItems    int
	ColdChainUnits    int
	FragileItems      int
}

// FeeCalculationResult represents the result of fee calculation
type FeeCalculationResult struct {
	StorageFee          float64 `json:"storageFee"`
	PickFee             float64 `json:"pickFee"`
	PackFee             float64 `json:"packFee"`
	ReceivingFee        float64 `json:"receivingFee"`
	ShippingFee         float64 `json:"shippingFee"`
	ReturnProcessingFee float64 `json:"returnProcessingFee"`
	GiftWrapFee         float64 `json:"giftWrapFee"`
	HazmatFee           float64 `json:"hazmatFee"`
	OversizedFee        float64 `json:"oversizedFee"`
	ColdChainFee        float64 `json:"coldChainFee"`
	FragileFee          float64 `json:"fragileFee"`
	TotalFees           float64 `json:"totalFees"`
}

// CalculateAllFees calculates all fees for a request
func (c *FeeCalculator) CalculateAllFees(req FeeCalculationRequest) *FeeCalculationResult {
	result := &FeeCalculationResult{
		StorageFee:          c.CalculateStorageFee(req.StorageCubicFeet),
		PickFee:             c.CalculatePickFee(req.UnitsPicked),
		PackFee:             c.CalculatePackFee(req.OrdersPacked),
		ReceivingFee:        c.CalculateReceivingFee(req.UnitsReceived),
		ShippingFee:         c.CalculateShippingFee(req.ShippingBaseCost),
		ReturnProcessingFee: c.CalculateReturnProcessingFee(req.ReturnsProcessed),
		GiftWrapFee:         c.CalculateGiftWrapFee(req.GiftWrapItems),
		HazmatFee:           c.CalculateHazmatFee(req.HazmatUnits),
		OversizedFee:        c.CalculateOversizedFee(req.OversizedItems),
		ColdChainFee:        c.CalculateColdChainFee(req.ColdChainUnits),
		FragileFee:          c.CalculateFragileFee(req.FragileItems),
	}

	result.TotalFees = result.StorageFee + result.PickFee + result.PackFee +
		result.ReceivingFee + result.ShippingFee + result.ReturnProcessingFee +
		result.GiftWrapFee + result.HazmatFee + result.OversizedFee +
		result.ColdChainFee + result.FragileFee

	return result
}

// CreateActivitiesFromResult creates billable activities from calculation result
func CreateActivitiesFromResult(
	tenantID, sellerID, facilityID string,
	schedule *FeeSchedule,
	req FeeCalculationRequest,
	referenceType, referenceID string,
) []*BillableActivity {
	var activities []*BillableActivity

	if req.StorageCubicFeet > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeStorage,
			"Daily storage fee",
			req.StorageCubicFeet,
			schedule.StorageFeePerCubicFtPerDay,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.UnitsPicked > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypePick,
			"Unit picking fee",
			float64(req.UnitsPicked),
			schedule.PickFeePerUnit,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.OrdersPacked > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypePack,
			"Order packing fee",
			float64(req.OrdersPacked),
			schedule.PackFeePerOrder,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.UnitsReceived > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeReceiving,
			"Receiving fee",
			float64(req.UnitsReceived),
			schedule.ReceivingFeePerUnit,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.ShippingBaseCost > 0 {
		shippingFee := req.ShippingBaseCost * (1 + schedule.ShippingMarkupPercent/100)
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeShipping,
			"Shipping fee",
			1,
			shippingFee,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.ReturnsProcessed > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeReturnProcessing,
			"Return processing fee",
			float64(req.ReturnsProcessed),
			schedule.ReturnProcessingFee,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.GiftWrapItems > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeGiftWrap,
			"Gift wrap fee",
			float64(req.GiftWrapItems),
			schedule.GiftWrapFee,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.HazmatUnits > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeHazmat,
			"Hazmat handling fee",
			float64(req.HazmatUnits),
			schedule.HazmatHandlingFee,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.OversizedItems > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeOversized,
			"Oversized item fee",
			float64(req.OversizedItems),
			schedule.OversizedItemFee,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.ColdChainUnits > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeColdChain,
			"Cold chain handling fee",
			float64(req.ColdChainUnits),
			schedule.ColdChainFeePerUnit,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	if req.FragileItems > 0 {
		activity, _ := NewBillableActivity(
			tenantID, sellerID, facilityID,
			ActivityTypeFragile,
			"Fragile item handling fee",
			float64(req.FragileItems),
			schedule.FragileHandlingFee,
			referenceType, referenceID,
		)
		if activity != nil {
			activities = append(activities, activity)
		}
	}

	return activities
}
