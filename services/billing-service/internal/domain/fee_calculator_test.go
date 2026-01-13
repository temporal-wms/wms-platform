package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeeCalculatorBasicCalculations(t *testing.T) {
	schedule := &FeeSchedule{
		StorageFeePerCubicFtPerDay: 0.10,
		PickFeePerUnit:             0.50,
		PackFeePerOrder:            1.25,
		ReceivingFeePerUnit:        0.30,
		ShippingMarkupPercent:      10,
		ReturnProcessingFee:        2.00,
		GiftWrapFee:                0.75,
		HazmatHandlingFee:          3.00,
		OversizedItemFee:           4.00,
		ColdChainFeePerUnit:        1.50,
		FragileHandlingFee:         0.60,
		VolumeDiscounts: []VolumeDiscount{
			{MinUnits: 10, MaxUnits: 0, DiscountPercent: 10},
		},
	}

	calculator := NewFeeCalculator(schedule)

	assert.Equal(t, 5.0, calculator.CalculateStorageFee(50))
	assert.Equal(t, 2.5, calculator.CalculatePickFee(5))
	assert.Equal(t, 4.5, calculator.CalculatePickFee(10))
	assert.Equal(t, 3.75, calculator.CalculatePackFee(3))
	assert.Equal(t, 3.0, calculator.CalculateReceivingFee(10))
	assert.Equal(t, 11.0, calculator.CalculateShippingFee(10))
	assert.Equal(t, 6.0, calculator.CalculateReturnProcessingFee(3))
	assert.Equal(t, 3.0, calculator.CalculateGiftWrapFee(4))
	assert.Equal(t, 6.0, calculator.CalculateHazmatFee(2))
	assert.Equal(t, 8.0, calculator.CalculateOversizedFee(2))
	assert.Equal(t, 6.0, calculator.CalculateColdChainFee(4))
	assert.Equal(t, 3.0, calculator.CalculateFragileFee(5))
}

func TestFeeCalculatorCalculateAllFees(t *testing.T) {
	schedule := &FeeSchedule{
		StorageFeePerCubicFtPerDay: 0.10,
		PickFeePerUnit:             0.50,
		PackFeePerOrder:            1.00,
		ReceivingFeePerUnit:        0.25,
		ShippingMarkupPercent:      20,
		ReturnProcessingFee:        2.00,
		GiftWrapFee:                0.50,
		HazmatHandlingFee:          3.00,
		OversizedItemFee:           4.00,
		ColdChainFeePerUnit:        1.50,
		FragileHandlingFee:         0.75,
	}

	calculator := NewFeeCalculator(schedule)
	req := FeeCalculationRequest{
		StorageCubicFeet: 100,
		UnitsPicked:      10,
		OrdersPacked:     4,
		UnitsReceived:    8,
		ShippingBaseCost: 25,
		ReturnsProcessed: 2,
		GiftWrapItems:    5,
		HazmatUnits:      1,
		OversizedItems:   2,
		ColdChainUnits:   3,
		FragileItems:     4,
	}

	result := calculator.CalculateAllFees(req)
	require.NotNil(t, result)

	expectedTotal := result.StorageFee + result.PickFee + result.PackFee +
		result.ReceivingFee + result.ShippingFee + result.ReturnProcessingFee +
		result.GiftWrapFee + result.HazmatFee + result.OversizedFee +
		result.ColdChainFee + result.FragileFee

	assert.Equal(t, expectedTotal, result.TotalFees)
	assert.Equal(t, 10.0, result.StorageFee)
	assert.Equal(t, 5.0, result.PickFee)
	assert.Equal(t, 4.0, result.PackFee)
	assert.Equal(t, 2.0, result.ReceivingFee)
	assert.Equal(t, 30.0, result.ShippingFee)
}

func TestCreateActivitiesFromResult(t *testing.T) {
	schedule := &FeeSchedule{
		StorageFeePerCubicFtPerDay: 0.10,
		PickFeePerUnit:             0.50,
		PackFeePerOrder:            1.00,
		ReceivingFeePerUnit:        0.25,
		ShippingMarkupPercent:      20,
		ReturnProcessingFee:        2.00,
		GiftWrapFee:                0.50,
		HazmatHandlingFee:          3.00,
		OversizedItemFee:           4.00,
		ColdChainFeePerUnit:        1.50,
		FragileHandlingFee:         0.75,
	}

	req := FeeCalculationRequest{
		StorageCubicFeet: 100,
		UnitsPicked:      10,
		OrdersPacked:     4,
		UnitsReceived:    8,
		ShippingBaseCost: 25,
		ReturnsProcessed: 2,
		GiftWrapItems:    5,
		HazmatUnits:      1,
		OversizedItems:   2,
		ColdChainUnits:   3,
		FragileItems:     4,
	}

	activities := CreateActivitiesFromResult(
		"TNT-001", "SLR-001", "FAC-001",
		schedule,
		req,
		"order",
		"ORD-123",
	)

	require.Len(t, activities, 11)

	amounts := map[ActivityType]float64{}
	for _, activity := range activities {
		require.NotNil(t, activity)
		assert.Equal(t, "TNT-001", activity.TenantID)
		assert.Equal(t, "SLR-001", activity.SellerID)
		assert.Equal(t, "FAC-001", activity.FacilityID)
		assert.False(t, activity.Invoiced)
		amounts[activity.Type] = activity.Amount
	}

	assert.Equal(t, 10.0, amounts[ActivityTypeStorage])
	assert.Equal(t, 5.0, amounts[ActivityTypePick])
	assert.Equal(t, 4.0, amounts[ActivityTypePack])
	assert.Equal(t, 2.0, amounts[ActivityTypeReceiving])
	assert.Equal(t, 30.0, amounts[ActivityTypeShipping])
	assert.Equal(t, 4.0, amounts[ActivityTypeReturnProcessing])
	assert.Equal(t, 2.5, amounts[ActivityTypeGiftWrap])
	assert.Equal(t, 3.0, amounts[ActivityTypeHazmat])
	assert.Equal(t, 8.0, amounts[ActivityTypeOversized])
	assert.Equal(t, 4.5, amounts[ActivityTypeColdChain])
	assert.Equal(t, 3.0, amounts[ActivityTypeFragile])
}

func TestCreateActivitiesFromResultEmpty(t *testing.T) {
	schedule := &FeeSchedule{
		StorageFeePerCubicFtPerDay: 0.10,
		PickFeePerUnit:             0.50,
		PackFeePerOrder:            1.00,
		ReceivingFeePerUnit:        0.25,
		ShippingMarkupPercent:      20,
		ReturnProcessingFee:        2.00,
		GiftWrapFee:                0.50,
		HazmatHandlingFee:          3.00,
		OversizedItemFee:           4.00,
		ColdChainFeePerUnit:        1.50,
		FragileHandlingFee:         0.75,
	}

	activities := CreateActivitiesFromResult(
		"TNT-001", "SLR-001", "FAC-001",
		schedule,
		FeeCalculationRequest{},
		"order",
		"ORD-123",
	)

	assert.Empty(t, activities)
}
