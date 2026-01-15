package domain

import (
	"fmt"
	"time"
)

// CostLayer represents a layer of inventory with a specific cost
// Used for FIFO/LIFO valuation methods to track inventory costs
type CostLayer struct {
	LayerID     string    `bson:"layerId" json:"layerId"`
	Quantity    int       `bson:"quantity" json:"quantity"`
	UnitCost    Money     `bson:"unitCost" json:"unitCost"`
	ReceivedAt  time.Time `bson:"receivedAt" json:"receivedAt"`
	ReferenceID string    `bson:"referenceId" json:"referenceId"` // PO ID or receiving reference
}

// NewCostLayer creates a new cost layer
func NewCostLayer(quantity int, unitCost Money, referenceID string) CostLayer {
	return CostLayer{
		LayerID:     generateLayerID(),
		Quantity:    quantity,
		UnitCost:    unitCost,
		ReceivedAt:  time.Now().UTC(),
		ReferenceID: referenceID,
	}
}

// TotalCost returns the total cost of this layer (quantity * unitCost)
func (cl CostLayer) TotalCost() (Money, error) {
	return cl.UnitCost.Multiply(cl.Quantity)
}

// IsEmpty returns true if the layer has zero quantity
func (cl CostLayer) IsEmpty() bool {
	return cl.Quantity == 0
}

// Consume removes quantity from this layer and returns the cost consumed
// Returns the updated layer and the cost consumed
func (cl CostLayer) Consume(qty int) (CostLayer, Money, error) {
	if qty > cl.Quantity {
		return cl, Money{}, fmt.Errorf("cannot consume %d units from layer with %d units", qty, cl.Quantity)
	}

	costConsumed, err := cl.UnitCost.Multiply(qty)
	if err != nil {
		return cl, Money{}, err
	}

	updatedLayer := cl
	updatedLayer.Quantity -= qty

	return updatedLayer, costConsumed, nil
}

// generateLayerID generates a unique layer ID
func generateLayerID() string {
	timestamp := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("LAYER-%s", timestamp)
}

// CostLayers is a collection of cost layers with helper methods
type CostLayers []CostLayer

// TotalQuantity returns the total quantity across all layers
func (cls CostLayers) TotalQuantity() int {
	total := 0
	for _, layer := range cls {
		total += layer.Quantity
	}
	return total
}

// TotalValue returns the total value across all layers
func (cls CostLayers) TotalValue() (Money, error) {
	if len(cls) == 0 {
		return ZeroMoney("USD"), nil
	}

	total := ZeroMoney(cls[0].UnitCost.Currency())
	for _, layer := range cls {
		layerCost, err := layer.TotalCost()
		if err != nil {
			return Money{}, err
		}

		total, err = total.Add(layerCost)
		if err != nil {
			return Money{}, err
		}
	}

	return total, nil
}

// AverageUnitCost calculates the weighted average unit cost
func (cls CostLayers) AverageUnitCost() (Money, error) {
	totalQty := cls.TotalQuantity()
	if totalQty == 0 {
		return ZeroMoney("USD"), nil
	}

	totalValue, err := cls.TotalValue()
	if err != nil {
		return Money{}, err
	}

	return totalValue.Divide(totalQty)
}

// RemoveEmptyLayers filters out layers with zero quantity
func (cls CostLayers) RemoveEmptyLayers() CostLayers {
	result := make(CostLayers, 0)
	for _, layer := range cls {
		if !layer.IsEmpty() {
			result = append(result, layer)
		}
	}
	return result
}
